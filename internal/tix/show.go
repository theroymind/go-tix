package tix

import (
	"io"
	"os"

	"github.com/theroymind/railyard/internal/cli"
	"github.com/theroymind/railyard/internal/tix/backend"
)

var showSpecification = cli.Specification{
	Options:           []cli.Option{{Name: optionJSON, Type: cli.OptionTypeBoolean}},
	AllowsPositionals: true,
}

func runShow(arguments []string, _ io.Reader, stdout io.Writer) error {
	parsed, err := cli.Parse(arguments, showSpecification)
	if err != nil {
		return err
	}
	if len(parsed.Positionals) != 1 {
		return cli.NewUsageError(ErrExpectedOneReference)
	}
	store, err := openBackend()
	if err != nil {
		return err
	}
	id, err := store.Resolve(parsed.Positionals[0])
	if err != nil {
		return err
	}
	// A backend without source text shows the ticket as JSON.
	if textReader, hasText := store.(backend.TextReader); hasText && !parsed.IsEnabled(optionJSON) {
		text, err := textReader.ReadText(id)
		if err != nil {
			return err
		}
		_, err = stdout.Write(text)
		return err
	}
	shown, err := store.Read(id)
	if err != nil {
		return err
	}
	return cli.WriteJSON(stdout, shown)
}

func openBackend() (backend.Backend, error) {
	workingDirectory, err := os.Getwd()
	if err != nil {
		return nil, err
	}
	return backend.Open(workingDirectory)
}
