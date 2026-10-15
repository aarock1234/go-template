package client

import (
	http "github.com/aarock1234/fphttp/v2"
)

// Option configures a Client.
type Option func(*Client)

// WithProfile sends requests as profile describes, which may be one from
// http.NewProfile that the caller has changed. It takes precedence over the
// browser, platform, and version options. New copies the profile. Nil
// restores ordinary catalog selection.
func WithProfile(profile *http.Profile) Option {
	return func(c *Client) { c.profile = profile }
}

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

// WithBrowser sets the browser the client presents.
func WithBrowser(b http.Browser) Option {
	return func(c *Client) { c.browser = b }
}

// WithPlatform sets the operating system the client presents.
func WithPlatform(p http.Platform) Option {
	return func(c *Client) { c.platform = p }
}

// WithBrowserVersion selects a browser release. A major version such as
// "152" selects its newest build; http.Latest selects the newest release.
func WithBrowserVersion(v http.Version) Option {
	return func(c *Client) { c.version = v }
}

// WithDefaultHeaderOverrides sets fields sent with every request, replacing
// the profile's own values such as User-Agent and Accept-Language. An
// override with an empty value suppresses that field, including ones the
// browser would generate. The option copies h.
func WithDefaultHeaderOverrides(h http.Header) Option {
	snapshot := h.Clone()

	return func(c *Client) { c.headerOverrides = snapshot.Clone() }
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
