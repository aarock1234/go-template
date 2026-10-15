package template

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"time"

	http "github.com/aarock1234/fphttp/v2"
)

const (
	// requestTimeout bounds a request, including reading its response body.
	requestTimeout = 30 * time.Second

	// maxResponseSize bounds how much of a response body is decoded.
	maxResponseSize = 10 << 20
)

// do sends a request and returns the response when its status is 2xx. The
// caller must close the response body.
func (s *Service) do(ctx context.Context, method, url string, body io.Reader) (*http.Response, error) {
	req, err := http.NewRequestWithContext(ctx, method, url, body)
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}

	req.Header.Set("Accept", "application/json")
	req.Header.Set("Accept-Language", "en-US,en;q=0.9")

	resp, err := s.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("send request: %w", err)
	}

	if resp.StatusCode < 200 || resp.StatusCode > 299 {
		_ = resp.Body.Close()

		return nil, fmt.Errorf("unexpected status: %s", resp.Status)
	}

	return resp, nil
}

// getJSON fetches url and decodes its JSON response body into a T.
func (s *Service) getJSON[T any](ctx context.Context, url string) (*T, error) {
	ctx, cancel := context.WithTimeout(ctx, requestTimeout)
	defer cancel()

	resp, err := s.do(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	defer func() { _ = resp.Body.Close() }()

	var result T
	if err := json.NewDecoder(io.LimitReader(resp.Body, maxResponseSize)).Decode(&result); err != nil {
		return nil, fmt.Errorf("decode response: %w", err)
	}

	return &result, nil
}
