// Package cycle provides a thread-safe round-robin line rotator backed by a
// file on disk.
package cycle

import (
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"sync"
)

// File serves the non-blank lines of a file in round-robin order. Lines are
// trimmed of surrounding whitespace. Its methods are safe for concurrent
// use.
type File struct {
	logger   *slog.Logger
	filename string

	mu    sync.Mutex
	lines []string
	index int
}

// New creates a File that serves the lines of the file at path.
func New(path string) (*File, error) {
	absPath, err := filepath.Abs(path)
	if err != nil {
		return nil, fmt.Errorf("resolve path: %w", err)
	}

	f := &File{
		logger:   slog.With(slog.String("file", filepath.Base(absPath))),
		filename: absPath,
	}

	if err := f.Reload(); err != nil {
		return nil, err
	}

	return f, nil
}

// Reload re-reads the file and replaces the current lines. Rotation keeps
// its position, restarting at the first line if the file got shorter.
func (f *File) Reload() error {
	lines, err := readLines(f.filename)
	if err != nil {
		return err
	}

	f.mu.Lock()
	f.lines = lines
	if f.index >= len(lines) {
		f.index = 0
	}
	f.mu.Unlock()

	if len(lines) == 0 {
		f.logger.Warn("no lines found in file")

		return nil
	}

	f.logger.Debug("loaded lines from file", slog.Int("count", len(lines)))

	return nil
}

// Next returns the next line in round-robin order. It reports false if the
// file has no lines.
func (f *File) Next() (string, bool) {
	f.mu.Lock()
	defer f.mu.Unlock()

	if len(f.lines) == 0 {
		return "", false
	}

	line := f.lines[f.index]
	f.index = (f.index + 1) % len(f.lines)

	return line, true
}

// Count returns the number of lines loaded from the file.
func (f *File) Count() int {
	f.mu.Lock()
	defer f.mu.Unlock()

	return len(f.lines)
}

// Reset moves the rotation back to the first line.
func (f *File) Reset() {
	f.mu.Lock()
	defer f.mu.Unlock()

	f.index = 0
}

// readLines returns the trimmed, non-blank lines of the file at path.
func readLines(path string) ([]string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read lines: %w", err)
	}

	var lines []string

	for line := range strings.Lines(string(data)) {
		if trimmed := strings.TrimSpace(line); trimmed != "" {
			lines = append(lines, trimmed)
		}
	}

	return lines, nil
}
