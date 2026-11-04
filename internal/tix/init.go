package tix

import (
	"errors"
	"io"
	"io/fs"
	"os"
	"path/filepath"

	"github.com/theroymind/go-tix/contracts/ticket"
	"github.com/theroymind/go-tix/internal/tix/backend"
	"github.com/theroymind/railyard-cli"
)

var initSpecification = cli.Specification{
	Options: []cli.Option{{Name: optionPrefix, Type: cli.OptionTypeString}},
}

func runInit(arguments []string, _ io.Reader, _ io.Writer) error {
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
	directory := filepath.Join(workingDirectory, ticket.DirectoryName)
	if _, err := os.Stat(directory); err == nil {
		return backend.AlreadyInitialized(directory)
	} else if !errors.Is(err, fs.ErrNotExist) {
		return err
	}
	config := backend.Config{Prefix: prefix}
	if err := backend.Validate(config, backend.ConfigPath(directory)); err != nil {
		return err
	}
	if err := os.Mkdir(directory, backend.DirectoryPermission); err != nil {
		return err
	}
	return backend.WriteConfig(directory, config)
}
