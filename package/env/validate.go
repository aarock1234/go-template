package env

import (
	"fmt"
	"os"
	"reflect"
	"strings"
)

// Validate checks that all environment variables referenced by `env` struct
// tags are set in the process environment. Tagged struct fields are populated
// with their corresponding values.
//
//	var config struct {
//		DatabaseURL string `env:"DATABASE_URL"`
//		LogLevel    string `env:"LOG_LEVEL"`
//	}
//
//	if err := env.Validate(&config); err != nil {
//		// err lists all missing variables
//	}
func Validate(config any) error {
	v := reflect.ValueOf(config)
	if v.Kind() != reflect.Pointer || v.Elem().Kind() != reflect.Struct {
		return fmt.Errorf("env.Validate: expected pointer to struct, got %T", config)
	}

	v = v.Elem()
	t := v.Type()

	var missing []string

	for i := range t.NumField() {
		field := t.Field(i)

		tag := field.Tag.Get("env")
		if tag == "" {
			continue
		}

		val, ok := os.LookupEnv(tag)
		if !ok || val == "" {
			missing = append(missing, tag)
			continue
		}

		if field.Type.Kind() == reflect.String && v.Field(i).CanSet() {
			v.Field(i).SetString(val)
		}
	}

	if len(missing) > 0 {
		return fmt.Errorf("missing required environment variables: %s", strings.Join(missing, ", "))
	}

	return nil
}
