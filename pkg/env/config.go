package env

import (
	"fmt"
	"os"
	"reflect"
	"strings"
)

// Config holds all environment variables required by the application.
// Add a field with an `env` tag to register a new variable.
//
// Tag format: `env:"VAR_NAME"` or `env:"VAR_NAME,required"`
//
// Example:
//
//	type Config struct {
//	    DatabaseURL string `env:"DATABASE_URL,required"`
//	    LogLevel    string `env:"LOG_LEVEL"`
//	}
type Config struct {
	DatabaseURL string `env:"DATABASE_URL,required"`
}

// New loads the .env file and returns a Config populated from the environment.
// Required variables that are missing or empty are returned as a single error.
func New() (*Config, error) {
	if err := Load(); err != nil {
		return nil, fmt.Errorf("env: load: %w", err)
	}

	var config Config
	if err := populate(&config); err != nil {
		return nil, err
	}

	return &config, nil
}

// populate reads environment variables into config using its struct tags.
func populate(config *Config) error {
	v := reflect.ValueOf(config).Elem()
	t := v.Type()

	var missing []string

	for i := range t.NumField() {
		field := t.Field(i)

		tag := field.Tag.Get("env")
		if tag == "" {
			continue
		}

		name, opts, _ := strings.Cut(tag, ",")
		required := opts == "required"

		val := os.Getenv(name)
		if val == "" && required {
			missing = append(missing, name)
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
