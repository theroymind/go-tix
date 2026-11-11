package tix

import (
	"fmt"
	"io"

	"github.com/theroymind/go-tix/contracts/ticket"
	"github.com/theroymind/go-tix/contracts/validation"
	"github.com/theroymind/railyard-cli"
)

var lsSpecification = cli.Specification{
	Options: []cli.Option{
		{Name: optionStatus, Type: cli.OptionTypeString},
		{Name: optionType, Type: cli.OptionTypeString},
		{Name: optionTag, Type: cli.OptionTypeString},
		{Name: optionParent, Type: cli.OptionTypeString},
		{Name: optionAll, Type: cli.OptionTypeBoolean},
		{Name: optionJSON, Type: cli.OptionTypeBoolean},
	},
}

func runLs(arguments []string, _ io.Reader, stdout io.Writer) error {
	parsed, err := cli.Parse(arguments, lsSpecification)
	if err != nil {
		return err
	}
	status, err := parseOption(parsed, optionStatus, ticket.Statuses)
	if err != nil {
		return err
	}
	tag, _ := parsed.String(optionTag)
	store, err := openBackend()
	if err != nil {
		return err
	}
	ticketType, err := parseType(parsed, store)
	if err != nil {
		return err
	}
	var parent string
	if reference, exists := parsed.String(optionParent); exists {
		if parent, err = store.Resolve(reference); err != nil {
			return err
		}
	}
	tickets, err := store.List()
	if err != nil {
		return err
	}
	listed := filter{
		status:           status,
		ticketType:       ticketType,
		tag:              tag,
		parent:           parent,
		includesResolved: parsed.IsEnabled(optionAll),
	}.apply(tickets)
	properties, err := readPropertyColumns(store)
	if err != nil {
		return err
	}
	return writeTickets(stdout, listed, properties, parsed.IsEnabled(optionJSON))
}

// parseOption returns the option's value, or "" when it's absent, and rejects a value outside options as a usage error.
func parseOption[Option ~string](parsed cli.Arguments, name string, options []Option) (Option, error) {
	value, exists := parsed.String(name)
	if !exists {
		return "", nil
	}
	for _, option := range options {
		if Option(value) == option {
			return option, nil
		}
	}
	return "", cli.NewUsageError(fmt.Errorf("%s%s %q %w: %s", longOptionStart, name, value, ErrInvalidOption, validation.InvalidOption(options)))
}
