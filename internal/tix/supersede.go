package tix

import (
	"fmt"
	"io"
	"strconv"

	"github.com/theroymind/railyard/internal/cli"
	"github.com/theroymind/railyard/internal/tix/backend"
)

var supersedeSpecification = cli.Specification{AllowsPositionals: true}

// runSupersede marks the old doc superseded by the new one, touching only the old doc's frontmatter. Both must be
// docs of one kind, the new doc must already exist, and the kind's statuses must include superseded, which
// ApplyDoc checks before it writes.
func runSupersede(arguments []string, _ io.Reader, _ io.Writer) error {
	parsed, err := cli.Parse(arguments, supersedeSpecification)
	if err != nil {
		return err
	}
	if len(parsed.Positionals) != 2 {
		return cli.NewUsageError(ErrExpectedOldAndNew)
	}
	store, err := openDocStore()
	if err != nil {
		return err
	}
	old, err := resolveDoc(store, parsed.Positionals[0])
	if err != nil {
		return err
	}
	replacement, err := resolveDoc(store, parsed.Positionals[1])
	if err != nil {
		return err
	}
	if old.ID == replacement.ID {
		return fmt.Errorf("%s %w", strconv.Quote(old.ID), ErrSupersedesItself)
	}
	if old.Kind != replacement.Kind {
		return fmt.Errorf("%w: %s is of kind %s, %s of kind %s", ErrKindsDiffer, old.ID, old.Kind, replacement.ID, replacement.Kind)
	}
	return store.ApplyDoc(old.ID, []backend.Change{
		backend.SetField{Field: optionStatus, Value: backend.StatusSuperseded},
		backend.SetField{Field: backend.SupersededByProperty, Value: replacement.ID},
	})
}

// resolveDoc reads the doc the reference names.
func resolveDoc(store backend.DocStore, reference string) (backend.Doc, error) {
	id, err := store.ResolveDoc(reference)
	if err != nil {
		return backend.Doc{}, err
	}
	doc, err := store.ReadDoc(id)
	if err != nil {
		return backend.Doc{}, err
	}
	return doc, nil
}
