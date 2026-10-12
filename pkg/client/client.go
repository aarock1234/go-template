// Package client provides an HTTP client with browser TLS and HTTP/2
// fingerprints, browser identity headers, a cookie jar that captures cookies
// across redirects, and proxy support.
package client

import (
	"crypto/tls"
	"errors"
	"fmt"
	"log/slog"
	"net/url"
	"slices"
	"strings"

	http "github.com/aarock1234/fphttp"
	utls "github.com/refraction-networking/utls"
)

// maxRedirects is the number of redirects Do follows before giving up.
const maxRedirects = 10

// ErrTooManyRedirects is returned when a request exceeds the maximum number
// of allowed redirects.
var ErrTooManyRedirects = errors.New("too many redirects")

// sensitiveHeaders are dropped when a redirect leaves the current host and
// its subdomains.
var sensitiveHeaders = []string{
	"Authorization",
	"Proxy-Authorization",
	"Www-Authenticate",
	"Cookie2",
}

// bodyHeaders describe a request body and are dropped when a redirect
// switches to a body-less request.
var bodyHeaders = []string{
	"Content-Encoding",
	"Content-Language",
	"Content-Length",
	"Content-Location",
	"Content-Type",
}

// Client is an HTTP client that presents a browser's TLS fingerprint,
// HTTP/2 settings, header order, and identity headers. It is safe for
// concurrent use by multiple goroutines.
type Client struct {
	http           *http.Client
	jar            *CookieJar
	defaultHeaders http.Header

	browser                http.Browser
	platform               http.Platform
	browserVersion         string
	fingerprint            *http.Fingerprint
	clientHelloID          *utls.ClientHelloID
	proxy                  *Proxy
	extractCookies         CookieExtractor
	defaultHeaderOverrides http.Header
	disableDecompression   bool
	insecureSkipVerify     bool
}

// New creates a Client with the given options. By default it presents
// Chrome on Windows and verifies TLS certificates. It returns an error for a
// browser and platform combination without a fingerprint profile or
// identity headers.
func New(opts ...Option) (*Client, error) {
	c := &Client{
		jar:            &CookieJar{},
		browser:        http.BrowserChrome,
		platform:       http.PlatformWindows,
		extractCookies: DefaultCookieExtractor,
	}

	for _, opt := range opts {
		opt(c)
	}

	fingerprint, err := c.resolveFingerprint()
	if err != nil {
		return nil, err
	}

	if c.browserVersion == "" {
		c.browserVersion = defaultVersion(c.browser, c.platform)
	}

	headers, err := identityHeaders(c.browser, c.platform, c.browserVersion)
	if err != nil {
		return nil, err
	}

	c.defaultHeaders = mergeHeaders(headers, c.defaultHeaderOverrides)

	transport, err := c.newTransport(fingerprint)
	if err != nil {
		return nil, err
	}

	c.http = &http.Client{
		Transport: transport,
		CheckRedirect: func(*http.Request, []*http.Request) error {
			return http.ErrUseLastResponse
		},
	}

	return c, nil
}

// resolveFingerprint returns a private copy of the configured fingerprint,
// or of the fphttp profile for the browser and platform.
func (c *Client) resolveFingerprint() (*http.Fingerprint, error) {
	fingerprint := c.fingerprint
	if fingerprint == nil {
		fingerprint = http.Profile(c.browser, c.platform)
	}

	if fingerprint == nil {
		return nil, fmt.Errorf("no fingerprint profile for %s on %s", c.browser, c.platform)
	}

	fingerprint = fingerprint.Clone()

	if c.clientHelloID != nil {
		fingerprint.ClientHelloID = *c.clientHelloID
	}

	if err := fingerprint.Validate(); err != nil {
		return nil, fmt.Errorf("validate fingerprint: %w", err)
	}

	return fingerprint, nil
}

// newTransport clones fphttp's default transport, which carries its dial,
// TLS handshake, and idle connection timeouts, and applies the client's
// fingerprint, TLS, and proxy settings.
func (c *Client) newTransport(fingerprint *http.Fingerprint) (*http.Transport, error) {
	base, ok := http.DefaultTransport.(*http.Transport)
	if !ok {
		return nil, fmt.Errorf("default transport is %T, want *http.Transport", http.DefaultTransport)
	}

	transport := base.Clone()
	transport.Fingerprint = fingerprint

	if c.insecureSkipVerify {
		transport.TLSClientConfig = &tls.Config{
			InsecureSkipVerify: true, //nolint:gosec // explicitly requested with WithInsecureSkipVerify
		}
	}

	if c.proxy != nil {
		transport.Proxy = http.ProxyURL(c.proxy.URL())
	}

	return transport, nil
}

// Do sends req and returns the final response. Redirects are followed
// manually so cookies set by intermediate responses reach the jar. Default
// headers fill in any header the request does not set; req itself is not
// modified. When the error is nil, the caller must close the response body.
func (c *Client) Do(req *http.Request) (*http.Response, error) {
	req = req.Clone(req.Context())

	for redirects := 0; ; redirects++ {
		c.applyDefaultHeaders(req)
		c.applyCookies(req)

		resp, err := c.http.Do(req)
		if err != nil {
			return nil, err
		}

		slog.DebugContext(req.Context(), "round trip complete",
			slog.String("host", req.URL.Hostname()),
			slog.Int("status", resp.StatusCode),
		)

		if err := c.storeCookies(resp); err != nil {
			_ = resp.Body.Close()

			return nil, err
		}

		method, ok := redirectMethod(req, resp)
		if !ok {
			if !c.disableDecompression {
				decompress(resp)
			}

			return resp, nil
		}

		_ = resp.Body.Close()

		if redirects == maxRedirects {
			return nil, ErrTooManyRedirects
		}

		req, err = redirectRequest(req, resp, method)
		if err != nil {
			return nil, err
		}
	}
}

// applyDefaultHeaders sets each default header that req does not set.
func (c *Client) applyDefaultHeaders(req *http.Request) {
	for key, values := range c.defaultHeaders {
		if _, ok := req.Header[key]; !ok {
			req.Header[key] = slices.Clone(values)
		}
	}
}

// applyCookies adds the jar's cookies for req's URL after any cookies the
// request already carries.
func (c *Client) applyCookies(req *http.Request) {
	cookies := c.jar.Cookies(req.URL)
	if len(cookies) == 0 {
		return
	}

	header := cookieHeader(cookies)
	if existing := req.Header.Get("Cookie"); existing != "" {
		header = existing + "; " + header
	}

	req.Header.Set("Cookie", header)
}

// storeCookies saves the cookies set by resp in the jar.
func (c *Client) storeCookies(resp *http.Response) error {
	cookies, err := c.extractCookies(resp)
	if err != nil {
		return fmt.Errorf("extract cookies: %w", err)
	}

	c.jar.SetCookies(resp.Request.URL, cookies)

	return nil
}

// redirectMethod returns the method for following resp, or false when resp
// is not a redirect that can be followed. Like net/http, 301, 302, and 303
// switch to GET (HEAD stays HEAD), while 307 and 308 keep the method and
// body and are not followed when the body cannot be replayed.
func redirectMethod(req *http.Request, resp *http.Response) (string, bool) {
	if resp.Header.Get("Location") == "" {
		return "", false
	}

	switch resp.StatusCode {
	case http.StatusMovedPermanently, http.StatusFound, http.StatusSeeOther:
		if req.Method == http.MethodHead {
			return http.MethodHead, true
		}

		return http.MethodGet, true
	case http.StatusTemporaryRedirect, http.StatusPermanentRedirect:
		if req.GetBody == nil && req.Body != nil && req.Body != http.NoBody {
			return "", false
		}

		return req.Method, true
	default:
		return "", false
	}
}

// redirectRequest builds the request that follows resp with method. It
// starts from a clone of req, so fphttp's per-request header order and
// priority overrides carry over, and drops headers that must not follow
// the redirect.
func redirectRequest(req *http.Request, resp *http.Response, method string) (*http.Request, error) {
	location, err := resp.Location()
	if err != nil {
		return nil, fmt.Errorf("parse redirect location: %w", err)
	}

	next := req.Clone(req.Context())
	next.Method = method
	next.URL = location
	next.Response = resp
	next.Header.Del("Cookie")

	if location.Host != req.URL.Host {
		next.Host = ""
	}

	if !sameOrSubdomain(location.Hostname(), req.URL.Hostname()) {
		for _, name := range sensitiveHeaders {
			next.Header.Del(name)
		}
	}

	switch resp.StatusCode {
	case http.StatusTemporaryRedirect, http.StatusPermanentRedirect:
		if req.GetBody == nil {
			break
		}

		body, err := req.GetBody()
		if err != nil {
			return nil, fmt.Errorf("replay request body: %w", err)
		}

		next.Body = body
	default:
		next.Body = nil
		next.GetBody = nil
		next.ContentLength = 0
		next.TransferEncoding = nil

		for _, name := range bodyHeaders {
			next.Header.Del(name)
		}
	}

	return next, nil
}

// sameOrSubdomain reports whether host is parent or one of its subdomains.
func sameOrSubdomain(host, parent string) bool {
	host = strings.ToLower(host)
	parent = strings.ToLower(parent)

	return host == parent || strings.HasSuffix(host, "."+parent)
}

// SetCookies stores cookies in the jar as if u had set them.
func (c *Client) SetCookies(u *url.URL, cookies []*http.Cookie) {
	c.jar.SetCookies(u, cookies)
}

// SetCookieString parses cookieString (see ParseCookies) and stores the
// cookies for domain and its subdomains, skipping the names in exclude. A
// leading "www." is removed from domain, so cookies set for
// "www.example.com" are shared with every subdomain of "example.com".
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

// FindCookie returns the cookie with the given name that would be sent to
// domain.
func (c *Client) FindCookie(name, domain string) (*http.Cookie, bool) {
	for _, cookie := range c.Cookies(domain) {
		if cookie.Name == name {
			return cookie, true
		}
	}

	return nil, false
}

// Cookies returns copies of the cookies that would be sent to domain. A
// leading-dot domain such as ".example.com" is looked up as its www host.
func (c *Client) Cookies(domain string) []*http.Cookie {
	if strings.HasPrefix(domain, ".") {
		domain = "www" + domain
	}

	return c.jar.Cookies(&url.URL{
		Scheme: "https",
		Host:   domain,
	})
}

// CookieString returns the cookies that would be sent to domain in Cookie
// header form ("name=value; name2=value2").
func (c *Client) CookieString(domain string) string {
	return cookieHeader(c.Cookies(domain))
}

// ClearCookies removes every cookie from the jar.
func (c *Client) ClearCookies() {
	c.jar.Clear()
}

// UserAgent returns the default User-Agent header.
func (c *Client) UserAgent() string {
	return c.defaultHeaders.Get("User-Agent")
}

// ClientHint returns the default Sec-Ch-Ua header, which is empty for
// browsers that do not send client hints.
func (c *Client) ClientHint() string {
	return c.defaultHeaders.Get("Sec-Ch-Ua")
}

// Proxy returns the proxy configuration used by this client, or nil.
func (c *Client) Proxy() *Proxy {
	return c.proxy
}

// Close closes idle connections. It is safe to call multiple times.
func (c *Client) Close() {
	c.http.CloseIdleConnections()
}
