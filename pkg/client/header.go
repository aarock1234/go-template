package client

import (
	"math/rand/v2"
	"net/http"
)

// HeaderOrderKey is a magic header key that controls the order of regular
// headers in HTTP/2 HEADERS frames and HTTP/1.1 requests. Its values specify
// header names in the desired wire order. The key itself is never sent.
const HeaderOrderKey = "Header-Order"

// PseudoHeaderOrderKey is a magic header key that overrides the browser
// profile's pseudo-header order for a single request. Values are pseudo-header
// names such as ":method", ":authority", ":scheme", and ":path".
const PseudoHeaderOrderKey = "Psuedo-Header-Order"

// isMagicKey reports whether key is a transport-internal ordering directive
// that must not be written on the wire.
func isMagicKey(key string) bool {
	return key == HeaderOrderKey || key == PseudoHeaderOrderKey
}

// headerOrder returns the regular header write order for req.
// If HeaderOrderKey is set on the request, its values are used directly.
// Otherwise keys are shuffled randomly, matching Chrome v106+ behavior.
func headerOrder(req *http.Request) []string {
	if order, ok := req.Header[HeaderOrderKey]; ok {
		return order
	}

	keys := make([]string, 0, len(req.Header))
	for k := range req.Header {
		if !isMagicKey(k) {
			keys = append(keys, k)
		}
	}

	rand.Shuffle(len(keys), func(i, j int) {
		keys[i], keys[j] = keys[j], keys[i]
	})

	return keys
}

// pseudoHeaderOrder returns the pseudo-header write order for req.
// If PseudoHeaderOrderKey is set on the request, it takes precedence
// over defaultOrder.
func pseudoHeaderOrder(req *http.Request, defaultOrder []string) []string {
	if order, ok := req.Header[PseudoHeaderOrderKey]; ok {
		return order
	}

	return defaultOrder
}
