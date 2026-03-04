package client

import (
	"bufio"
	"fmt"
	"io"
	"net"
	"net/http"
	"sync"

	"golang.org/x/sync/errgroup"
)

// roundTripH1 performs an HTTP/1.1 request on conn with controlled header order.
// The response is parsed using the standard library's http.ReadResponse.
func (t *transport) roundTripH1(conn net.Conn, req *http.Request) (*http.Response, error) {
	bw := bufio.NewWriter(conn)

	path := req.URL.RequestURI()
	if _, err := fmt.Fprintf(bw, "%s %s HTTP/1.1\r\n", req.Method, path); err != nil {
		return nil, fmt.Errorf("writing request line: %w", err)
	}

	host := req.Host
	if host == "" {
		host = req.URL.Host
	}

	if _, err := fmt.Fprintf(bw, "Host: %s\r\n", host); err != nil {
		return nil, fmt.Errorf("writing host header: %w", err)
	}

	written := map[string]bool{"Host": true}
	order := headerOrder(req)

	for _, key := range order {
		if written[key] || isMagicKey(key) {
			continue
		}

		for _, val := range req.Header.Values(key) {
			if _, err := fmt.Fprintf(bw, "%s: %s\r\n", key, val); err != nil {
				return nil, fmt.Errorf("writing header %s: %w", key, err)
			}
		}

		written[key] = true
	}

	for key := range req.Header {
		if written[key] || isMagicKey(key) {
			continue
		}

		for _, val := range req.Header.Values(key) {
			if _, err := fmt.Fprintf(bw, "%s: %s\r\n", key, val); err != nil {
				return nil, fmt.Errorf("writing header %s: %w", key, err)
			}
		}
	}

	if _, err := bw.WriteString("\r\n"); err != nil {
		return nil, fmt.Errorf("writing header terminator: %w", err)
	}

	if req.Body != nil && req.Body != http.NoBody {
		if _, err := io.Copy(bw, req.Body); err != nil {
			return nil, fmt.Errorf("writing request body: %w", err)
		}
	}

	if err := bw.Flush(); err != nil {
		return nil, fmt.Errorf("flushing request: %w", err)
	}

	resp, err := http.ReadResponse(bufio.NewReader(conn), req)
	if err != nil {
		return nil, fmt.Errorf("reading response: %w", err)
	}

	resp.Body = &h1ResponseBody{
		ReadCloser: resp.Body,
		conn:       conn,
	}

	return resp, nil
}

// h1ResponseBody wraps a response body to close the underlying connection
// when the body is closed.
type h1ResponseBody struct {
	io.ReadCloser
	conn net.Conn
	once sync.Once
}

// Close closes the response body and the underlying connection exactly once.
func (b *h1ResponseBody) Close() error {
	var g errgroup.Group
	g.Go(func() error {
		return b.ReadCloser.Close()
	})
	g.Go(func() error {
		return b.conn.Close()
	})

	return g.Wait()
}
