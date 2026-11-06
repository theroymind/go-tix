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
	"github.com/theroymind/go-tix/internal/tix/starter"
	"github.com/theroymind/railyard-cli"
)

var initSpecification = cli.Specification{
	Options: []cli.Option{
		{Name: optionPrefix, Type: cli.OptionTypeString},
		{Name: optionBackend, Type: cli.OptionTypeString},
	},
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
	kindAnswer, hasKind := parsed.String(optionBackend)
	if !hasKind {
		kindAnswer = string(backend.KindVault)
	}
	if err := checkKind(kindAnswer); err != nil {
		return err
	}
	config := configFor(prefix, backend.Kind(kindAnswer), backend.DefaultVaultRoot)
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
	if config.Vault != nil {
		if err := starter.WriteTemplates(filepath.Join(workingDirectory, config.Vault.Root)); err != nil {
			return err
		}
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
