package client

import (
	"fmt"
	"io"
	"log/slog"
	"strings"

	ehttp "github.com/enetx/http"
	http "github.com/saucesteals/fhttp"
)

const maxRedirects = 10

const setCookieHeader = "Set-Cookie"

// Do sends an HTTP request and returns an HTTP response.
// Redirects are followed manually so that Set-Cookie headers from intermediate
// responses are captured and applied to subsequent requests.
func (c *Client) Do(req *http.Request) (*http.Response, error) {
	for range maxRedirects {
		c.setRequestCookies(req)

		resp, err := c.doRoundTrip(req)
		if err != nil {
			return nil, err
		}

		if err := c.storeResponseCookies(resp); err != nil {
			resp.Body.Close()
			return nil, err
		}

		method, ok := c.shouldRedirect(resp)
		if !ok {
			return resp, nil
		}

		resp.Body.Close()

		next, err := c.redirectRequest(req, resp, method)
		if err != nil {
			return nil, err
		}

		req = next.WithContext(req.Context())
	}

	return nil, fmt.Errorf("stopped after %d redirects", maxRedirects)
}

func (c *Client) doRoundTrip(req *http.Request) (*http.Response, error) {
	resp, err := c.http.Do(req)
	if err != nil {
		return nil, err
	}

	slog.Debug("round trip complete",
		slog.String("host", req.URL.Hostname()),
		slog.Int("status", resp.StatusCode),
	)

	return resp, nil
}

func (c *Client) setRequestCookies(req *http.Request) {
	if c.jar == nil {
		return
	}

	cookies := c.jar.Cookies(req.URL)
	if len(cookies) == 0 {
		return
	}

	var b strings.Builder
	for i, cookie := range cookies {
		if i > 0 {
			b.WriteString("; ")
		}
		b.WriteString(cookie.Name)
		b.WriteByte('=')
		b.WriteString(cookie.Value)
	}

	req.Header.Set("Cookie", b.String())
}

func (c *Client) storeResponseCookies(resp *http.Response) error {
	if c.jar == nil {
		return nil
	}

	cookies, err := c.extractCookies(resp)
	if err != nil {
		return fmt.Errorf("extracting cookies: %w", err)
	}

	c.jar.SetCookies(resp.Request.URL, cookies)

	return nil
}

// DefaultCookieExtractor parses standard Set-Cookie response headers.
func DefaultCookieExtractor(resp *http.Response) ([]*http.Cookie, error) {
	var cookies []*http.Cookie
	for _, raw := range resp.Header.Values(setCookieHeader) {
		parsed, err := ehttp.ParseSetCookie(raw)
		if err != nil {
			return nil, fmt.Errorf("parsing set-cookie: %w", err)
		}

		cookies = append(cookies, toNetHTTPCookie(parsed))
	}

	return cookies, nil
}

// shouldRedirect returns (method, true) if the response is a redirect that
// should be followed, or ("", false) otherwise.
func (c *Client) shouldRedirect(resp *http.Response) (method string, ok bool) {
	switch resp.StatusCode {
	case http.StatusMovedPermanently, http.StatusFound, http.StatusSeeOther:
		return http.MethodGet, true
	case http.StatusTemporaryRedirect, http.StatusPermanentRedirect:
		return resp.Request.Method, true
	default:
		return "", false
	}
}

func (c *Client) redirectRequest(origin *http.Request, resp *http.Response, method string) (*http.Request, error) {
	location, err := resp.Location()
	if err != nil {
		return nil, fmt.Errorf("getting redirect location: %w", err)
	}

	if location.Scheme == "" || location.Host == "" {
		location = resp.Request.URL.ResolveReference(location)
	}

	if method == "" {
		method = http.MethodGet
	}

	var body io.ReadCloser
	if method == origin.Method && origin.Body != nil && origin.GetBody != nil {
		var err error
		body, err = origin.GetBody()
		if err != nil {
			return nil, fmt.Errorf("getting request body for redirect: %w", err)
		}
	}

	next, err := http.NewRequestWithContext(origin.Context(), method, location.String(), body)
	if err != nil {
		if body != nil {
			body.Close()
		}

		return nil, fmt.Errorf("creating redirect request: %w", err)
	}

	for k, v := range origin.Header {
		if k != "Cookie" && k != "Host" {
			next.Header[k] = v
		}
	}

	return next, nil
}
