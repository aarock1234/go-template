// Package cycle provides a thread-safe round-robin line rotator backed by a
// file on disk.
package cycle

import (
	"bytes"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"sync"

	_ "github.com/aarock1234/go-template/pkg/log"
)

// FileRotator reads lines from a file and serves them in round-robin order.
// All methods are safe for concurrent use.
type FileRotator struct {
	logger   *slog.Logger
	mu       sync.RWMutex
	filename string
	index    int
	lines    []string
}

// NewFileRotator creates a FileRotator that reads lines from filePath.
func NewFileRotator(filePath string) (*FileRotator, error) {
	absPath, err := filepath.Abs(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve file path: %w", err)
	}

	logger := slog.With(
		slog.String("file", filepath.Base(absPath)),
	)

	logger.Debug("creating new file rotator")

	r := &FileRotator{
		index:    0,
		filename: absPath,
		logger:   logger,
	}

	if err := r.load(); err != nil {
		return nil, err
	}

	return r, nil
}

// Reload re-reads the file from disk, replacing the current line set.
func (r *FileRotator) Reload() error {
	return r.load()
}

// Next returns the next line in round-robin order. It returns an empty
// string if the rotator has no lines.
func (r *FileRotator) Next() string {
	r.mu.Lock()
	defer r.mu.Unlock()

	if len(r.lines) == 0 {
		return ""
	}

	line := r.lines[r.index]
	r.index = (r.index + 1) % len(r.lines)

	if line != "" {
		r.logger.Debug("returning next line", slog.String("line", line), slog.Int("index", r.index-1))
	}

	return line
}

// Count returns the number of lines loaded from the file.
func (r *FileRotator) Count() int {
	r.mu.RLock()
	defer r.mu.RUnlock()

	return len(r.lines)
}

// Reset moves the rotation index back to the first line.
func (r *FileRotator) Reset() {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.index = 0
}

func (r *FileRotator) load() error {
	r.mu.Lock()
	defer r.mu.Unlock()

	fileData, err := os.ReadFile(r.filename)
	if err != nil {
		return fmt.Errorf("failed to read file %s: %w", r.filename, err)
	}

	fileData = bytes.ReplaceAll(fileData, []byte("\r"), []byte(""))

	r.lines = []string{}
	for line := range bytes.SplitSeq(fileData, []byte("\n")) {
		trimmed := bytes.TrimSpace(line)
		if len(trimmed) == 0 {
			continue
		}
		r.lines = append(r.lines, string(trimmed))
	}

	r.logger.Info("loaded lines from file", slog.Int("count", len(r.lines)))

	if len(r.lines) == 0 {
		r.logger.Warn("no lines found in file")
	}

	return nil
}
