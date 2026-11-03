package tix

import (
	"io"
	"time"

	"github.com/theroymind/railyard/internal/cli"
	"github.com/theroymind/railyard/internal/contracts/ticket"
	"github.com/theroymind/railyard/internal/contracts/timestamp"
	"github.com/theroymind/railyard/internal/tix/backend"
)

var unsetSpecification = cli.Specification{AllowsPositionals: true}

// runUnset removes each key from the ticket or doc REF names, all of them or none, and bumps updated_at where the
// file keeps it.
func runUnset(arguments []string, _ io.Reader, _ io.Writer) error {
	parsed, err := cli.Parse(arguments, unsetSpecification)
	if err != nil {
		return err
	}
	if len(parsed.Positionals) < 2 {
		return cli.NewUsageError(ErrExpectedKeys)
	}
	store, err := openBackend()
	if err != nil {
		return err
	}
	id, apply, err := resolveAny(store, parsed.Positionals[0])
	if err != nil {
		return err
	}
	keys := parsed.Positionals[1:]
	changes := make([]backend.Change, 0, len(keys)+1)
	for _, key := range keys {
		changes = append(changes, backend.RemoveField{Field: key})
	}
	changes = append(changes, backend.SetField{Field: ticket.FieldUpdatedAt, Value: timestamp.Format(time.Now())})
	return apply(id, changes)
}
