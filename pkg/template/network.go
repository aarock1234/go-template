package template

import (
	"context"
	"fmt"
	"log/slog"
	"time"
)

// Protocol selects the HTTP version tls.but.sh observes.
type Protocol string

const (
	// HTTP1 observes an HTTP/1.1 connection.
	HTTP1 Protocol = "h1"
	// HTTP2 observes an HTTP/2 connection.
	HTTP2 Protocol = "h2"
)

// networkURL returns the endpoint that reports fingerprints for p.
func (p Protocol) networkURL() (string, error) {
	switch p {
	case HTTP1:
		return "https://h1.tls.but.sh/api/network", nil
	case HTTP2:
		return "https://h2.tls.but.sh/api/network", nil
	default:
		return "", fmt.Errorf("unsupported protocol %q", p)
	}
}

// Network is the fingerprint tls.but.sh observed for one connection. Only
// the HTTP section matching the requested protocol is set.
type Network struct {
	CollectedAt time.Time `json:"collected_at"`
	TLS         TLS       `json:"tls"`
	TCP         TCP       `json:"tcp"`
	HTTP1       *HTTP     `json:"http1"`
	HTTP2       *HTTP     `json:"http2"`
}

// TLS describes the observed TLS handshake.
type TLS struct {
	JA3        Fingerprint `json:"ja3"`
	JA4        Fingerprint `json:"ja4"`
	Negotiated Negotiated  `json:"negotiated"`
}

// Fingerprint is a raw fingerprint string and its hash.
type Fingerprint struct {
	Raw  string `json:"raw"`
	Hash string `json:"hash"`
}

// Negotiated holds the parameters the TLS handshake settled on.
type Negotiated struct {
	Version string `json:"version_name"`
	Cipher  string `json:"cipher_name"`
	ALPN    string `json:"alpn"`
}

// TCP describes the observed TCP connection.
type TCP struct {
	OS OSEstimate `json:"os_estimate"`
}

// OSEstimate is the operating system inferred from the TCP handshake.
type OSEstimate struct {
	Family     string `json:"family"`
	Confidence string `json:"confidence"`
}

// HTTP describes the observed HTTP request.
type HTTP struct {
	Protocol    string `json:"protocol"`
	HeaderOrder string `json:"header_order"`
	Wire        Wire   `json:"wire"`
}

// Wire holds connection-level details. Akamai is only reported for HTTP/2.
type Wire struct {
	Akamai Fingerprint `json:"akamai"`
}

// Network fetches the fingerprint tls.but.sh observes for a connection
// using protocol.
func (s *Service) Network(ctx context.Context, protocol Protocol) (*Network, error) {
	url, err := protocol.networkURL()
	if err != nil {
		return nil, err
	}

	network, err := s.getJSON[Network](ctx, url)
	if err != nil {
		return nil, fmt.Errorf("fetch %s network: %w", protocol, err)
	}

	return network, nil
}

// HTTP returns the observed HTTP request, or nil if none was reported.
func (n *Network) HTTP() *HTTP {
	if n.HTTP2 != nil {
		return n.HTTP2
	}

	return n.HTTP1
}

// LogValue reports the fingerprint hashes and negotiated protocol.
func (n *Network) LogValue() slog.Value {
	attrs := []slog.Attr{
		slog.String("tls", n.TLS.Negotiated.Version),
		slog.String("alpn", n.TLS.Negotiated.ALPN),
		slog.String("ja3", n.TLS.JA3.Hash),
		slog.String("ja4", n.TLS.JA4.Hash),
		slog.String("os", n.TCP.OS.Family),
	}

	if request := n.HTTP(); request != nil {
		attrs = append(attrs, slog.String("header_order", request.HeaderOrder))

		if request.Wire.Akamai.Hash != "" {
			attrs = append(attrs, slog.String("akamai", request.Wire.Akamai.Hash))
		}
	}

	return slog.GroupValue(attrs...)
}
