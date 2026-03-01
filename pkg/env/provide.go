package env

import (
	"fmt"

	"github.com/samber/do/v2"

	"go-template/pkg/db"
)

// Provide registers the config provider with the injector.
func Provide(i do.Injector) {
	do.Provide(i, func(i do.Injector) (*db.Config, error) {
		if err := Load(); err != nil {
			return nil, fmt.Errorf("env: load: %w", err)
		}

		var cfg struct {
			DatabaseURL string `env:"DATABASE_URL"`
		}

		if err := Validate(&cfg); err != nil {
			return nil, fmt.Errorf("env: validate: %w", err)
		}

		return &db.Config{DatabaseURL: cfg.DatabaseURL}, nil
	})
}
