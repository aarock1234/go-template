// Package template is a skeleton service that demonstrates how to use the
// fingerprinted HTTP client against a JSON API. Replace it with your own
// domain logic.
package template

import (
	"fmt"

	"github.com/aarock1234/go-template/pkg/client"
)

// Service fetches the network fingerprints that tls.but.sh observes for
// its client.
type Service struct {
	client *client.Client
}

// New creates a Service. Requests are routed through proxy when it is
// non-nil.
func New(proxy *client.Proxy) (*Service, error) {
	var opts []client.Option
	if proxy != nil {
		opts = append(opts, client.WithProxy(proxy))
	}

	c, err := client.New(opts...)
	if err != nil {
		return nil, fmt.Errorf("create client: %w", err)
	}

	return &Service{
		client: c,
	}, nil
}

// Close releases the idle connections held by the service's client.
func (s *Service) Close() {
	s.client.Close()
}
