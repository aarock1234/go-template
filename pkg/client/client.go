// Package client provides browser-aware HTTP requests, custom cookie storage,
// and proxy support.
package client

import (
	"errors"
	"net/url"
	"slices"
	"strings"

	http "github.com/aarock1234/fphttp"
	"github.com/aarock1234/fphttp/browser"
	utls "github.com/refraction-networking/utls"
)

const maxRedirects = 10

// ErrTooManyRedirects is returned when a request exceeds the redirect limit.
var ErrTooManyRedirects = errors.New("too many redirects")

// Client presents a selected browser identity and owns its cookie storage.
// It is safe for concurrent use. Use New to initialize it.
type Client struct {
	http           *browser.Client
	jar            *CookieJar
	defaultHeaders http.Header
	profile        *browser.Profile

	browser                http.Browser
	platform               http.Platform
	browserVersion         string
	fingerprint            *http.Fingerprint
	clientHelloID          *utls.ClientHelloID
	proxy                  *Proxy
	extractCookies         CookieExtractor
	defaultHeaderOverrides http.Header
	disableHeaders         bool
	disableDecompression   bool
	insecureSkipVerify     bool
}

// Do sends req with basic browser identity headers. It does not modify req.
// The caller owns the returned response body and must close it.
func (c *Client) Do(req *http.Request) (*http.Response, error) {
	return c.http.Do(req)
}

// DoWithOptions sends req with explicit browser context or another profile.
// One operation owns all redirects and eligible client-hint restarts.
// The caller owns the returned response body and must close it.
func (c *Client) DoWithOptions(req *http.Request, options browser.RequestOptions) (*http.Response, error) {
	return c.http.DoWithOptions(req, options)
}

// Descriptor returns an independently owned snapshot of the default identity.
// WithFingerprint and WithClientHelloID are manual overrides of its wire data.
func (c *Client) Descriptor() browser.Descriptor {
	return c.profile.Descriptor()
}

// SetCookies stores cookies in the jar as if u had set them.
func (c *Client) SetCookies(u *url.URL, cookies []*http.Cookie) {
	c.jar.SetCookies(u, cookies)
}

// SetCookieString parses cookieString and stores the cookies for domain and
// its subdomains, skipping names in exclude. A leading www. is removed.
func (c *Client) SetCookieString(domain, cookieString string, exclude ...string) error {
	cookies, err := ParseCookies(cookieString)
	if err != nil {
		return err
	}

	domain = strings.TrimPrefix(strings.TrimPrefix(domain, "."), "www.")
	kept := make([]*http.Cookie, 0, len(cookies))
	for _, cookie := range cookies {
		if slices.Contains(exclude, cookie.Name) {
			continue
		}

		cookie.Domain = domain
		kept = append(kept, cookie)
	}

	c.jar.SetCookies(&url.URL{
		Scheme: "https",
		Host:   domain,
	}, kept)

	return nil
}

// FindCookie returns the named cookie that would be sent to domain.
func (c *Client) FindCookie(name, domain string) (*http.Cookie, bool) {
	for _, cookie := range c.Cookies(domain) {
		if cookie.Name == name {
			return cookie, true
		}
	}

	return nil, false
}

// Cookies returns copies of the cookies that would be sent to domain.
// A leading-dot domain is looked up as its www host.
func (c *Client) Cookies(domain string) []*http.Cookie {
	if strings.HasPrefix(domain, ".") {
		domain = "www" + domain
	}

	return c.jar.Cookies(&url.URL{
		Scheme: "https",
		Host:   domain,
	})
}

// CookieString returns the cookies that would be sent to domain in header form.
func (c *Client) CookieString(domain string) string {
	return cookieHeader(c.Cookies(domain))
}

// ClearCookies removes all cookies and learned client-hint opt-ins.
func (c *Client) ClearCookies() {
	c.jar.Clear()
	c.http.ClearClientHints()
}

// UserAgent returns the default User-Agent header.
func (c *Client) UserAgent() string {
	return c.defaultHeaders.Get("User-Agent")
}

// ClientHint returns the default Sec-Ch-Ua header, or an empty string.
func (c *Client) ClientHint() string {
	return c.defaultHeaders.Get("Sec-Ch-Ua")
}

// Proxy returns a copy of the proxy configuration, or nil.
func (c *Client) Proxy() *Proxy {
	if c.proxy == nil {
		return nil
	}

	proxy := *c.proxy

	return &proxy
}

// Close closes idle connections. It is safe to call multiple times.
func (c *Client) Close() {
	c.http.CloseIdleConnections()
}
