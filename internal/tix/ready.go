package tix

import (
	"io"

	"github.com/theroymind/go-tix/internal/tix/graph"
	"github.com/theroymind/railyard-cli"
)

var readySpecification = cli.Specification{
	Options: []cli.Option{
		{Name: optionType, Type: cli.OptionTypeString},
		{Name: optionParent, Type: cli.OptionTypeString},
		{Name: optionJSON, Type: cli.OptionTypeBoolean},
	},
}

func runReady(arguments []string, _ io.Reader, stdout io.Writer) error {
	parsed, err := cli.Parse(arguments, readySpecification)
	if err != nil {
		return err
	}
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
	ready := graph.Ready(tickets)
	if ticketType != "" || parent != "" {
		ready = filter{ticketType: ticketType, parent: parent}.apply(ready)
	}
	properties, err := readPropertyColumns(store)
	if err != nil {
		return err
	}
	return writeTickets(stdout, ready, properties, parsed.IsEnabled(optionJSON))
}
