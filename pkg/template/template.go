// Package template is a skeleton service that demonstrates how to wire an HTTP
// client and database together. Replace it with your own domain logic.
package template

import (
	"github.com/aarock1234/go-template/pkg/client"
	"github.com/aarock1234/go-template/pkg/db"
)

// Template orchestrates HTTP requests and database access for a single domain.
type Template struct {
	db     *db.DB
	client *client.Client
}

// New creates a Template with an HTTP client configured for the given proxy.
func New(db *db.DB, proxy *client.Proxy) (*Template, error) {
	var opts []client.Option
	if proxy != nil {
		opts = append(opts, client.WithProxy(proxy))
	}

	c, err := client.New(opts...)
	if err != nil {
		return nil, err
	}

	return &Template{
		db:     db,
		client: c,
	}, nil
}
