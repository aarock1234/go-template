// Package env loads environment variables from a .env file into the process
// environment and populates a typed configuration struct from them.
// Existing process environment variables take precedence over file-defined
// values.
package env

import (
	"bufio"
	"errors"
	"fmt"
	"io/fs"
	"log/slog"
	"os"
	"strings"
	"sync"
)

var (
	mu     sync.Mutex
	loaded = make(map[string]error)
)

// Load reads KEY=value lines from a .env file and sets them in the process
// environment. Existing variables are not overwritten, a missing file is
// not an error, and repeated calls with the same path return the cached
// result.
//
// Blank lines and lines starting with # are ignored, an optional "export "
// prefix is allowed, and values may be wrapped in matching single or double
// quotes. If no path is provided, ".env" is used.
func Load(path ...string) error {
	filename := ".env"
	if len(path) > 0 && path[0] != "" {
		filename = path[0]
	}

	mu.Lock()
	defer mu.Unlock()

	if err, ok := loaded[filename]; ok {
		return err
	}

	err := load(filename)
	loaded[filename] = err

	return err
}

func load(filename string) error {
	vars, err := readFile(filename)
	if errors.Is(err, fs.ErrNotExist) {
		return nil
	}

	if err != nil {
		return err
	}

	for key, value := range vars {
		if _, exists := os.LookupEnv(key); exists {
			continue
		}

		if err := os.Setenv(key, value); err != nil {
			return fmt.Errorf("set %s: %w", key, err)
		}
	}

	return nil
}

// readFile parses the variables defined in filename. Later definitions of
// a key replace earlier ones.
func readFile(filename string) (map[string]string, error) {
	f, err := os.Open(filename)
	if err != nil {
		return nil, fmt.Errorf("open %s: %w", filename, err)
	}
	defer func() { _ = f.Close() }()

	vars := make(map[string]string)

	scanner := bufio.NewScanner(f)
	for lineNumber := 1; scanner.Scan(); lineNumber++ {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		key, value, ok := strings.Cut(strings.TrimPrefix(line, "export "), "=")
		key = strings.TrimSpace(key)

		if !ok || key == "" {
			slog.Warn("skipping invalid env line",
				slog.String("file", filename),
				slog.Int("line", lineNumber),
			)

			continue
		}

		vars[key] = unquote(strings.TrimSpace(value))
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("scan %s: %w", filename, err)
	}

	return vars, nil
}

// unquote removes one pair of matching single or double quotes around value.
func unquote(value string) string {
	if len(value) >= 2 && (value[0] == '"' || value[0] == '\'') && value[0] == value[len(value)-1] {
		return value[1 : len(value)-1]
	}

	return value
}
