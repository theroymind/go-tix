package tix

import (
	"os"
	"path/filepath"

	"github.com/theroymind/go-tix/internal/tix/backend"
)

// workspace is an open backend with its config, its .tix directory and the directory holding it.
type workspace struct {
	store        backend.Backend
	config       backend.Config
	tixDirectory string
	root         string
}

// openWorkspace opens the backend the working directory is in, with its config.
func openWorkspace() (workspace, error) {
	workingDirectory, err := os.Getwd()
	if err != nil {
		return workspace{}, err
	}
	tixDirectory, err := backend.FindDirectory(workingDirectory)
	if err != nil {
		return workspace{}, err
	}
	config, err := backend.ReadConfig(tixDirectory)
	if err != nil {
		return workspace{}, err
	}
	store, err := backend.Open(workingDirectory)
	if err != nil {
		return workspace{}, err
	}
	return workspace{store: store, config: config, tixDirectory: tixDirectory, root: filepath.Dir(tixDirectory)}, nil
}
