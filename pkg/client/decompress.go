package client

import (
	"compress/gzip"
	"compress/zlib"
	"errors"
	"fmt"
	"io"
	"strings"

	http "github.com/aarock1234/fphttp"
	"github.com/andybalholm/brotli"
	"github.com/klauspost/compress/zstd"
)

// decompress replaces resp.Body with a reader that decodes its single
// Content-Encoding, then removes the Content-Encoding and Content-Length
// headers and marks the response as uncompressed. Unknown or stacked
// encodings leave resp unchanged.
func decompress(resp *http.Response) {
	encoding := strings.ToLower(strings.TrimSpace(resp.Header.Get("Content-Encoding")))
	if resp.Uncompressed || !supportedEncoding(encoding) {
		return
	}

	resp.Body = &decodingBody{
		body:     resp.Body,
		encoding: encoding,
	}
	resp.Header.Del("Content-Encoding")
	resp.Header.Del("Content-Length")
	resp.ContentLength = -1
	resp.Uncompressed = true
}

// supportedEncoding reports whether decompress can decode encoding.
func supportedEncoding(encoding string) bool {
	switch encoding {
	case "gzip", "x-gzip", "deflate", "br", "zstd":
		return true
	default:
		return false
	}
}

// decodingBody decodes a response body. The decoder is created on the
// first Read, so a body that is never read, such as an empty one, needs no
// valid encoding header, and decoding errors surface from Read.
type decodingBody struct {
	body     io.ReadCloser
	encoding string
	decoder  io.ReadCloser
	err      error
}

// Read reads decoded bytes from the body.
func (b *decodingBody) Read(p []byte) (int, error) {
	if b.decoder == nil && b.err == nil {
		b.decoder, b.err = newDecoder(b.encoding, b.body)
	}

	if b.err != nil {
		return 0, b.err
	}

	return b.decoder.Read(p)
}

// Close releases the decoder and closes the underlying body.
func (b *decodingBody) Close() error {
	var decoderErr error
	if b.decoder != nil {
		decoderErr = b.decoder.Close()
	}

	return errors.Join(decoderErr, b.body.Close())
}

// newDecoder returns a reader that decodes r. Errors from the gzip and
// zlib readers are returned unwrapped, so an empty body reads as io.EOF.
func newDecoder(encoding string, r io.Reader) (io.ReadCloser, error) {
	switch encoding {
	case "gzip", "x-gzip":
		return gzip.NewReader(r)
	case "deflate":
		return zlib.NewReader(r)
	case "br":
		return io.NopCloser(brotli.NewReader(r)), nil
	case "zstd":
		decoder, err := zstd.NewReader(r, zstd.WithDecoderConcurrency(1))
		if err != nil {
			return nil, fmt.Errorf("create zstd decoder: %w", err)
		}

		return decoder.IOReadCloser(), nil
	default:
		return nil, fmt.Errorf("unsupported content encoding %q", encoding)
	}
}
