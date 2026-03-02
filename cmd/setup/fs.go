package main

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"slices"
	"strings"
)

// feature describes a set of files, directories, config sections, and compose
// services to remove when a feature is disabled.
type feature struct {
	dirs     []string
	files    []string
	sections []section
	compose  []string
}

// section identifies a tagged block inside a file (e.g. "# [tag]"..."# [/tag]").
type section struct {
	file string
	tag  string
}

// removeFeature applies all removal operations for a feature.
// Individual failures are collected and returned as a joined error.
func removeFeature(f feature) error {
	var errs []error

	for _, dir := range f.dirs {
		if err := removeDir(dir); err != nil {
			errs = append(errs, err)
		}
	}

	for _, file := range f.files {
		if err := removeFile(file); err != nil {
			errs = append(errs, err)
		}
	}

	for _, s := range f.sections {
		if err := removeMarkedSections(s.file, s.tag); err != nil {
			errs = append(errs, err)
		}
	}

	for _, svc := range f.compose {
		if err := removeComposeService("compose.yaml", svc); err != nil {
			errs = append(errs, err)
		}
	}

	return errors.Join(errs...)
}

// removeDir deletes a directory tree. Returns nil if the path does not exist.
func removeDir(path string) error {
	if err := os.RemoveAll(path); err != nil {
		return fmt.Errorf("remove directory %s: %w", path, err)
	}
	return nil
}

// removeFile deletes a single file. Returns nil if the file does not exist.
func removeFile(path string) error {
	if err := os.Remove(path); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("remove file %s: %w", path, err)
	}
	return nil
}

// removeMarkedSections removes all lines between "[tag]" and "[/tag]"
// markers (inclusive) from the file at path. Supports both # and // comment
// prefixes. Returns nil if the file contains no matching markers.
func removeMarkedSections(path, tag string) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("read %s: %w", path, err)
	}

	opens := []string{"# [" + tag + "]", "// [" + tag + "]"}
	closes := []string{"# [/" + tag + "]", "// [/" + tag + "]"}

	lines := strings.Split(string(data), "\n")
	var out []string
	skip := false

	for _, line := range lines {
		trimmed := strings.TrimSpace(line)

		if slices.Contains(opens, trimmed) {
			skip = true
			continue
		}

		if slices.Contains(closes, trimmed) {
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
func removeSelf() error {
	return removeFeature(feature{
		dirs:     []string{"cmd/setup"},
		sections: []section{{file: "Makefile", tag: "setup"}},
	})
}

// tidyModules runs go mod tidy to prune unused dependencies.
func tidyModules() error {
	cmd := exec.Command("go", "mod", "tidy")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("go mod tidy: %w", err)
	}

	return nil
}
