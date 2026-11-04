package tix

import (
	"io"
	"time"

	"github.com/theroymind/go-tix/contracts/ticket"
	"github.com/theroymind/go-tix/contracts/timestamp"
	"github.com/theroymind/go-tix/internal/tix/backend"
	"github.com/theroymind/railyard-cli"
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
