package tix

import (
	"io"

	"github.com/theroymind/railyard/internal/cli"
	"github.com/theroymind/railyard/internal/contracts/ticket"
	"github.com/theroymind/railyard/internal/tix/graph"
)

var readySpecification = cli.Specification{
	Options: []cli.Option{
		{Name: optionType, Type: cli.OptionTypeString},
		{Name: optionJSON, Type: cli.OptionTypeBoolean},
	},
}

func runReady(arguments []string, _ io.Reader, stdout io.Writer) error {
	parsed, err := cli.Parse(arguments, readySpecification)
	if err != nil {
		return err
	}
	ticketType, err := parseOption(parsed, optionType, ticket.Types)
	if err != nil {
		return err
	}
	ticketStore, err := openStore()
	if err != nil {
		return err
	}
	tickets, err := ticketStore.ReadTickets()
	if err != nil {
		return err
	}
	ready := graph.Ready(tickets)
	if ticketType != "" {
		ready = filter{ticketType: ticketType}.apply(ready)
	}
	return writeTickets(stdout, ready, parsed.IsEnabled(optionJSON))
}
