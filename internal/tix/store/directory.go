package store

import (
	"os"
	"path/filepath"
)

func FindDirectory(startDirectory string) (string, error) {
	directory, err := filepath.Abs(startDirectory)
	if err != nil {
		return "", err
	}
	for {
		candidate := filepath.Join(directory, DirectoryName)
		if info, err := os.Stat(candidate); err == nil && info.IsDir() {
			return candidate, nil
		}
		parent := filepath.Dir(directory)
		if parent == directory {
			return "", ErrNotInRepository
		}
		directory = parent
	}
}
