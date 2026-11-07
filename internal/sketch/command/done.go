package command

import (
	"fmt"
	"io"
	"os"
	"path/filepath"

	cli "github.com/theroymind/railyard-cli"
)

// runDone deletes the notes that belong to the feature its one argument names, printing each path as it goes,
// and fails with ErrNoNotes when there are none. It then fails naming each note that didn't read, since one of
// them may belong to the feature too. Committing the deletions is left to the caller.
func runDone(arguments []string, _ io.Reader, stdout io.Writer) error {
	parsed, err := cli.Parse(arguments, cli.Specification{AllowsPositionals: true})
	if err != nil {
		return err
	}
	if len(parsed.Positionals) != 1 {
		return cli.NewUsageError(ErrExpectedOneFeature)
	}
	feature := parsed.Positionals[0]
	root, loaded, err := loadNotes(feature)
	if err != nil {
		return err
	}
	if len(loaded.Notes) == 0 {
		return fmt.Errorf("%w: %s", ErrNoNotes, feature)
	}
	for _, value := range loaded.Notes {
		if err := os.Remove(filepath.Join(root, filepath.FromSlash(value.Path))); err != nil {
			return err
		}
		if _, err := fmt.Fprintln(stdout, value.Path); err != nil {
			return err
		}
	}
	return unreadable(loaded.Problems)
}
