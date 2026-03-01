package main

import (
	"fmt"
	"log/slog"
	"os"
	"os/exec"
	"strings"

	"gopkg.in/yaml.v3"
)

// removeDir deletes a directory tree. Returns nil if the path does not exist.
func removeDir(path string) error {
	if err := os.RemoveAll(path); err != nil {
		return fmt.Errorf("remove directory %s: %w", path, err)
	}
	slog.Info("removed directory", "path", path)
	return nil
}

// removeComposeService removes a named service from a Docker Compose file
// using YAML parsing to preserve structure.
func removeComposeService(path, service string) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("read compose file: %w", err)
	}

	var doc map[string]any
	if err := yaml.Unmarshal(data, &doc); err != nil {
		return fmt.Errorf("parse compose file: %w", err)
	}

	services, ok := doc["services"].(map[string]any)
	if !ok {
		return fmt.Errorf("compose file missing services key")
	}

	if _, exists := services[service]; !exists {
		return nil
	}

	delete(services, service)

	out, err := yaml.Marshal(doc)
	if err != nil {
		return fmt.Errorf("marshal compose file: %w", err)
	}

	if err := os.WriteFile(path, out, 0644); err != nil {
		return fmt.Errorf("write compose file: %w", err)
	}

	slog.Info("removed compose service", "service", service)
	return nil
}

// removeMarkedSections removes all lines between "# [tag]" and "# [/tag]"
// markers (inclusive) from the file at path. Multiple sections with the same
// tag are all removed. Returns nil if the file contains no matching markers.
func removeMarkedSections(path, tag string) error {
	open := "# [" + tag + "]"
	close := "# [/" + tag + "]"

	data, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("read %s: %w", path, err)
	}

	lines := strings.Split(string(data), "\n")
	var out []string
	skip := false

	for _, line := range lines {
		trimmed := strings.TrimSpace(line)

		if trimmed == open {
			skip = true
			continue
		}

		if trimmed == close {
			skip = false
			continue
		}

		if !skip {
			out = append(out, line)
		}
	}

	if err := os.WriteFile(path, []byte(strings.Join(out, "\n")), 0644); err != nil {
		return fmt.Errorf("write %s: %w", path, err)
	}

	slog.Info("removed marked sections", "path", path, "tag", tag)
	return nil
}

// removeSelf deletes the cmd/setup directory and its marked sections in the
// Makefile, so go mod tidy will also strip any setup-only dependencies.
func removeSelf() {
	if err := removeDir("cmd/setup"); err != nil {
		slog.Error("remove setup directory", "error", err)
	}
	if err := removeMarkedSections("Makefile", "setup"); err != nil {
		slog.Error("remove setup makefile section", "error", err)
	}
}

// tidyModules runs go mod tidy to prune unused dependencies.
func tidyModules() {
	cmd := exec.Command("go", "mod", "tidy")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		slog.Error("go mod tidy", "error", err)
	}
}
