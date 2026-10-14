package client

import (
	"crypto/tls"
	"errors"
	"fmt"

	http "github.com/aarock1234/fphttp"
	"github.com/aarock1234/fphttp/browser"
)

// New creates a client using the latest catalogued Chrome identity on Windows
// unless options select another identity. Browser versions resolve both wire
// profiles and identity headers. Unknown profiles return errors. A known but
// unavailable identity requires an explicit manual wire override.
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
	if c.extractCookies == nil {
		return nil, errors.New("cookie extractor must not be nil")
	}

	profile, err := c.resolveProfile()
	if err != nil {
		return nil, fmt.Errorf("resolve browser profile: %w", err)
	}
	c.profile = profile
	descriptor := profile.Descriptor()
	c.browser = descriptor.Browser
	c.platform = descriptor.Platform
	c.browserVersion = descriptor.Version
	c.defaultHeaders = mergeHeaders(profile.IdentityHeaders(), c.defaultHeaderOverrides)
	if c.disableHeaders {
		c.defaultHeaders = mergeHeaders(make(http.Header), c.defaultHeaderOverrides)
	}

	transport, err := c.newTransport()
	if err != nil {
		return nil, err
	}
	fingerprint, err := c.manualFingerprint(profile)
	if err != nil {
		return nil, err
	}
	c.http, err = browser.New(browser.Config{
		Profile:              profile,
		Transport:            transport,
		Fingerprint:          fingerprint,
		DefaultHeaders:       c.defaultHeaderOverrides,
		CookiePolicy:         cookiePolicy{client: c},
		CheckRedirect:        redirectLimit,
		DisableHeaders:       c.disableHeaders,
		DisableDecompression: c.disableDecompression,
	})
	if err != nil {
		return nil, fmt.Errorf("create browser client: %w", err)
	}

	return c, nil
}

func (c *Client) resolveProfile() (*browser.Profile, error) {
	if c.profile != nil {
		if c.profile.Descriptor().ID == "" {
			return nil, errors.New("browser profile is not initialized")
		}

		return c.profile, nil
	}
	selector := browser.Selector{
		Browser:  c.browser,
		Platform: c.platform,
		Version:  c.browserVersion,
	}
	profile, err := browser.Resolve(selector)
	if err == nil {
		return profile, nil
	}
	manual := c.fingerprint != nil || c.clientHelloID != nil
	if !manual || !errors.Is(err, browser.ErrUnavailable) {
		return nil, err
	}

	return browser.ResolveIdentity(selector)
}

func (c *Client) manualFingerprint(profile *browser.Profile) (*http.Fingerprint, error) {
	if c.fingerprint == nil && c.clientHelloID == nil {
		return nil, nil
	}

	fingerprint := c.fingerprint
	if fingerprint == nil {
		fingerprint = profile.Fingerprint()
		if fingerprint == nil {
			// An explicit hello override preserves the legacy H2 selection when
			// this published identity has no executable catalog definition.
			fingerprint = http.Profile(c.browser, c.platform).Clone()
		}
	} else {
		fingerprint = fingerprint.Clone()
	}
	if fingerprint == nil {
		return nil, errors.New("manual hello requires an explicit fingerprint for this browser and platform")
	}
	if c.clientHelloID != nil {
		fingerprint.ClientHelloID = *c.clientHelloID
		fingerprint.ClientHelloSpec = nil
		fingerprint.ClientHelloSpecFactory = nil
	}
	if err := fingerprint.Validate(); err != nil {
		return nil, fmt.Errorf("validate manual fingerprint: %w", err)
	}

	return fingerprint, nil
}

func (c *Client) newTransport() (*http.Transport, error) {
	base, ok := http.DefaultTransport.(*http.Transport)
	if !ok {
		return nil, fmt.Errorf("default transport is %T, want *http.Transport", http.DefaultTransport)
	}

	transport := base.Clone()
	if c.insecureSkipVerify {
		transport.TLSClientConfig = &tls.Config{
			InsecureSkipVerify: true, //nolint:gosec // Explicitly selected by the caller.
		}
	}
	if c.proxy != nil {
		transport.Proxy = http.ProxyURL(c.proxy.URL())
	}

	return transport, nil
}

func redirectLimit(_ *http.Request, via []*http.Request) error {
	if len(via) > maxRedirects {
		return ErrTooManyRedirects
	}

	return nil
}
