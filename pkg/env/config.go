package env

import (
	"fmt"
	"os"
	"reflect"
	"strconv"
	"strings"
	"time"
)

var durationType = reflect.TypeFor[time.Duration]()

// Config holds all environment variables required by the application.
// Add a field with an `env` tag to register a new variable.
//
// Tag format:
//
//	`env:"VAR_NAME"`                — optional, zero value default
//	`env:"VAR_NAME,required"`       — must be set and non-empty
//	`env:"VAR_NAME,default=value"`  — uses value when unset
//
// Supported field types: string, int, int64, bool, time.Duration.
type Config struct {
	// [postgres]
	DatabaseURL string `env:"DATABASE_URL,required"`
	// [/postgres]
	// [server]
	Port         string        `env:"PORT,default=8080"`
	ReadTimeout  time.Duration `env:"READ_TIMEOUT,default=10s"`
	WriteTimeout time.Duration `env:"WRITE_TIMEOUT,default=10s"`
	// [/server]
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

		name, rest, hasOpts := strings.Cut(tag, ",")
		var required bool
		var defaultVal string
		if hasOpts {
			required, defaultVal = parseOpts(rest)
		}

		val := os.Getenv(name)
		if val == "" {
			if required {
				missing = append(missing, name)
				continue
			}
			if defaultVal != "" {
				val = defaultVal
			}
		}

		if val == "" {
			continue
		}

		if err := setField(v.Field(i), val); err != nil {
			return fmt.Errorf("env: set %s: %w", name, err)
		}
	}

	if len(missing) > 0 {
		return fmt.Errorf("env: missing required variables: %s", strings.Join(missing, ", "))
	}

	return nil
}

// parseOpts extracts the required flag and default value from tag options.
func parseOpts(opts string) (required bool, defaultVal string) {
	for opt := range strings.SplitSeq(opts, ",") {
		switch {
		case opt == "required":
			required = true
		case strings.HasPrefix(opt, "default="):
			defaultVal = strings.TrimPrefix(opt, "default=")
		}
	}

	return required, defaultVal
}

// setField assigns a string value to a reflect.Value based on its type.
func setField(field reflect.Value, val string) error {
	if field.Type() == durationType {
		d, err := time.ParseDuration(val)
		if err != nil {
			return fmt.Errorf("parse duration: %w", err)
		}
		field.SetInt(int64(d))

		return nil
	}

	switch field.Kind() {
	case reflect.String:
		field.SetString(val)

	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		n, err := strconv.ParseInt(val, 10, 64)
		if err != nil {
			return fmt.Errorf("parse int: %w", err)
		}
		field.SetInt(n)

	case reflect.Bool:
		b, err := strconv.ParseBool(val)
		if err != nil {
			return fmt.Errorf("parse bool: %w", err)
		}
		field.SetBool(b)

	default:
		return fmt.Errorf("unsupported type: %s", field.Kind())
	}

	return nil
}
