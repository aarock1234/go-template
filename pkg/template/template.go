// Package template is a skeleton service that demonstrates how to wire an HTTP
// client and database together. Replace it with your own domain logic.
package template

import (
	"go-template/pkg/client"
	"go-template/pkg/db"
)

// Template orchestrates HTTP requests and database access for a single domain.
type Template struct {
	db     *db.DB
	client *client.Client
}

// New creates a Template with an HTTP client configured for the given proxy.
func New(db *db.DB, proxy *client.Proxy) (*Template, error) {
	client, err := client.New(proxy)
	if err != nil {
		return nil, err
	}

	return &Template{
		db:     db,
		client: client,
	}, nil
}
