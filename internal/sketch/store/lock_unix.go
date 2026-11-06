//go:build unix

package store

import (
	"os"
	"syscall"
)

// lock holds an exclusive lock on the directory dir until the returned function runs. It locks a directory, not
// the sketch file, because a save renames a new file over the sketch, and a directory leaves nothing in the vault.
func lock(dir string) (func(), error) {
	handle, err := os.Open(dir)
	if err != nil {
		return nil, err
	}
	if err := syscall.Flock(int(handle.Fd()), syscall.LOCK_EX); err != nil {
		_ = handle.Close()
		return nil, err
	}
	return func() {
		_ = syscall.Flock(int(handle.Fd()), syscall.LOCK_UN)
		_ = handle.Close()
	}, nil
}
