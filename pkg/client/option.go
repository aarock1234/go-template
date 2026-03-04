package client

import (
	"net/http"

	utls "github.com/refraction-networking/utls"
)

// Option configures a Client.
type Option func(*Client)

// WithProxy sets the proxy configuration for the client.
func WithProxy(p *Proxy) Option {
	return func(c *Client) { c.proxy = p }
}

// WithBrowser sets the browser identity for TLS and HTTP/2 fingerprinting.
func WithBrowser(b Browser) Option {
	return func(c *Client) { c.browser = b }
}

// WithBrowserVersion sets the browser version for TLS fingerprinting.
func WithBrowserVersion(v string) Option {
	return func(c *Client) { c.browserVersion = v }
}

// WithPlatform sets the OS platform for TLS and HTTP/2 fingerprinting.
func WithPlatform(p Platform) Option {
	return func(c *Client) { c.platform = p }
}

// WithClientHelloID overrides the TLS ClientHelloID from the browser profile.
func WithClientHelloID(id utls.ClientHelloID) Option {
	return func(c *Client) { c.clientHelloID = new(id) }
}

// WithH2Profile overrides the HTTP/2 profile from the browser profile.
func WithH2Profile(p H2Profile) Option {
	return func(c *Client) { c.h2Profile = &p }
}

// WithCookieExtractor replaces the default Set-Cookie extraction.
func WithCookieExtractor(fn CookieExtractor) Option {
	return func(c *Client) { c.extractCookies = fn }
}

// WithDefaultHeaderOverrides overrides browser profile default headers.
func WithDefaultHeaderOverrides(h http.Header) Option {
	return func(c *Client) { c.defaultHeaderOverrides = h.Clone() }
}

// WithDisableKeepAlives controls whether HTTP keep-alives are disabled.
func WithDisableKeepAlives(d bool) Option {
	return func(c *Client) { c.disableKeepAlives = d }
}

// WithDisableSessionTickets controls whether TLS session tickets are disabled.
func WithDisableSessionTickets(d bool) Option {
	return func(c *Client) { c.disableSessionTickets = d }
}

// WithInsecureSkipVerify controls whether TLS certificate verification is skipped.
func WithInsecureSkipVerify(d bool) Option {
	return func(c *Client) { c.insecureSkipVerify = d }
}

// WithDisableDecompression controls whether automatic response decompression is disabled.
func WithDisableDecompression(d bool) Option {
	return func(c *Client) { c.disableDecompression = d }
}
