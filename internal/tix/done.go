package tix

import (
	"io"
	"time"

	"github.com/theroymind/railyard/internal/cli"
	"github.com/theroymind/railyard/internal/contracts/ticket"
	"github.com/theroymind/railyard/internal/contracts/timestamp"
	"github.com/theroymind/railyard/internal/tix/store"
)

var doneSpecification = cli.Specification{AllowsPositionals: true}

func runDone(arguments []string, _ io.Writer) error {
	parsed, err := cli.Parse(arguments, doneSpecification)
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
	return ticketStore.UpdateTicket(id, []store.Change{
		{Field: ticket.FieldStatus, Value: ticket.StatusCompleted},
		{Field: ticket.FieldUpdatedAt, Value: timestamp.Format(time.Now())},
	})
}
