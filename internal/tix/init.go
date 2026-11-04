package tix

import (
	"errors"
	"fmt"
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

func runInit(arguments []string, stdin io.Reader, stdout io.Writer) error {
	parsed, err := cli.Parse(arguments, initSpecification)
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
	prefix, hasPrefix := parsed.String(optionPrefix)
	config := backend.Config{Prefix: prefix}
	if !hasPrefix {
		setup := newPrompter(stdin, stdout)
		config, err = askConfig(setup)
		if err != nil {
			return err
		}
	}
	configPath := backend.ConfigPath(directory)
	if err := backend.Validate(config, configPath); err != nil {
		return err
	}
	if err := os.Mkdir(directory, backend.DirectoryPermission); err != nil {
		return err
	}
	if err := backend.WriteConfig(directory, config); err != nil {
		return err
	}
	if hasPrefix {
		return nil
	}
	_, err = fmt.Fprintf(stdout, initializedMessage, configPath)
	return err
}
