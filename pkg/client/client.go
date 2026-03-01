// Package client provides an HTTP client with cookie jar and proxy support.
package client

import (
	"fmt"
	"net/url"
	"slices"

	utls "github.com/refraction-networking/utls"
	http "github.com/saucesteals/fhttp"
	"github.com/saucesteals/mimic"
)

// BrowserBrand identifies the browser for TLS and HTTP/2 fingerprinting.
type BrowserBrand string

const (
	// BrandChrome identifies Google Chrome.
	BrandChrome BrowserBrand = "chrome"
	// BrandEdge identifies Microsoft Edge.
	BrandEdge BrowserBrand = "edge"
	// BrandBrave identifies the Brave browser.
	BrandBrave BrowserBrand = "brave"
)

// BrowserPlatform identifies the OS platform for TLS and HTTP/2 fingerprinting.
type BrowserPlatform string

const (
	// PlatformWindows targets the Windows operating system.
	PlatformWindows BrowserPlatform = "windows"
	// PlatformMac targets macOS.
	PlatformMac BrowserPlatform = "mac"
	// PlatformLinux targets Linux.
	PlatformLinux BrowserPlatform = "linux"
	// PlatformIOS targets iOS (mapped to macOS in mimic until native support lands).
	PlatformIOS BrowserPlatform = "ios"
)

// IOSSafariUserAgent is a User-Agent string for iOS Safari 18.7.
const IOSSafariUserAgent = "Mozilla/5.0 (iPhone; CPU iPhone OS 18_7 like Mac OS X) AppleWebKit/605.1.15 (KHTML, like Gecko) Mobile/15E148"

const (
	defaultBrowserVersion  = "144.0.0.0"
	defaultBrowserBrand    = BrandChrome
	defaultBrowserPlatform = PlatformWindows
)

func (b BrowserBrand) mimicBrand() mimic.Brand {
	switch b {
	case BrandEdge:
		return mimic.BrandEdge
	case BrandBrave:
		return mimic.BrandBrave
	default:
		return mimic.BrandChrome
	}
}

func (p BrowserPlatform) mimicPlatform() mimic.Platform {
	switch p {
	case PlatformMac:
		return mimic.PlatformMac
	case PlatformLinux:
		return mimic.PlatformLinux
	case PlatformIOS:
		// temporary: mimic does not support ios directly yet.
		return mimic.PlatformMac
	default:
		return mimic.PlatformWindows
	}
}

// CookieExtractor extracts cookies from an HTTP response.
// Called after each round trip, including intermediate redirects.
// The returned cookies are stored in the client's jar.
type CookieExtractor func(resp *http.Response) ([]*http.Cookie, error)

// Option configures a Client.
type Option func(*Client)

// WithCookieExtractor replaces the default Set-Cookie extraction.
// Use this to handle non-standard cookie headers (e.g., custom JSON cookie headers).
func WithCookieExtractor(fn CookieExtractor) Option {
	return func(c *Client) {
		c.extractCookies = fn
	}
}

// WithBrowserVersion sets the Chrome version for TLS fingerprinting.
// Default: "144.0.0.0".
func WithBrowserVersion(version string) Option {
	return func(c *Client) { c.browserVersion = version }
}

// WithBrowserBrand sets the browser brand for TLS fingerprinting.
// Default: BrandChrome.
func WithBrowserBrand(brand BrowserBrand) Option {
	return func(c *Client) { c.browserBrand = brand }
}

// WithBrowserPlatform sets the OS platform for TLS fingerprinting.
// Default: PlatformWindows.
func WithBrowserPlatform(platform BrowserPlatform) Option {
	return func(c *Client) { c.browserPlatform = platform }
}

// WithClientHelloID sets the ClientHelloID for TLS fingerprinting.
// Overrides the browser brand and platform.
// Default: nil.
func WithClientHelloID(id utls.ClientHelloID) Option {
	return func(c *Client) { c.clientHelloID = &id }
}

// WithDefaultHeaderOverrides overrides mimic default headers.
func WithDefaultHeaderOverrides(headers http.Header) Option {
	return func(c *Client) { c.defaultHeaderOverrides = headers.Clone() }
}

// WithDisableSessionTickets disables session tickets.
// Default: true.
func WithDisableSessionTickets(disable bool) Option {
	return func(c *Client) { c.disableSessionTickets = disable }
}

// WithDisableKeepAlives disables keep-alives.
// Default: true.
func WithDisableKeepAlives(disable bool) Option {
	return func(c *Client) { c.disableKeepAlives = disable }
}

// WithInsecureSkipVerify disables TLS verification.
// Default: true.
func WithInsecureSkipVerify(insecure bool) Option {
	return func(c *Client) { c.insecureSkipVerify = insecure }
}

// Client is an HTTP client with TLS fingerprinting, cookie jar, and proxy support.
type Client struct {
	http  *http.Client
	jar   http.CookieJar
	proxy *Proxy

	browserVersion  string
	browserBrand    BrowserBrand
	browserPlatform BrowserPlatform

	defaultHeaderOverrides http.Header
	extractCookies         CookieExtractor
	clientHelloID          *utls.ClientHelloID
	disableSessionTickets  bool
	disableKeepAlives      bool
	insecureSkipVerify     bool
}

// New creates a new HTTP client with the specified proxy configuration.
func New(proxy *Proxy, opts ...Option) (*Client, error) {
	jar, err := newCookieJar()
	if err != nil {
		return nil, fmt.Errorf("creating cookie jar: %w", err)
	}

	c := &Client{
		jar:                   jar,
		proxy:                 proxy,
		browserVersion:        defaultBrowserVersion,
		browserBrand:          defaultBrowserBrand,
		browserPlatform:       defaultBrowserPlatform,
		extractCookies:        DefaultCookieExtractor,
		disableSessionTickets: true,
		disableKeepAlives:     true,
		insecureSkipVerify:    true,
	}

	for _, opt := range opts {
		opt(c)
	}

	var proxyURL func(r *http.Request) (*url.URL, error)
	if proxy != nil {
		proxyURL = http.ProxyURL(proxy.URL())
	}

	httpTransport := &http.Transport{
		TLSClientConfig: &utls.Config{
			InsecureSkipVerify:     c.insecureSkipVerify,
			SessionTicketsDisabled: c.disableSessionTickets,
		},
		DisableCompression: true,
		Proxy:              proxyURL,
		DisableKeepAlives:  c.disableKeepAlives,
	}

	if c.disableSessionTickets {
		httpTransport.TLSClientConfig.ClientSessionCache = nil
	}

	if c.disableKeepAlives {
		httpTransport.MaxIdleConns = 0
		httpTransport.MaxIdleConnsPerHost = 0
	}

	transport, err := mimic.NewTransport(mimic.TransportOptions{
		Version:   c.browserVersion,
		Brand:     c.browserBrand.mimicBrand(),
		Platform:  c.browserPlatform.mimicPlatform(),
		Transport: httpTransport,
	})
	if err != nil {
		return nil, fmt.Errorf("creating mimic transport: %w", err)
	}

	for key, values := range c.defaultHeaderOverrides {
		transport.DefaultHeaders.Del(key)
		if len(values) == 0 || values[0] == "" {
			continue
		}

		transport.DefaultHeaders.Set(key, values[0])
	}

	if c.clientHelloID != nil {
		transport.Transport.(*http.Transport).GetTlsClientHelloSpec = func() *utls.ClientHelloSpec {
			spec, err := utls.UTLSIdToSpec(*c.clientHelloID)
			if err != nil {
				return nil
			}

			sanitizeCurves(&spec)

			return &spec
		}
	}

	c.http = &http.Client{
		Transport: transport,
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			return http.ErrUseLastResponse
		},
	}

	return c, nil
}

// UserAgent returns the default User-Agent header from the mimic transport.
func (c *Client) UserAgent() string {
	if transport, ok := c.http.Transport.(*mimic.Transport); ok {
		return transport.DefaultHeaders.Get("user-agent")
	}

	return ""
}

// ClientHint returns the default sec-ch-ua header from the mimic transport.
func (c *Client) ClientHint() string {
	if transport, ok := c.http.Transport.(*mimic.Transport); ok {
		return transport.DefaultHeaders.Get("sec-ch-ua")
	}

	return ""
}

// Proxy returns the proxy configuration used by this client.
func (c *Client) Proxy() *Proxy {
	return c.proxy
}

var supportedCurves = map[utls.CurveID]struct{}{
	utls.CurveP256: {},
	utls.CurveP384: {},
	utls.X25519:    {},
}

// sanitizeCurves removes curves from the SupportedCurvesExtension that the
// underlying TLS stack cannot negotiate. Randomized ClientHello specs may
// include experimental or fake curves (e.g. Kyber drafts) that cause
// "CurvePreferences includes unsupported curve" errors.
func sanitizeCurves(spec *utls.ClientHelloSpec) {
	for _, ext := range spec.Extensions {
		sce, ok := ext.(*utls.SupportedCurvesExtension)
		if !ok {
			continue
		}

		sce.Curves = slices.DeleteFunc(sce.Curves, func(c utls.CurveID) bool {
			_, ok := supportedCurves[c]
			return !ok
		})

		return
	}
}
