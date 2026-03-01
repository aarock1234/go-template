package main

import (
	"fmt"
	"log/slog"
	"os"
	"os/exec"
	"slices"
	"strings"

	"gopkg.in/yaml.v3"
)

// removeDir deletes a directory tree. No-op if it does not exist.
func removeDir(path string) {
	if err := os.RemoveAll(path); err != nil {
		slog.Error("remove directory", "path", path, "error", err)
		return
	}
	slog.Info("removed directory", "path", path)
}

// removeComposeService removes a named service from a compose file using
// proper YAML parsing.
func removeComposeService(path, service string) {
	data, err := os.ReadFile(path)
	if err != nil {
		slog.Error("read compose file", "path", path, "error", err)
		return
	}

	var doc map[string]any
	if err := yaml.Unmarshal(data, &doc); err != nil {
		slog.Error("parse compose file", "path", path, "error", err)
		return
	}

	services, ok := doc["services"].(map[string]any)
	if !ok {
		slog.Error("compose file missing services key", "path", path)
		return
	}

	if _, exists := services[service]; !exists {
		return
	}

	delete(services, service)

	out, err := yaml.Marshal(doc)
	if err != nil {
		slog.Error("marshal compose file", "error", err)
		return
	}

	if err := os.WriteFile(path, out, 0644); err != nil {
		slog.Error("write compose file", "path", path, "error", err)
		return
	}

	slog.Info("removed compose service", "service", service)
}

// removeMatchingLines drops every line that contains at least one of the
// provided substrings.
func removeMatchingLines(path string, substrings ...string) {
	if err := rewriteLines(path, func(lines []string) []string {
		return slices.DeleteFunc(lines, func(line string) bool {
			for _, sub := range substrings {
				if strings.Contains(line, sub) {
					return true
				}
			}
			return false
		})
	}); err != nil {
		slog.Error("remove matching lines", "path", path, "error", err)
		return
	}
	slog.Info("cleaned env file", "path", path)
}

// removeMakeTargets strips named target blocks and their .PHONY entries
// from a Makefile.
func removeMakeTargets(path string, targets ...string) {
	drop := make(map[string]bool, len(targets))
	for _, t := range targets {
		drop[t] = true
	}

	if err := rewriteLines(path, func(lines []string) []string {
		var out []string
		skip := false
		for _, line := range lines {
			if strings.HasPrefix(line, ".PHONY:") {
				parts := slices.DeleteFunc(strings.Fields(line), func(p string) bool {
					return p != ".PHONY:" && drop[p]
				})
				out = append(out, strings.Join(parts, " "))
				continue
			}

			name, _, hasColon := strings.Cut(line, ":")
			if hasColon && drop[strings.TrimSpace(name)] {
				skip = true
				continue
			}

			if skip && !strings.HasPrefix(line, "\t") {
				if strings.TrimSpace(line) == "" {
					continue
				}
				skip = false
			}

			if !skip {
				out = append(out, line)
			}
		}
		return out
	}); err != nil {
		slog.Error("remove make targets", "path", path, "error", err)
		return
	}
	slog.Info("removed make targets", "path", path, "targets", targets)
}

// tidyModules runs go mod tidy to prune unused dependencies.
func tidyModules() {
	cmd := exec.Command("go", "mod", "tidy")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		slog.Error("go mod tidy", "error", err)
		return
	}
	slog.Info("cleaned go module dependencies")
}

// removeSelf deletes the cmd/setup directory and its Makefile target,
// so go mod tidy will also strip any setup-only dependencies.
func removeSelf() {
	removeDir("cmd/setup")
	removeMakeTargets("Makefile", "setup")
	slog.Info("removed setup command")
}

// rewriteLines reads a file, transforms its lines, and writes the result back.
func rewriteLines(path string, fn func([]string) []string) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("read %s: %w", path, err)
	}

	lines := fn(strings.Split(string(data), "\n"))

	if err := os.WriteFile(path, []byte(strings.Join(lines, "\n")), 0644); err != nil {
		return fmt.Errorf("write %s: %w", path, err)
	}

	return nil
}
