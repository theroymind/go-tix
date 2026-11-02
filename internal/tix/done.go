package tix

import (
	"io"
	"time"

	"github.com/theroymind/railyard/internal/cli"
	"github.com/theroymind/railyard/internal/contracts/ticket"
	"github.com/theroymind/railyard/internal/contracts/timestamp"
	"github.com/theroymind/railyard/internal/tix/backend"
)

var doneSpecification = cli.Specification{AllowsPositionals: true}

func runDone(arguments []string, _ io.Reader, _ io.Writer) error {
	parsed, err := cli.Parse(arguments, doneSpecification)
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
	return store.Apply(id, []backend.Change{
		backend.SetField{Field: ticket.FieldStatus, Value: ticket.StatusCompleted},
		backend.SetField{Field: ticket.FieldUpdatedAt, Value: timestamp.Format(time.Now())},
	})
}
