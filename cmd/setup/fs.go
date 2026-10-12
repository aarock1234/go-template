package main

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"strings"
)

// markedFile is a project file parsed for feature markers.
type markedFile struct {
	path  string
	crlf  bool // whether the file is written back with CRLF line endings
	lines []markedLine
}

// loadMarkedFiles reads and parses each file in paths, skipping files that
// do not exist. It reads everything before setup changes anything, so a
// malformed marker aborts setup with the project untouched.
func loadMarkedFiles(paths []string) ([]markedFile, error) {
	var files []markedFile

	for _, path := range paths {
		data, err := os.ReadFile(path)
		if errors.Is(err, fs.ErrNotExist) {
			continue
		}

		if err != nil {
			return nil, fmt.Errorf("read %s: %w", path, err)
		}

		// Parse with "\n" line endings. A file using any CRLF line ending is
		// written back with CRLF throughout.
		text := string(data)
		crlf := strings.Contains(text, "\r\n")

		lines, err := parseMarkers(strings.ReplaceAll(text, "\r\n", "\n"))
		if err != nil {
			return nil, fmt.Errorf("parse markers in %s: %w", path, err)
		}

		file := markedFile{
			path:  path,
			crlf:  crlf,
			lines: lines,
		}
		files = append(files, file)
	}

	return files, nil
}

// rewriteFiles rewrites each file without the sections tagged with removed
// and without any remaining markers. Individual failures are joined.
func rewriteFiles(files []markedFile, removed []tag) error {
	var errs []error

	for _, file := range files {
		data := render(file.lines, removed)
		if file.crlf {
			data = strings.ReplaceAll(data, "\n", "\r\n")
		}

		if err := os.WriteFile(file.path, []byte(data), 0o644); err != nil {
			errs = append(errs, fmt.Errorf("write %s: %w", file.path, err))
		}
	}

	return errors.Join(errs...)
}

// removePaths deletes each file or directory tree in paths, ignoring paths
// that do not exist. Individual failures are joined.
func removePaths(paths []string) error {
	var errs []error

	for _, path := range paths {
		if err := os.RemoveAll(path); err != nil {
			errs = append(errs, fmt.Errorf("remove %s: %w", path, err))
		}
	}

	return errors.Join(errs...)
}
