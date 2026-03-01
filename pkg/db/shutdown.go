package db

import "github.com/samber/do/v2"

// Shutdown implements do.ShutdownerWithError, closing the connection pool.
func (d *DB) Shutdown() error {
	d.Close()
	return nil
}

// Compile-time interface check.
var _ do.ShutdownerWithError = (*DB)(nil)
