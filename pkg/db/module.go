package db

import (
	"context"
	"fmt"

	"go.uber.org/fx"

	"go-template/pkg/env"
)

// Module provides the database connection to the fx container.
var Module = fx.Module("db",
	fx.Provide(func(lc fx.Lifecycle, cfg *env.Config) (*DB, error) {
		db, err := New(context.Background(), cfg.DatabaseURL)
		if err != nil {
			return nil, fmt.Errorf("db: %w", err)
		}

		lc.Append(fx.StopHook(db.Close))
		return db, nil
	}),
)
