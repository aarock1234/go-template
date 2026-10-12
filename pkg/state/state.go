// Package state provides generic, file-backed JSON state persistence with
// cross-process locking and crash-safe saves.
package state

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sync"
)

// File persists state of type T as JSON at a path.
//
// While open, a File holds an exclusive lock on the companion file
// "<path>.lock", so only one process uses the state at a time. Save writes
// the new state to a temporary file, flushes it to disk, and renames it over
// the state file, so a failed or interrupted save leaves the previous state
// intact. The rename is atomic on POSIX filesystems and a best-effort
// replacement on Windows. Methods are safe for concurrent use.
type File[T any] struct {
	mu   sync.Mutex
	path string
	lock *os.File
}

// Open acquires a non-blocking exclusive lock for the state at path and
// returns a handle for loading and saving it. If another process holds the
// lock, Open returns an error immediately. Call Close to release the lock;
// the lock file itself is left in place.
func Open[T any](path string) (*File[T], error) {
	lock, err := os.OpenFile(path+".lock", os.O_CREATE|os.O_RDWR, 0o600)
	if err != nil {
		return nil, fmt.Errorf("open state lock: %w", err)
	}

	if err := lockFile(lock); err != nil {
		_ = lock.Close()

		return nil, fmt.Errorf("acquire state lock: %w", err)
	}

	return &File[T]{
		path: path,
		lock: lock,
	}, nil
}

// Load reads the persisted state. It returns (nil, nil) when no state has
// been saved yet.
func (f *File[T]) Load() (*T, error) {
	f.mu.Lock()
	defer f.mu.Unlock()

	data, err := os.ReadFile(f.path)
	if errors.Is(err, fs.ErrNotExist) {
		return nil, nil
	}

	if err != nil {
		return nil, fmt.Errorf("read state file: %w", err)
	}

	if len(data) == 0 {
		return nil, nil
	}

	var v T
	if err := json.Unmarshal(data, &v); err != nil {
		return nil, fmt.Errorf("decode state file: %w", err)
	}

	return &v, nil
}

// Save replaces the persisted state with v. It returns only after the new
// state is flushed to disk.
func (f *File[T]) Save(v T) error {
	data, err := json.Marshal(v)
	if err != nil {
		return fmt.Errorf("encode state: %w", err)
	}

	f.mu.Lock()
	defer f.mu.Unlock()

	return replaceFile(f.path, data)
}

// Close releases the lock. The File must not be used afterward.
func (f *File[T]) Close() error {
	f.mu.Lock()
	defer f.mu.Unlock()

	return errors.Join(unlockFile(f.lock), f.lock.Close())
}

// replaceFile writes data to a temporary file beside path, flushes it, and
// renames it over path. The temporary file is removed if any step fails.
func replaceFile(path string, data []byte) (err error) {
	dir := filepath.Dir(path)

	tmp, err := os.CreateTemp(dir, filepath.Base(path)+".*.tmp")
	if err != nil {
		return fmt.Errorf("create temporary state file: %w", err)
	}

	defer func() {
		if err != nil {
			_ = tmp.Close()
			_ = os.Remove(tmp.Name())
		}
	}()

	if _, err := tmp.Write(data); err != nil {
		return fmt.Errorf("write temporary state file: %w", err)
	}

	if err := tmp.Sync(); err != nil {
		return fmt.Errorf("sync temporary state file: %w", err)
	}

	if err := tmp.Close(); err != nil {
		return fmt.Errorf("close temporary state file: %w", err)
	}

	if err := os.Rename(tmp.Name(), path); err != nil {
		return fmt.Errorf("replace state file: %w", err)
	}

	if err := syncDir(dir); err != nil {
		return fmt.Errorf("sync state directory: %w", err)
	}

	return nil
}
