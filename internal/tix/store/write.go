package store

import "os"

func writeAtomically(path string, contents []byte) error {
	temporaryPath := path + temporaryFileSuffix
	if err := os.WriteFile(temporaryPath, contents, filePermission); err != nil {
		return err
	}
	return os.Rename(temporaryPath, path)
}
