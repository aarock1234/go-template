//go:build windows

package state

import (
	"os"

	"golang.org/x/sys/windows"
)

func lockFile(f *os.File) error {
	var overlapped windows.Overlapped

	return windows.LockFileEx(
		windows.Handle(f.Fd()),
		windows.LOCKFILE_EXCLUSIVE_LOCK|windows.LOCKFILE_FAIL_IMMEDIATELY,
		0,
		1, 0,
		&overlapped,
	)
}

func unlockFile(f *os.File) error {
	var overlapped windows.Overlapped

	return windows.UnlockFileEx(
		windows.Handle(f.Fd()),
		0,
		1, 0,
		&overlapped,
	)
}

// syncDir is a no-op: Windows cannot flush a directory handle, and NTFS
// journals the rename itself.
func syncDir(string) error {
	return nil
}
