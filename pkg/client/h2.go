package client

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"log/slog"
	"net"
	"net/http"
	"strconv"
	"strings"
	"sync"

	"golang.org/x/net/http2"
	"golang.org/x/net/http2/hpack"
)

const (
	h2DefaultWindow = 65535
	h2MaxFrameSize  = 16384
)

// h2ResponseBody wraps a pipe reader and the underlying connection,
// closing both when the body is closed.
type h2ResponseBody struct {
	io.ReadCloser
	conn net.Conn
	once sync.Once
}

// Close closes the pipe reader and the underlying connection exactly once.
func (b *h2ResponseBody) Close() error {
	var err error
	b.once.Do(func() {
		err = b.ReadCloser.Close()
		_ = b.conn.Close()
	})

	return err
}

// roundTripH2 performs an HTTP/2 request on conn with controlled SETTINGS,
// WINDOW_UPDATE, pseudo-header order, header order, and PRIORITY.
func (t *transport) roundTripH2(conn net.Conn, req *http.Request) (*http.Response, error) {
	bw := bufio.NewWriter(conn)
	if _, err := bw.WriteString("PRI * HTTP/2.0\r\n\r\nSM\r\n\r\n"); err != nil {
		return nil, fmt.Errorf("writing h2 preface: %w", err)
	}

	framer := http2.NewFramer(bw, conn)
	framer.ReadMetaHeaders = hpack.NewDecoder(4096, nil)
	framer.ReadMetaHeaders.SetEmitEnabled(false)

	if err := framer.WriteSettings(t.h2Profile.Settings...); err != nil {
		return nil, fmt.Errorf("writing settings: %w", err)
	}

	if t.h2Profile.ConnectionWindow > h2DefaultWindow {
		incr := t.h2Profile.ConnectionWindow - h2DefaultWindow
		if err := framer.WriteWindowUpdate(0, incr); err != nil {
			return nil, fmt.Errorf("writing window update: %w", err)
		}
	}

	headerBlock := t.encodeHeaders(req)

	endStream := req.Body == nil || req.Body == http.NoBody
	if err := t.writeH2Headers(framer, 1, headerBlock, endStream, t.h2Profile.Priority); err != nil {
		return nil, err
	}

	if !endStream {
		if err := t.writeH2Body(framer, 1, req.Body); err != nil {
			return nil, fmt.Errorf("writing request body: %w", err)
		}
	}

	if err := bw.Flush(); err != nil {
		return nil, fmt.Errorf("flushing writes: %w", err)
	}

	return t.readH2Response(framer, conn, req)
}

// encodeHeaders encodes request headers using HPACK with custom pseudo-header
// and regular header ordering.
func (t *transport) encodeHeaders(req *http.Request) []byte {
	var buf bytes.Buffer
	enc := hpack.NewEncoder(&buf)

	order := pseudoHeaderOrder(req, t.h2Profile.PseudoOrder)

	authority := req.Host
	if authority == "" {
		authority = req.URL.Host
	}

	pseudoValues := map[string]string{
		":method":    req.Method,
		":authority": authority,
		":scheme":    req.URL.Scheme,
		":path":      req.URL.RequestURI(),
	}

	for _, name := range order {
		if val, ok := pseudoValues[name]; ok {
			_ = enc.WriteField(hpack.HeaderField{
				Name:  name,
				Value: val,
			})
		}
	}

	written := make(map[string]bool)
	regularOrder := headerOrder(req)

	for _, key := range regularOrder {
		if written[key] || isMagicKey(key) {
			continue
		}

		for _, val := range req.Header.Values(key) {
			_ = enc.WriteField(hpack.HeaderField{
				Name:  strings.ToLower(key),
				Value: val,
			})
		}

		written[key] = true
	}

	for key := range req.Header {
		if written[key] || isMagicKey(key) {
			continue
		}

		for _, val := range req.Header.Values(key) {
			_ = enc.WriteField(hpack.HeaderField{
				Name:  strings.ToLower(key),
				Value: val,
			})
		}
	}

	return buf.Bytes()
}

// writeH2Headers writes a HEADERS frame on streamID, splitting the header
// block into CONTINUATION frames if it exceeds h2MaxFrameSize.
func (t *transport) writeH2Headers(framer *http2.Framer, streamID uint32, headerBlock []byte, endStream bool, priority http2.PriorityParam) error {
	first := true
	for len(headerBlock) > 0 || first {
		chunk := headerBlock
		if len(chunk) > h2MaxFrameSize {
			chunk = chunk[:h2MaxFrameSize]
		}

		headerBlock = headerBlock[len(chunk):]
		endHeaders := len(headerBlock) == 0

		if first {
			err := framer.WriteHeaders(http2.HeadersFrameParam{
				StreamID:      streamID,
				BlockFragment: chunk,
				EndStream:     endStream,
				EndHeaders:    endHeaders,
				Priority:      priority,
			})
			if err != nil {
				return fmt.Errorf("writing headers frame: %w", err)
			}

			first = false
		} else {
			if err := framer.WriteContinuation(streamID, endHeaders, chunk); err != nil {
				return fmt.Errorf("writing continuation frame: %w", err)
			}
		}
	}

	return nil
}

// writeH2Body writes the request body as DATA frames on streamID.
// The last frame has EndStream set to true.
func (t *transport) writeH2Body(framer *http2.Framer, streamID uint32, body io.Reader) error {
	buf := make([]byte, h2MaxFrameSize)
	for {
		n, err := body.Read(buf)
		if n > 0 {
			endStream := err == io.EOF
			if writeErr := framer.WriteData(streamID, endStream, buf[:n]); writeErr != nil {
				return fmt.Errorf("writing data frame: %w", writeErr)
			}

			if endStream {
				return nil
			}
		}

		if err == io.EOF {
			return framer.WriteData(streamID, true, nil)
		}

		if err != nil {
			return fmt.Errorf("reading request body: %w", err)
		}
	}
}

// readH2Response reads frames from the framer until a complete response is
// received. It handles SETTINGS, WINDOW_UPDATE, PING, GOAWAY, and RST_STREAM
// frames during the read loop.
func (t *transport) readH2Response(framer *http2.Framer, conn net.Conn, req *http.Request) (*http.Response, error) {
	for {
		frame, err := framer.ReadFrame()
		if err != nil {
			return nil, fmt.Errorf("reading frame: %w", err)
		}

		switch f := frame.(type) {
		case *http2.MetaHeadersFrame:
			resp := t.buildH2Response(f, req)

			if f.StreamEnded() {
				resp.Body = http.NoBody
				_ = conn.Close()

				return resp, nil
			}

			pr, pw := io.Pipe()
			resp.Body = &h2ResponseBody{
				ReadCloser: pr,
				conn:       conn,
			}

			go t.readH2Body(framer, pw)

			return resp, nil
		case *http2.SettingsFrame:
			if !f.IsAck() {
				if err := framer.WriteSettingsAck(); err != nil {
					return nil, fmt.Errorf("writing settings ack: %w", err)
				}
			}
		case *http2.WindowUpdateFrame:
			// Server updating our send window; ignored in single-flight mode.
		case *http2.PingFrame:
			if err := framer.WritePing(true, f.Data); err != nil {
				return nil, fmt.Errorf("writing ping ack: %w", err)
			}
		case *http2.GoAwayFrame:
			return nil, fmt.Errorf("server sent goaway: error code %v", f.ErrCode)
		case *http2.RSTStreamFrame:
			return nil, fmt.Errorf("server reset stream: error code %v", f.ErrCode)
		}
	}
}

// readH2Body reads DATA frames from the framer and writes them to the pipe
// writer. It sends WINDOW_UPDATE frames for flow control and handles control
// frames received during body reading.
func (t *transport) readH2Body(framer *http2.Framer, pw *io.PipeWriter) {
	defer func() { _ = pw.Close() }()

	for {
		frame, err := framer.ReadFrame()
		if err != nil {
			pw.CloseWithError(fmt.Errorf("reading body frame: %w", err))

			return
		}

		switch f := frame.(type) {
		case *http2.DataFrame:
			data := f.Data()
			if len(data) > 0 {
				if _, err := pw.Write(data); err != nil {
					return
				}

				// Best-effort WINDOW_UPDATE for flow control
				if err := framer.WriteWindowUpdate(f.Header().StreamID, uint32(len(data))); err != nil {
					slog.Warn("writing stream window update", "error", err)
				}

				if err := framer.WriteWindowUpdate(0, uint32(len(data))); err != nil {
					slog.Warn("writing global window update", "error", err)
				}
			}

			if f.StreamEnded() {
				return
			}
		case *http2.SettingsFrame:
			if !f.IsAck() {
				if err := framer.WriteSettingsAck(); err != nil {
					slog.Warn("writing settings ack", "error", err)
				}
			}
		case *http2.PingFrame:
			if err := framer.WritePing(true, f.Data); err != nil {
				slog.Warn("writing ping", "error", err)
			}
		case *http2.WindowUpdateFrame:
			// Ignored; we're done sending.
		case *http2.GoAwayFrame:
			if err := pw.CloseWithError(fmt.Errorf("server sent goaway: error code %v", f.ErrCode)); err != nil {
				slog.Warn("closing pipe writer with goaway error", "error", err)
			}

			return
		case *http2.RSTStreamFrame:
			if err := pw.CloseWithError(fmt.Errorf("server reset stream: error code %v", f.ErrCode)); err != nil {
				slog.Warn("closing pipe writer with reset stream error", "error", err)
			}

			return
		}
	}
}

// buildH2Response constructs an *http.Response from a MetaHeadersFrame.
func (t *transport) buildH2Response(mh *http2.MetaHeadersFrame, req *http.Request) *http.Response {
	resp := &http.Response{
		Proto:         "HTTP/2.0",
		ProtoMajor:    2,
		Header:        make(http.Header),
		Request:       req,
		ContentLength: -1,
	}

	for _, f := range mh.Fields {
		switch f.Name {
		case ":status":
			if statusCode, err := strconv.Atoi(f.Value); err != nil {
				slog.Warn("converting status code", "error", err)
			} else {
				resp.StatusCode = statusCode
				resp.Status = f.Value + " " + http.StatusText(statusCode)
			}
		default:
			resp.Header.Add(f.Name, f.Value)
		}
	}

	if cl := resp.Header.Get("Content-Length"); cl != "" {
		if contentLength, err := strconv.ParseInt(cl, 10, 64); err != nil {
			slog.Warn("converting content length", "error", err)
		} else {
			resp.ContentLength = contentLength
		}
	}

	return resp
}
