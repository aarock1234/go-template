package env

import (
	"encoding"
	"fmt"
	"os"
	"reflect"
	"strconv"
	"strings"
	"time"
)

var durationType = reflect.TypeFor[time.Duration]()

// New loads the .env file and returns a T populated from the environment.
// T must be a struct with at least one field tagged `env:"VAR_NAME"`.
//
// Tag format:
//
//	`env:"VAR_NAME"`                — optional, zero value default
//	`env:"VAR_NAME,required"`       — must be set and non-empty
//	`env:"VAR_NAME,default=value"`  — uses value when unset or empty
//
// A default value cannot contain a comma. Supported field types are
// strings, booleans, integers, unsigned integers, floats, time.Duration,
// and types whose pointer implements encoding.TextUnmarshaler, such as
// slog.Level.
func New[T any](path ...string) (*T, error) {
	if err := Load(path...); err != nil {
		return nil, fmt.Errorf("env: load: %w", err)
	}

	var config T
	if err := populate(&config); err != nil {
		return nil, err
	}

	return &config, nil
}

// populate reads environment variables into the struct config points to,
// using its field tags. Every missing required variable is reported.
func populate(config any) error {
	v := reflect.ValueOf(config)
	if v.Kind() != reflect.Pointer || v.Elem().Kind() != reflect.Struct {
		return fmt.Errorf("env: expected pointer to struct, got %T", config)
	}

	v = v.Elem()
	t := v.Type()

	var (
		tagged  bool
		missing []string
	)

	for field := range t.Fields() {
		tag := field.Tag.Get("env")
		if tag == "" {
			continue
		}

		tagged = true

		name, options, _ := strings.Cut(tag, ",")
		required, defaultValue := parseOptions(options)

		value := os.Getenv(name)
		if value == "" {
			value = defaultValue
		}

		if value == "" {
			if required {
				missing = append(missing, name)
			}

			continue
		}

		if err := setField(v.FieldByIndex(field.Index), value); err != nil {
			return fmt.Errorf("env: set %s: %w", name, err)
		}
	}

	if !tagged {
		return fmt.Errorf("env: %s has no fields with env tags", t.Name())
	}

	if len(missing) > 0 {
		return fmt.Errorf("env: missing required variables: %s", strings.Join(missing, ", "))
	}

	return nil
}

// parseOptions extracts the required flag and default value from tag
// options.
func parseOptions(options string) (required bool, defaultValue string) {
	for option := range strings.SplitSeq(options, ",") {
		if option == "required" {
			required = true
		}

		if value, ok := strings.CutPrefix(option, "default="); ok {
			defaultValue = value
		}
	}

	return required, defaultValue
}

// setField parses value into field according to the field's type.
func setField(field reflect.Value, value string) error {
	if field.Type() == durationType {
		d, err := time.ParseDuration(value)
		if err != nil {
			return fmt.Errorf("parse duration: %w", err)
		}

		field.SetInt(int64(d))

		return nil
	}

	if unmarshaler, ok := field.Addr().Interface().(encoding.TextUnmarshaler); ok {
		if err := unmarshaler.UnmarshalText([]byte(value)); err != nil {
			return fmt.Errorf("parse %s: %w", field.Type(), err)
		}

		return nil
	}

	switch field.Kind() {
	case reflect.String:
		field.SetString(value)
	case reflect.Bool:
		b, err := strconv.ParseBool(value)
		if err != nil {
			return fmt.Errorf("parse bool: %w", err)
		}

		field.SetBool(b)
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		n, err := strconv.ParseInt(value, 10, field.Type().Bits())
		if err != nil {
			return fmt.Errorf("parse int: %w", err)
		}

		field.SetInt(n)
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		n, err := strconv.ParseUint(value, 10, field.Type().Bits())
		if err != nil {
			return fmt.Errorf("parse uint: %w", err)
		}

		field.SetUint(n)
	case reflect.Float32, reflect.Float64:
		f, err := strconv.ParseFloat(value, field.Type().Bits())
		if err != nil {
			return fmt.Errorf("parse float: %w", err)
		}

		field.SetFloat(f)
	default:
		return fmt.Errorf("unsupported type %s", field.Type())
	}

	return nil
}
