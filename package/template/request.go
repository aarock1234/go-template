package template

import (
	"bytes"
	"compress/gzip"
	"compress/zlib"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/url"
	"strings"

	"github.com/andybalholm/brotli"
	"github.com/klauspost/compress/zstd"
	"github.com/saucesteals/fhttp"
)

const baseURL = "https://example.com"

var (
	defaultHeaders = http.Header{
		"Accept":          {"*/*"},
		"Accept-Language": {"en-US,en;q=0.9"},
		"Accept-Encoding": {"gzip, deflate, br"},
	}
)

func (t *Template) doRequest(ctx context.Context, method, path string, payload any, response any) error {
	return t.doRequestWithHeaders(ctx, method, path, payload, response, http.Header{})
}

func (t *Template) doRequestWithHeaders(ctx context.Context, method, path string, payload any, response any, headers http.Header) error {
	var (
		body        io.Reader
		contentType string
	)

	switch p := payload.(type) {
	case url.Values:
		body = bytes.NewReader([]byte(p.Encode()))
		contentType = "application/x-www-form-urlencoded"
	case nil:
		// no body
	default:
		data, err := json.Marshal(p)
		if err != nil {
			return fmt.Errorf("marshaling request: %w", err)
		}
		body = bytes.NewReader(data)
		contentType = "application/json"
	}

	fullURL := baseURL + path
	if _, err := url.Parse(fullURL); err != nil {
		return fmt.Errorf("invalid url %q: %w", fullURL, err)
	}

	req, err := http.NewRequestWithContext(ctx, method, fullURL, body)
	if err != nil {
		return fmt.Errorf("creating request: %w", err)
	}
	req.Header = defaultHeaders.Clone()

	if contentType != "" {
		req.Header.Set("content-type", contentType)
	}

	for k, v := range headers {
		req.Header.Set(k, strings.Join(v, ", "))
	}

	resp, err := t.client.Do(req)
	if err != nil {
		return fmt.Errorf("executing request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("unexpected status: %d", resp.StatusCode)
	}

	if !resp.Uncompressed {
		reader, err := decompressResponse(resp.Body, resp.Header.Get("content-encoding"))
		if err != nil {
			return fmt.Errorf("decompressing response: %w", err)
		}
		defer reader.Close()

		resp.Body = reader
	}

	if response != nil {
		if err := decodeResponse(resp.Body, response); err != nil {
			return fmt.Errorf("decoding response: %w", err)
		}
	}

	return nil
}

func decompressResponse(body io.Reader, encoding string) (io.ReadCloser, error) {
	if body == nil {
		return nil, fmt.Errorf("body is nil")
	}

	if encoding == "" {
		return io.NopCloser(body), nil
	}

	switch encoding {
	case "deflate":
		r, err := zlib.NewReader(body)
		if err != nil {
			return nil, fmt.Errorf("decompressing deflate: %w", err)
		}
		return r, nil
	case "gzip":
		r, err := gzip.NewReader(body)
		if err != nil {
			return nil, fmt.Errorf("decompressing gzip: %w", err)
		}
		return r, nil
	case "br":
		return io.NopCloser(brotli.NewReader(body)), nil
	case "zstd":
		decoder, err := zstd.NewReader(body)
		if err != nil {
			return nil, fmt.Errorf("decompressing zstd: %w", err)
		}
		return decoder.IOReadCloser(), nil
	default:
		slog.Warn("unknown encoding", "encoding", encoding)
		return io.NopCloser(body), nil
	}
}

func decodeResponse(body io.Reader, response any) error {
	switch v := response.(type) {
	case *string:
		data, err := io.ReadAll(body)
		if err != nil {
			return fmt.Errorf("reading string: %w", err)
		}
		*v = string(data)

	case *[]byte:
		data, err := io.ReadAll(body)
		if err != nil {
			return fmt.Errorf("reading bytes: %w", err)
		}
		*v = data

	default:
		if err := json.NewDecoder(body).Decode(response); err != nil {
			return fmt.Errorf("decoding json: %w", err)
		}
	}

	return nil
}
