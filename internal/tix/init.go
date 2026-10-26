package tix

import (
	"errors"
	"io"
	"io/fs"
	"os"
	"path/filepath"

	"github.com/theroymind/railyard/internal/cli"
	"github.com/theroymind/railyard/internal/tix/store"
)

var initSpecification = cli.Specification{
	Options: []cli.Option{{Name: optionPrefix, Type: cli.OptionTypeString}},
}

func runInit(arguments []string, _ io.Writer) error {
	parsed, err := cli.Parse(arguments, initSpecification)
	if err != nil {
		return err
	}
	prefix, err := parsed.RequiredString(optionPrefix)
	if err != nil {
		return err
	}
	workingDirectory, err := os.Getwd()
	if err != nil {
		return err
	}
	directory := filepath.Join(workingDirectory, store.DirectoryName)
	if _, err := os.Stat(directory); err == nil {
		return store.AlreadyInitialized(directory)
	} else if !errors.Is(err, fs.ErrNotExist) {
		return err
	}
	config := store.Config{Prefix: prefix}
	if err := store.Validate(config, store.ConfigPath(directory)); err != nil {
		return err
	}
	if err := os.Mkdir(directory, store.DirectoryPermission); err != nil {
		return err
	}
	return store.WriteConfig(directory, config)
}
