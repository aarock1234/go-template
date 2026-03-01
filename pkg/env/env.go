// Package env loads environment variables from a .env file into the process
// environment. Existing variables take precedence over file-defined values.
package env

import (
	"bufio"
	"fmt"
	"log/slog"
	"os"
	"strings"
	"sync"
)

var (
	muLoad sync.Mutex
	loaded = make(map[string]error)
)

// Load reads environment variables from a .env file and sets them in the
// process environment. Existing environment variables take precedence.
// Repeated calls with the same file path return the cached result.
func Load(file ...string) error {
	if len(file) > 1 {
		slog.Warn("env.Load called with multiple files, only the first will be used")
	}

	filename := ".env"
	if len(file) > 0 && file[0] != "" {
		filename = file[0]
	}

	muLoad.Lock()
	defer muLoad.Unlock()

	if err, ok := loaded[filename]; ok {
		return err
	}

	err := load(filename)
	loaded[filename] = err

	return err
}

func load(filename string) error {
	_, err := os.Stat(filename)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}

		return fmt.Errorf("statting env file %q: %w", filename, err)
	}

	f, err := os.Open(filename)
	if err != nil {
		return fmt.Errorf("opening env file %q: %w", filename, err)
	}
	defer f.Close()

	var (
		envVars    = make(map[string]string)
		keyLines   = make(map[string]int)
		scanner    = bufio.NewScanner(f)
		lineNumber = 0
	)

	for scanner.Scan() {
		lineNumber++
		line := scanner.Text()
		line = strings.TrimSpace(line)

		if strings.HasPrefix(line, "#") || line == "" {
			continue
		}

		split := strings.SplitN(line, "=", 2)
		if len(split) != 2 {
			slog.Warn("ignoring invalid env line",
				slog.String("file", filename),
				slog.Int("line_number", lineNumber),
				slog.String("line", line),
			)

			continue
		}

		key := strings.TrimSpace(split[0])
		value := strings.TrimSpace(split[1])

		if len(value) > 1 && ((value[0] == '"' && value[len(value)-1] == '"') || (value[0] == '\'' && value[len(value)-1] == '\'')) {
			value = value[1 : len(value)-1]
		}

		if previousLine, exists := keyLines[key]; exists {
			slog.Warn("overwriting previous definition for key in env file",
				slog.String("file", filename),
				slog.String("key", key),
				slog.Int("previous_line", previousLine),
				slog.Int("new_line", lineNumber),
			)
		}

		envVars[key] = value
		keyLines[key] = lineNumber
	}

	if err := scanner.Err(); err != nil {
		return fmt.Errorf("scanning env file %q: %w", filename, err)
	}

	for key, value := range envVars {
		if _, exists := os.LookupEnv(key); exists {
			slog.Warn("skipping env var from file, already set in environment",
				slog.String("file", filename),
				slog.String("key", key),
				slog.Int("line_defined", keyLines[key]),
			)

			continue
		}

		err = os.Setenv(key, value)
		if err != nil {
			slog.Warn("failed to set env var from file",
				slog.String("file", filename),
				slog.String("key", key),
				slog.Int("line_defined", keyLines[key]),
				slog.String("error", err.Error()),
			)
		}
	}

	return nil
}
