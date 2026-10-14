package client

import (
	"fmt"
	"log/slog"

	http "github.com/aarock1234/fphttp"
)

// cookiePolicy preserves the template's cookie serializer and extractor while
// leaving redirect and navigation ownership with the browser client.
type cookiePolicy struct {
	client *Client
}

func (p cookiePolicy) ApplyCookies(req *http.Request) {
	if req.Response != nil {
		// A redirected request is populated from the jar for its new URL.
		req.Header.Del("Cookie")
	}
	cookies := p.client.jar.Cookies(req.URL)
	if len(cookies) == 0 {
		return
	}

	header := cookieHeader(cookies)
	if existing := req.Header.Get("Cookie"); existing != "" {
		header = existing + "; " + header
	}
	req.Header.Set("Cookie", header)
}

func (p cookiePolicy) StoreCookies(resp *http.Response) error {
	slog.DebugContext(resp.Request.Context(), "round trip complete",
		slog.String("host", resp.Request.URL.Hostname()),
		slog.Int("status", resp.StatusCode),
	)
	cookies, err := p.client.extractCookies(resp)
	if err != nil {
		return fmt.Errorf("extract cookies: %w", err)
	}
	p.client.jar.SetCookies(resp.Request.URL, cookies)

	return nil
}
