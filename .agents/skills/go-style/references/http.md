# HTTP client architecture and proxy configuration

[Back to the Go entrypoint](../SKILL.md). Read the sections matching the current change; follow crosslinks when the change crosses a boundary.

- [Ordinary clients and specialized applicability](#go-http-client)
- [Specialized client.go](#go-http-client-structure)
- [option.go](#go-http-options)
- [proxy.go](#go-proxy-configuration)
- [Specialized usage example](#go-http-usage)

<a id="go-http-client"></a>

## HTTP Client Pattern

Structure: `pkg/client/{client.go, option.go, proxy.go, cookie.go, ...}`

This is the local template's specialized browser-fingerprinting pattern. The current template uses `github.com/aarock1234/fphttp/v2` (imported as `http`), a `net/http` fork with browser fingerprints, rather than the standard-library transport described in the earlier guide. It wraps `http.NewClient` with functional options, typed proxy configuration, and a custom cookie jar, and lets fphttp own redirects, browser headers, and response decoding. Inspect the actual dependency versions and request/response types before copying a fragment.

For ordinary HTTP APIs, prefer a reused standard `net/http.Client` with an explicit deadline policy and a cookie jar when needed. The standard client's jar already receives cookies across redirects; ordinary cookie capture alone does not require a manual redirect engine.

A custom redirect engine owns every intermediate response and must preserve method/body replay semantics, restrict credential forwarding across origins, and retain request cancellation. Treat the template as a structural reference with a specialized purpose, not proof that each transport policy fits a new application.

Reuse clients and transports. Reading a response body to EOF before closing supports connection reuse; one JSON decode does not establish EOF. Choose bounded consumption or deliberate early close, and avoid draining an untrusted or stalled body without a size and time budget.

Source: [HTTP request and response ownership](https://pkg.go.dev/net/http@go1.26.1#Client.Do).


Sources: [reference 1](https://pkg.go.dev/net/http@go1.26.1#Client).

<a id="go-http-client-structure"></a>

### client.go

Functional options configure the client. Sensible defaults mean callers only override what they need.

```go
package client

import http "github.com/aarock1234/fphttp/v2"

// Client sends requests as a selected browser release and owns its
// cookie storage.
type Client struct {
    http      *http.Client
    jar       *CookieJar
    profile   *http.Profile
    proxy     *Proxy
    browser   http.Browser
    platform  http.Platform
    version   http.Version
    // ...
}

// New creates a new Client with the given options.
func New(opts ...Option) (*Client, error) {
    c := &Client{
        browser:  http.BrowserChrome,
        platform: http.PlatformWindows,
        // sensible defaults...
    }

    for _, opt := range opts {
        opt(c)
    }

    // resolve the profile with http.NewProfile, then build the client
    // with http.NewClient, set its Jar, and configure the proxy...

    return c, nil
}

// Do sends req with the browser's headers, cookies, and redirect
// handling. The caller owns the returned response body and must close it.
func (c *Client) Do(req *http.Request) (*http.Response, error) {
    return c.http.Do(req)
}
```

The abbreviated construction describes responsibilities, not a drop-in implementation. fphttp's `Client` applies the jar, follows redirects, and strips credentials across hosts as a browser does, so the template no longer follows redirects by hand. Returning a final response transfers responsibility for closing its body to the caller.


Sources: [reference 1](https://pkg.go.dev/net/http@go1.26.1#Client).

<a id="go-http-options"></a>

### option.go

One option per concern. Each returns an `Option` closure.

```go
package client

// Option configures a Client.
type Option func(*Client)

func WithProxy(p *Proxy) Option {
    return func(c *Client) { c.proxy = p }
}

func WithBrowser(b http.Browser) Option {
    return func(c *Client) { c.browser = b }
}

func WithPlatform(p http.Platform) Option {
    return func(c *Client) { c.platform = p }
}

func WithBrowserVersion(v http.Version) Option {
    return func(c *Client) { c.version = v }
}

func WithDefaultHeaderOverrides(h http.Header) Option {
    return func(c *Client) { c.defaultHeaderOverrides = h.Clone() }
}
```


<a id="go-proxy-configuration"></a>

### proxy.go

Typed `Proxy` struct instead of raw `*url.URL`. Keeps host, port, credentials, and scheme separate for cleaner access and conversion.

The shorthand below is a project-specific input format, not a general URI parser. Its colon separator cannot represent arbitrary IPv6 hosts or credentials containing colons. Use a URL-based format when those values are required. Diagnostics identify fields and line numbers without printing credential-bearing input.

```go
package client

// Proxy holds a parsed proxy configuration.
type Proxy struct {
    Scheme   ProxyScheme
    Host     string
    Port     string
    Username string
    Password string
}

// URL returns the proxy as a *url.URL suitable for http.Transport.Proxy.
func (p *Proxy) URL() *url.URL {
    u := &url.URL{
        Scheme: string(p.Scheme),
        Host:   net.JoinHostPort(p.Host, p.Port),
    }

    if p.Username != "" && p.Password != "" {
        u.User = url.UserPassword(p.Username, p.Password)
    }

    return u
}

// ParseProxy parses a "host:port" or "host:port:user:pass" string.
// An empty string returns (nil, nil).
func ParseProxy(proxy string, scheme ProxyScheme) (*Proxy, error) {
    proxy = strings.TrimSpace(proxy)
    if proxy == "" {
        return nil, nil
    }

    split := strings.Split(proxy, ":")
    if len(split) != 2 && len(split) != 4 {
        return nil, fmt.Errorf("got %d proxy fields, want 2 or 4", len(split))
    }

    for _, field := range split {
        if field == "" {
            return nil, errors.New("proxy fields must not be empty")
        }
    }

    port, err := strconv.Atoi(split[1])
    if err != nil || port < 1 || port > 65535 {
        return nil, errors.New("proxy port must be an integer between 1 and 65535")
    }

    parsed := &Proxy{
        Scheme: scheme,
        Host:   split[0],
        Port:   split[1],
    }

    if len(split) == 4 {
        parsed.Username = split[2]
        parsed.Password = split[3]
    }

    return parsed, nil
}

// ImportProxies reads proxy configs from a file, one per line.
func ImportProxies(filename string, scheme ProxyScheme) ([]*Proxy, error) {
    f, err := os.Open(filename)
    if err != nil {
        return nil, fmt.Errorf("opening proxy file: %w", err)
    }
    defer func() { _ = f.Close() }()

    var proxies []*Proxy
    scanner := bufio.NewScanner(f)
    lineNumber := 0
    for scanner.Scan() {
        lineNumber++
        line := strings.TrimSpace(scanner.Text())
        if line == "" {
            continue
        }

        proxy, err := ParseProxy(line, scheme)
        if err != nil {
            return nil, fmt.Errorf("parsing proxy line %d: %w", lineNumber, err)
        }

        proxies = append(proxies, proxy)
    }

    if err := scanner.Err(); err != nil {
        return nil, fmt.Errorf("scanning proxy file: %w", err)
    }

    return proxies, nil
}
```


Sources: [reference 1](https://pkg.go.dev/net#JoinHostPort), [reference 2](https://pkg.go.dev/log/slog@go1.26.1#LogValuer).

<a id="go-http-usage"></a>

### Usage Example

Downstream packages compose the specialized client with the options they need. Use the same `fphttp` request/response types as the current template when adopting this client; standard-library examples are not interchangeable by import name alone.

```go
type APIClient struct {
    http    *client.Client
    baseURL string
}

func New(cfg *Config) (*APIClient, error) {
    proxy, err := client.ParseProxy(cfg.Proxy, client.ProxySchemeHTTP)
    if err != nil {
        return nil, fmt.Errorf("parsing proxy: %w", err)
    }

    opts := []client.Option{
        client.WithPlatform(http.PlatformWindows),
        client.WithBrowserVersion("152"),
    }

    if proxy != nil {
        opts = append(opts, client.WithProxy(proxy))
    }

    httpClient, err := client.New(opts...)
    if err != nil {
        return nil, fmt.Errorf("creating http client: %w", err)
    }

    return &APIClient{
        http:    httpClient,
        baseURL: cfg.BaseURL,
    }, nil
}

// doRequest is a Go 1.27+ method with its own type parameter, as in the
// template's Service.getJSON. On an older baseline, declare it as a
// package-level function instead. Interface methods cannot declare type
// parameters. Callers receive *T without type assertions.
func (c *APIClient) doRequest[T any](ctx context.Context, path string) (*T, error) {
    req, err := http.NewRequestWithContext(ctx, "GET", c.baseURL+path, nil)
    if err != nil {
        return nil, fmt.Errorf("creating request: %w", err)
    }
    req.Header.Set("Accept", "application/json")

    res, err := c.http.Do(req)
    if err != nil {
        return nil, fmt.Errorf("executing request: %w", err)
    }
    defer func() { _ = res.Body.Close() }()

    if res.StatusCode != http.StatusOK {
        return nil, fmt.Errorf("unexpected status: %d", res.StatusCode)
    }

    var result T
    if err := json.NewDecoder(res.Body).Decode(&result); err != nil {
        return nil, fmt.Errorf("decoding response: %w", err)
    }

    return &result, nil
}

// Usage
item, err := api.doRequest[Item](ctx, "/items/123")
```

This helper decodes a known response shape; it does not validate domain invariants or impose a response-size limit. Select accepted status codes for the endpoint's contract, and keep a bounded request context. The nil check after parsing also handles whitespace-only proxy configuration, for which the parser intentionally returns `(nil, nil)`.


Sources: [reference 1](https://go.dev/doc/go1.27#language), [reference 2](https://pkg.go.dev/encoding/json@go1.26.1#Decoder).
