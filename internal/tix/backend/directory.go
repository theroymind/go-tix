package backend

import (
	"os"
	"path/filepath"

	"github.com/theroymind/railyard/internal/contracts/ticket"
)

func FindDirectory(startDirectory string) (string, error) {
	directory, err := filepath.Abs(startDirectory)
	if err != nil {
		return "", err
	}
	for {
		candidate := filepath.Join(directory, ticket.DirectoryName)
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
