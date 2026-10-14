package client

import (
	http "github.com/aarock1234/fphttp"
	"github.com/aarock1234/fphttp/browser"
	utls "github.com/refraction-networking/utls"
)

// Option configures a Client.
type Option func(*Client)

// WithProfile selects an immutable resolved profile, including channel, device,
// and configuration variants. It takes precedence over the browser, platform,
// and version options. Nil restores ordinary catalog selection.
func WithProfile(profile *browser.Profile) Option {
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

// WithBrowser sets the browser whose fingerprint and identity headers the
// client presents.
func WithBrowser(b http.Browser) Option {
	return func(c *Client) { c.browser = b }
}

// WithBrowserVersion selects a catalogued browser version and its matching
// wire profile. A major or dotted version prefix selects its latest build.
func WithBrowserVersion(v string) Option {
	return func(c *Client) { c.browserVersion = v }
}

// WithPlatform sets the operating system whose fingerprint and identity
// headers the client presents.
func WithPlatform(p http.Platform) Option {
	return func(c *Client) { c.platform = p }
}

// WithFingerprint manually overrides the selected identity's wire profile.
// New clones f. This does not establish equivalence with the catalogued browser.
func WithFingerprint(f *http.Fingerprint) Option {
	return func(c *Client) { c.fingerprint = f }
}

// WithClientHelloID manually replaces the TLS hello and clears any spec factory.
// A known identity without a catalog wire definition retains the legacy H2
// baseline. Use WithFingerprint to supply different connection settings.
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

// WithDisableHeaders disables generated browser headers. Explicit request
// headers and WithDefaultHeaderOverrides remain in effect.
func WithDisableHeaders(disable bool) Option {
	return func(c *Client) { c.disableHeaders = disable }
}

// WithInsecureSkipVerify controls whether TLS certificate verification is
// skipped. Enable it only for targets or test environments where
// certificate validation is not possible.
func WithInsecureSkipVerify(skip bool) Option {
	return func(c *Client) { c.insecureSkipVerify = skip }
}
