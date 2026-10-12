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

// newDecoderFunc returns a reader that decodes r.
type newDecoderFunc func(r io.Reader) (io.ReadCloser, error)

// decompress replaces resp.Body with a reader that decodes its single
// Content-Encoding, then removes the Content-Encoding and Content-Length
// headers and marks the response as uncompressed. Unknown or stacked
// encodings leave resp unchanged.
func decompress(resp *http.Response) {
	if resp.Uncompressed {
		return
	}

	newDecoder, ok := decoderFor(strings.ToLower(strings.TrimSpace(resp.Header.Get("Content-Encoding"))))
	if !ok {
		return
	}

	resp.Body = &decodingBody{
		body:       resp.Body,
		newDecoder: newDecoder,
	}
	resp.Header.Del("Content-Encoding")
	resp.Header.Del("Content-Length")
	resp.ContentLength = -1
	resp.Uncompressed = true
}

// decoderFor returns the decoder constructor for a Content-Encoding, or
// false if the encoding is not supported. Errors from the gzip and zlib
// readers are returned unwrapped, so an empty body reads as io.EOF.
func decoderFor(encoding string) (newDecoderFunc, bool) {
	switch encoding {
	case "gzip", "x-gzip":
		return func(r io.Reader) (io.ReadCloser, error) {
			return gzip.NewReader(r)
		}, true
	case "deflate":
		return zlib.NewReader, true
	case "br":
		return func(r io.Reader) (io.ReadCloser, error) {
			return io.NopCloser(brotli.NewReader(r)), nil
		}, true
	case "zstd":
		return newZstdDecoder, true
	default:
		return nil, false
	}
}

// newZstdDecoder returns a single-threaded zstd decoder, which suits
// streaming one response body.
func newZstdDecoder(r io.Reader) (io.ReadCloser, error) {
	decoder, err := zstd.NewReader(r, zstd.WithDecoderConcurrency(1))
	if err != nil {
		return nil, fmt.Errorf("create zstd decoder: %w", err)
	}

	return decoder.IOReadCloser(), nil
}

// decodingBody decodes a response body. The decoder is created on the
// first Read, so a body that is never read, such as an empty one, needs no
// valid encoding header, and decoding errors surface from Read.
type decodingBody struct {
	body       io.ReadCloser
	newDecoder newDecoderFunc
	decoder    io.ReadCloser
	err        error
}

// Read reads decoded bytes from the body.
func (b *decodingBody) Read(p []byte) (int, error) {
	if b.decoder == nil && b.err == nil {
		b.decoder, b.err = b.newDecoder(b.body)
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
