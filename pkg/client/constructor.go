package client

import (
	"crypto/tls"
	"fmt"
	"slices"

	http "github.com/aarock1234/fphttp/v2"
)

// New creates a client that sends requests as the newest Chrome release on
// Windows unless options select another browser, platform, version, or
// profile. An unknown browser, platform, or version returns an error.
func New(opts ...Option) (*Client, error) {
	c := &Client{
		jar:      &CookieJar{},
		browser:  http.BrowserChrome,
		platform: http.PlatformWindows,
		version:  http.Latest,
	}
	for _, opt := range opts {
		opt(c)
	}

	profile, err := c.resolveProfile()
	if err != nil {
		return nil, fmt.Errorf("resolve browser profile: %w", err)
	}

	c.profile = profile
	c.http = http.NewClient(profile)
	c.http.Jar = c.jar

	if err := c.configureTransport(); err != nil {
		return nil, err
	}

	return c, nil
}

// resolveProfile returns a copy of the profile WithProfile selected, or the
// profile of the selected browser release, with header overrides applied.
func (c *Client) resolveProfile() (*http.Profile, error) {
	profile := c.profile.Clone()
	if profile == nil {
		resolved, err := http.NewProfile(c.browser, c.platform, c.version)
		if err != nil {
			return nil, err
		}

		profile = resolved
	}

	if len(c.headerOverrides) > 0 && profile.Header == nil {
		profile.Header = make(http.Header)
	}
	for name, values := range c.headerOverrides {
		profile.Header[http.CanonicalHeaderKey(name)] = slices.Clone(values)
	}

	return profile, nil
}

// configureTransport applies the proxy, decompression, and certificate
// settings to the client's transport.
func (c *Client) configureTransport() error {
	transport, ok := c.http.Transport.(*http.Transport)
	if !ok {
		return fmt.Errorf("client transport is %T, want *http.Transport", c.http.Transport)
	}

	transport.DisableCompression = c.disableDecompression
	if c.insecureSkipVerify {
		transport.TLSClientConfig = &tls.Config{
			InsecureSkipVerify: true, //nolint:gosec // Explicitly selected by the caller.
		}
	}
	if c.proxy != nil {
		transport.Proxy = http.ProxyURL(c.proxy.URL())
	}

	return nil
}
