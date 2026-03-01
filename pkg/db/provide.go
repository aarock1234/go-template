package db

import (
	"context"
	"fmt"

	"github.com/samber/do/v2"
)

// Provide registers the DB constructor with the injector.
func Provide(i do.Injector) {
	do.Provide(i, func(i do.Injector) (*DB, error) {
		cfg := do.MustInvoke[*Config](i)

		db, err := New(context.Background(), cfg.DatabaseURL)
		if err != nil {
			return nil, fmt.Errorf("db: %w", err)
		}

		return db, nil
	})
}
