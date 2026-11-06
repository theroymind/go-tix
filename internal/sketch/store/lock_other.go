//go:build !unix

package store

// lock refuses on a platform without flock, so Update never edits unlocked.
func lock(string) (func(), error) {
	return nil, ErrLockUnsupported
}
