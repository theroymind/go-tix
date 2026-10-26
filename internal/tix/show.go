package tix

import (
	"io"
	"os"

	"github.com/theroymind/railyard/internal/cli"
	"github.com/theroymind/railyard/internal/tix/store"
)

var showSpecification = cli.Specification{
	Options:           []cli.Option{{Name: optionJSON, Type: cli.OptionTypeBoolean}},
	AllowsPositionals: true,
}

func runShow(arguments []string, stdout io.Writer) error {
	parsed, err := cli.Parse(arguments, showSpecification)
	if err != nil {
		return err
	}
	if len(parsed.Positionals) != 1 {
		return cli.NewUsageError(ErrExpectedOneReference)
	}
	ticketStore, err := openStore()
	if err != nil {
		return err
	}
	id, err := ticketStore.ResolveTicketID(parsed.Positionals[0])
	if err != nil {
		return err
	}
	file, err := ticketStore.ReadTicketFile(id)
	if err != nil {
		return err
	}
	if parsed.IsEnabled(optionJSON) {
		return cli.WriteJSON(stdout, file.Ticket)
	}
	_, err = stdout.Write(file.Text)
	return err
}

func openStore() (store.Store, error) {
	workingDirectory, err := os.Getwd()
	if err != nil {
		return store.Store{}, err
	}
	return store.Open(workingDirectory)
}
