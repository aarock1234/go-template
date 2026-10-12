package client

import (
	http "github.com/aarock1234/fphttp"
	utls "github.com/refraction-networking/utls"
)

// Option configures a Client.
type Option func(*Client)

// WithProxy routes requests through a copy of p. A nil p disables the
// proxy.
func WithProxy(p *Proxy) Option {
	return func(c *Client) {
		if p == nil {
			c.proxy = nil

			return
		}

		proxy := *p
		c.proxy = &proxy
	}
}

// WithBrowser sets the browser whose fingerprint and identity headers the
// client presents.
func WithBrowser(b http.Browser) Option {
	return func(c *Client) { c.browser = b }
}

// WithBrowserVersion overrides the browser version used in identity
// headers. By default it matches the selected fingerprint profile.
func WithBrowserVersion(v string) Option {
	return func(c *Client) { c.browserVersion = v }
}

// WithPlatform sets the operating system whose fingerprint and identity
// headers the client presents.
func WithPlatform(p http.Platform) Option {
	return func(c *Client) { c.platform = p }
}

// WithFingerprint replaces the profile selected by browser and platform,
// for example with http.Chrome148Fetch or a customized profile. New clones
// f, so the caller keeps ownership.
func WithFingerprint(f *http.Fingerprint) Option {
	return func(c *Client) { c.fingerprint = f }
}

// WithClientHelloID overrides the TLS ClientHelloID of the fingerprint.
func WithClientHelloID(id utls.ClientHelloID) Option {
	return func(c *Client) { c.clientHelloID = &id }
}

// WithCookieExtractor replaces the default Set-Cookie extraction.
func WithCookieExtractor(fn CookieExtractor) Option {
	return func(c *Client) { c.extractCookies = fn }
}

// WithDefaultHeaderOverrides overrides identity headers. An override with
// an empty value removes the header. The option copies h.
func WithDefaultHeaderOverrides(h http.Header) Option {
	snapshot := h.Clone()

	return func(c *Client) { c.defaultHeaderOverrides = snapshot.Clone() }
}

// WithDisableDecompression controls whether automatic response
// decompression is disabled.
func WithDisableDecompression(disable bool) Option {
	return func(c *Client) { c.disableDecompression = disable }
}

// WithInsecureSkipVerify controls whether TLS certificate verification is
// skipped. Enable it only for targets or test environments where
// certificate validation is not possible.
func WithInsecureSkipVerify(skip bool) Option {
	return func(c *Client) { c.insecureSkipVerify = skip }
}
