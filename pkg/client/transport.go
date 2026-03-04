package client

import (
	"bufio"
	"context"
	"encoding/base64"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"strings"

	utls "github.com/refraction-networking/utls"
	"golang.org/x/net/proxy"
)

// transport implements http.RoundTripper with TLS fingerprinting,
// HTTP/2 fingerprint control, and header ordering.
type transport struct {
	clientHelloID         utls.ClientHelloID
	h2Profile             *H2Profile
	defaultHeaders        http.Header
	proxyURL              *url.URL
	disableKeepAlives     bool
	insecureSkipVerify    bool
	disableSessionTickets bool
}

var _ http.RoundTripper = (*transport)(nil)

// RoundTrip executes a single HTTP transaction. It dials a TLS connection
// using utls for fingerprinting, checks the negotiated ALPN protocol,
// and routes to the appropriate HTTP/1.1 or HTTP/2 handler.
func (t *transport) RoundTrip(req *http.Request) (*http.Response, error) {
	if req.URL.Scheme != "https" {
		return nil, fmt.Errorf("unsupported scheme %q, only https is supported", req.URL.Scheme)
	}

	// Apply default headers (don't overwrite existing)
	for key, values := range t.defaultHeaders {
		if req.Header.Get(key) == "" && !isMagicKey(key) {
			for _, v := range values {
				req.Header.Set(key, v)
			}
		}
	}

	addr := req.URL.Host
	if !strings.Contains(addr, ":") {
		addr += ":443"
	}

	conn, proto, err := t.dialTLS(req.Context(), addr)
	if err != nil {
		return nil, fmt.Errorf("dialing tls: %w", err)
	}

	switch proto {
	case "h2":
		resp, err := t.roundTripH2(conn, req)
		if err != nil {
			conn.Close()

			return nil, err
		}

		return resp, nil
	default:
		resp, err := t.roundTripH1(conn, req)
		if err != nil {
			conn.Close()

			return nil, err
		}

		return resp, nil
	}
}

// dialTLS establishes a TLS connection to addr using utls for fingerprinting.
// It returns the connection and the negotiated ALPN protocol.
func (t *transport) dialTLS(ctx context.Context, addr string) (net.Conn, string, error) {
	host, _, err := net.SplitHostPort(addr)
	if err != nil {
		return nil, "", fmt.Errorf("splitting host port: %w", err)
	}

	tcpConn, err := t.dialTCP(ctx, addr)
	if err != nil {
		return nil, "", fmt.Errorf("dialing tcp: %w", err)
	}

	tlsConfig := &utls.Config{
		ServerName:             host,
		InsecureSkipVerify:     t.insecureSkipVerify,
		SessionTicketsDisabled: t.disableSessionTickets,
	}

	if t.disableSessionTickets {
		tlsConfig.ClientSessionCache = nil
	}

	tlsConn := utls.UClient(tcpConn, tlsConfig, t.clientHelloID)

	if err := tlsConn.HandshakeContext(ctx); err != nil {
		tcpConn.Close()

		return nil, "", fmt.Errorf("tls handshake: %w", err)
	}

	proto := tlsConn.ConnectionState().NegotiatedProtocol

	return tlsConn, proto, nil
}

// dialTCP establishes a TCP connection, optionally through a proxy.
func (t *transport) dialTCP(ctx context.Context, addr string) (net.Conn, error) {
	if t.proxyURL == nil {
		return (&net.Dialer{}).DialContext(ctx, "tcp", addr)
	}

	switch t.proxyURL.Scheme {
	case "http", "https":
		return t.dialHTTPProxy(ctx, addr)
	case "socks5", "socks5h":
		return t.dialSOCKS5Proxy(ctx, addr)
	default:
		return nil, fmt.Errorf("unsupported proxy scheme: %s", t.proxyURL.Scheme)
	}
}

// dialHTTPProxy establishes a connection through an HTTP CONNECT proxy.
func (t *transport) dialHTTPProxy(ctx context.Context, targetAddr string) (net.Conn, error) {
	proxyAddr := t.proxyURL.Host

	conn, err := (&net.Dialer{}).DialContext(ctx, "tcp", proxyAddr)
	if err != nil {
		return nil, fmt.Errorf("connecting to proxy: %w", err)
	}

	connectReq := &http.Request{
		Method: http.MethodConnect,
		URL:    &url.URL{Opaque: targetAddr},
		Host:   targetAddr,
		Header: make(http.Header),
	}

	if t.proxyURL.User != nil {
		user := t.proxyURL.User.Username()
		pass, _ := t.proxyURL.User.Password()
		auth := base64.StdEncoding.EncodeToString([]byte(user + ":" + pass))
		connectReq.Header.Set("Proxy-Authorization", "Basic "+auth)
	}

	if err := connectReq.Write(conn); err != nil {
		conn.Close()

		return nil, fmt.Errorf("writing connect request: %w", err)
	}

	resp, err := http.ReadResponse(bufio.NewReader(conn), connectReq)
	if err != nil {
		conn.Close()

		return nil, fmt.Errorf("reading connect response: %w", err)
	}
	resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		conn.Close()

		return nil, fmt.Errorf("proxy connect returned status %d", resp.StatusCode)
	}

	return conn, nil
}

// dialSOCKS5Proxy establishes a connection through a SOCKS5 proxy.
func (t *transport) dialSOCKS5Proxy(ctx context.Context, targetAddr string) (net.Conn, error) {
	var auth *proxy.Auth
	if t.proxyURL.User != nil {
		user := t.proxyURL.User.Username()
		pass, _ := t.proxyURL.User.Password()
		auth = &proxy.Auth{
			User:     user,
			Password: pass,
		}
	}

	dialer, err := proxy.SOCKS5("tcp", t.proxyURL.Host, auth, &net.Dialer{})
	if err != nil {
		return nil, fmt.Errorf("creating socks5 dialer: %w", err)
	}

	// Use ContextDialer if available (modern x/net)
	if cd, ok := dialer.(proxy.ContextDialer); ok {
		conn, err := cd.DialContext(ctx, "tcp", targetAddr)
		if err != nil {
			return nil, fmt.Errorf("socks5 dial: %w", err)
		}

		return conn, nil
	}

	conn, err := dialer.Dial("tcp", targetAddr)
	if err != nil {
		return nil, fmt.Errorf("socks5 dial: %w", err)
	}

	return conn, nil
}
