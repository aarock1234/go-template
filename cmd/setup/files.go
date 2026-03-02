package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"os/exec"
	"strings"
)

// removeDir deletes a directory tree. Returns nil if the path does not exist.
func removeDir(path string) error {
	if err := os.RemoveAll(path); err != nil {
		return fmt.Errorf("remove directory %s: %w", path, err)
	}
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

	return nil
}

// removeSelf deletes the cmd/setup directory and its marked sections in the
// Makefile, so go mod tidy will also strip any setup-only dependencies.
func removeSelf(ctx context.Context) {
	if err := removeDir("cmd/setup"); err != nil {
		slog.ErrorContext(ctx, "remove setup directory", "error", err)
	}
	if err := removeMarkedSections("Makefile", "setup"); err != nil {
		slog.ErrorContext(ctx, "remove setup makefile section", "error", err)
	}
	slog.InfoContext(ctx, "removed setup command")
}

// tidyModules runs go mod tidy to prune unused dependencies.
func tidyModules(ctx context.Context) {
	cmd := exec.Command("go", "mod", "tidy")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		slog.ErrorContext(ctx, "go mod tidy", "error", err)
		return
	}
	slog.InfoContext(ctx, "cleaned go module dependencies")
}
