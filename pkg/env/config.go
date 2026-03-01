package env

import (
	"fmt"
	"os"
	"reflect"
	"strings"
)

// Config holds all environment variables required by the application.
// Add a field with an `env` tag to register a new variable.
// Use `required` to make it mandatory:
//
//	LogLevel string `env:"LOG_LEVEL"`              // optional
//	APIURL   string `env:"API_URL,required"`        // required
type Config struct {
	DatabaseURL string `env:"DATABASE_URL,required"`
}

// New loads the .env file and returns a populated Config.
func New() (*Config, error) {
	if err := Load(); err != nil {
		return nil, fmt.Errorf("env: load: %w", err)
	}

	var cfg Config
	if err := populate(&cfg); err != nil {
		return nil, err
	}

	return &cfg, nil
}

// populate reads env vars into cfg using struct tags.
func populate(cfg *Config) error {
	v := reflect.ValueOf(cfg).Elem()
	t := v.Type()

	var missing []string

	for i := range t.NumField() {
		field := t.Field(i)
		tag := field.Tag.Get("env")
		if tag == "" {
			continue
		}

		parts := strings.SplitN(tag, ",", 2)
		key := parts[0]
		required := len(parts) > 1 && parts[1] == "required"

		val := os.Getenv(key)
		if val == "" && required {
			missing = append(missing, key)
			continue
		}

		if field.Type.Kind() == reflect.String {
			v.Field(i).SetString(val)
		}
	}

	if len(missing) > 0 {
		return fmt.Errorf("env: missing required variables: %s", strings.Join(missing, ", "))
	}

	return nil
}
