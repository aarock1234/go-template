package client

import (
	"bufio"
	"errors"
	"fmt"
	"net"
	"net/url"
	"os"
	"slices"
	"strconv"
	"strings"
)

// ProxyScheme identifies the protocol used to connect through a proxy.
type ProxyScheme string

const (
	// ProxySchemeHTTP routes traffic through an HTTP CONNECT proxy.
	ProxySchemeHTTP ProxyScheme = "http"
	// ProxySchemeHTTPS routes traffic through an HTTPS CONNECT proxy.
	ProxySchemeHTTPS ProxyScheme = "https"
	// ProxySchemeSOCKS5 routes traffic through a SOCKS5 proxy with local DNS resolution.
	ProxySchemeSOCKS5 ProxyScheme = "socks5"
	// ProxySchemeSOCKS5H routes traffic through a SOCKS5 proxy with remote DNS resolution.
	ProxySchemeSOCKS5H ProxyScheme = "socks5h"
)

// String returns the scheme as its protocol string.
func (s ProxyScheme) String() string {
	return string(s)
}

// ParseProxyScheme converts a string to a ProxyScheme.
func ParseProxyScheme(scheme string) (ProxyScheme, error) {
	switch s := ProxyScheme(scheme); s {
	case ProxySchemeHTTP, ProxySchemeHTTPS, ProxySchemeSOCKS5, ProxySchemeSOCKS5H:
		return s, nil
	default:
		return "", fmt.Errorf("invalid proxy scheme %q", scheme)
	}
}

// Proxy holds a parsed proxy configuration.
type Proxy struct {
	Scheme   ProxyScheme
	Host     string
	Port     string
	Username string
	Password string
}

// URL returns the proxy as a URL suitable for Transport.Proxy, including
// its credentials.
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

// String returns the proxy URL with its password redacted, so a Proxy can
// be logged or formatted safely.
func (p *Proxy) String() string {
	return p.URL().Redacted()
}

// ParseProxy parses a "host:port" or "host:port:user:pass" string into a
// Proxy. An empty or whitespace-only string returns (nil, nil). The format
// cannot represent IPv6 hosts or credentials containing colons. Errors
// never include the input, which may contain credentials.
func ParseProxy(proxy string, scheme ProxyScheme) (*Proxy, error) {
	proxy = strings.TrimSpace(proxy)
	if proxy == "" {
		return nil, nil
	}

	fields := strings.Split(proxy, ":")
	if len(fields) != 2 && len(fields) != 4 {
		return nil, fmt.Errorf("got %d proxy fields, want 2 or 4", len(fields))
	}

	if slices.Contains(fields, "") {
		return nil, errors.New("proxy fields must not be empty")
	}

	if port, err := strconv.Atoi(fields[1]); err != nil || port < 1 || port > 65535 {
		return nil, errors.New("proxy port must be an integer between 1 and 65535")
	}

	parsed := &Proxy{
		Scheme: scheme,
		Host:   fields[0],
		Port:   fields[1],
	}

	if len(fields) == 4 {
		parsed.Username = fields[2]
		parsed.Password = fields[3]
	}

	return parsed, nil
}

// ImportProxies reads proxies from a file with one ParseProxy string per
// line, skipping blank lines.
func ImportProxies(filename string, scheme ProxyScheme) ([]*Proxy, error) {
	f, err := os.Open(filename)
	if err != nil {
		return nil, fmt.Errorf("open proxy file: %w", err)
	}
	defer func() { _ = f.Close() }()

	var proxies []*Proxy

	scanner := bufio.NewScanner(f)
	for lineNumber := 1; scanner.Scan(); lineNumber++ {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}

		proxy, err := ParseProxy(line, scheme)
		if err != nil {
			return nil, fmt.Errorf("parse proxy on line %d: %w", lineNumber, err)
		}

		proxies = append(proxies, proxy)
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("scan proxy file: %w", err)
	}

	return proxies, nil
}
