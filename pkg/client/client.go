// Package client provides browser-fingerprinted HTTP requests, custom
// cookie storage, and proxy support.
package client

import (
	"net/url"
	"slices"
	"strings"

	http "github.com/aarock1234/fphttp/v2"
)

// Client sends requests as a selected browser release and owns its cookie
// storage. It is safe for concurrent use. Use New to initialize it.
type Client struct {
	http    *http.Client
	jar     *CookieJar
	profile *http.Profile

	browser              http.Browser
	platform             http.Platform
	version              http.Version
	proxy                *Proxy
	headerOverrides      http.Header
	disableDecompression bool
	insecureSkipVerify   bool
}

// Do sends req with the browser's headers, cookies, and redirect handling.
// Set req.Destination and req.Initiator to make it look like a fetch, a
// script, or an image rather than a page load. The caller owns the returned
// response body and must close it.
func (c *Client) Do(req *http.Request) (*http.Response, error) {
	return c.http.Do(req)
}

// Profile returns a copy of the profile the client sends requests with.
func (c *Client) Profile() *http.Profile {
	return c.profile.Clone()
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

// ClearCookies removes all cookies.
func (c *Client) ClearCookies() {
	c.jar.Clear()
}

// UserAgent returns the User-Agent header the client sends.
func (c *Client) UserAgent() string {
	return c.profile.Header.Get("User-Agent")
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
