package tix

import (
	"io"
	"time"

	"github.com/theroymind/go-tix/contracts/ticket"
	"github.com/theroymind/go-tix/contracts/timestamp"
	"github.com/theroymind/go-tix/internal/tix/backend"
	"github.com/theroymind/railyard-cli"
)

var statusSpecification = cli.Specification{AllowsPositionals: true}

func runStart(arguments []string, _ io.Reader, _ io.Writer) error {
	return runSetStatus(arguments, ticket.StatusInProgress)
}

func runReview(arguments []string, _ io.Reader, _ io.Writer) error {
	return runSetStatus(arguments, ticket.StatusInReview)
}

// runSetStatus moves every ticket the arguments name to status, after resolving them all, so a reference that
// matches no ticket changes none. A vault writes the note status the ticket's mapping gives it, and refuses a status
// the mapping doesn't name.
func runSetStatus(arguments []string, status ticket.Status) error {
	parsed, err := cli.Parse(arguments, statusSpecification)
	if err != nil {
		return err
	}
	if len(parsed.Positionals) == 0 {
		return cli.NewUsageError(ErrExpectedTicketReferences)
	}
	store, err := openBackend()
	if err != nil {
		return err
	}
	ids, err := resolveAll(store, parsed.Positionals)
	if err != nil {
		return err
	}
	return setStatuses(store, ids, status)
}

// setStatuses moves each ticket to status in turn, stopping at the first that fails.
func setStatuses(store backend.Backend, ids []string, status ticket.Status) error {
	for _, id := range ids {
		if err := setStatus(store, id, status); err != nil {
			return err
		}
	}
	return nil
}

func setStatus(store backend.Backend, id string, status ticket.Status) error {
	return store.Apply(id, []backend.Change{
		backend.SetField{Field: ticket.FieldStatus, Value: status},
		backend.SetField{Field: ticket.FieldUpdatedAt, Value: timestamp.Format(time.Now())},
	})
}
