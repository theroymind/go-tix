package backend

import "os"

// WriteAtomically replaces path with contents, writing a temporary file and renaming it so a reader never sees a partial file.
func WriteAtomically(path string, contents []byte) error {
	temporaryPath := path + temporaryFileSuffix
	if err := os.WriteFile(temporaryPath, contents, FilePermission); err != nil {
		return err
	}
	return os.Rename(temporaryPath, path)
}
