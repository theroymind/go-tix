package tix

import (
	"errors"
	"fmt"
	"io"

	"github.com/theroymind/railyard/internal/cli"
	"github.com/theroymind/railyard/internal/tix/backend"
)

var backlinksSpecification = cli.Specification{
	Options:           []cli.Option{{Name: optionJSON, Type: cli.OptionTypeBoolean}},
	AllowsPositionals: true,
}

// runBacklinks lists the notes linking to the ticket or doc the reference names, one id per line or as JSON. A
// reference that names neither is taken as a slug as written, since notes link to unmapped notes too.
func runBacklinks(arguments []string, _ io.Reader, stdout io.Writer) error {
	parsed, err := cli.Parse(arguments, backlinksSpecification)
	if err != nil {
		return err
	}
	if len(parsed.Positionals) != 1 {
		return cli.NewUsageError(ErrExpectedOneReference)
	}
	store, err := openDocStore()
	if err != nil {
		return err
	}
	id, _, err := resolveEither(store.(backend.Backend), parsed.Positionals[0])
	if errors.Is(err, backend.ErrNoMatch) {
		id, err = parsed.Positionals[0], nil
	}
	if err != nil {
		return err
	}
	slugs, err := store.Backlinks(id)
	if err != nil {
		return err
	}
	if parsed.IsEnabled(optionJSON) {
		if slugs == nil {
			slugs = []string{}
		}
		return cli.WriteJSON(stdout, slugs)
	}
	for _, slug := range slugs {
		if _, err := fmt.Fprintln(stdout, slug); err != nil {
			return err
		}
	}
	return nil
}
