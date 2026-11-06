package tix

import (
	"fmt"
	"io"

	"github.com/theroymind/go-tix/internal/tix/backend"
	"github.com/theroymind/railyard-cli"
)

var pathSpecification = cli.Specification{AllowsPositionals: true}

// runPath prints the absolute path of the file holding the ticket REF names, for another app, such as sketch, to
// find what sits beside it.
func runPath(arguments []string, _ io.Reader, stdout io.Writer) error {
	parsed, err := cli.Parse(arguments, pathSpecification)
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
	pather, hasPaths := store.(backend.NotePather)
	if !hasPaths {
		return fmt.Errorf("path %w", backend.ErrUnsupported)
	}
	id, err := store.Resolve(parsed.Positionals[0])
	if err != nil {
		return err
	}
	path, err := pather.NotePath(id)
	if err != nil {
		return err
	}
	_, err = fmt.Fprintln(stdout, path)
	return err
}
