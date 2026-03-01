package main

import (
	"fmt"
	"log/slog"
	"os"
	"os/exec"
	"slices"
	"strings"
)

// removeDir deletes a directory tree. No-op if it does not exist.
func removeDir(path string) {
	if err := os.RemoveAll(path); err != nil {
		slog.Error("remove directory", "path", path, "error", err)
		return
	}
	slog.Info("removed directory", "path", path)
}

// removeComposeService strips a named service block from a compose file.
// A service block starts with "  <name>:" and ends at the next line with
// two-space indent (the next sibling key).
func removeComposeService(path, service string) {
	prefix := "  " + service + ":"

	if err := rewriteFile(path, func(lines []string) []string {
		var out []string
		skip := false
		for _, line := range lines {
			if strings.HasPrefix(line, prefix) {
				skip = true
				continue
			}
			if skip && len(line) > 0 && !strings.HasPrefix(line, "   ") {
				skip = false
			}
			if !skip {
				out = append(out, line)
			}
		}
		return out
	}); err != nil {
		slog.Error("remove compose service", "service", service, "error", err)
		return
	}
	slog.Info("removed compose service", "service", service)
}

// removeMatchingLines drops every line that contains at least one of the
// provided substrings.
func removeMatchingLines(path string, substrings ...string) {
	if err := rewriteFile(path, func(lines []string) []string {
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

	if err := rewriteFile(path, func(lines []string) []string {
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

// rewriteFile reads a file, transforms its lines, and writes the result back.
func rewriteFile(path string, fn func([]string) []string) error {
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
