package tix

import (
	"bytes"
	"fmt"
	"io"
	"path/filepath"

	"github.com/theroymind/go-tix/internal/tix/backend"
	"github.com/theroymind/railyard-cli"
)

var gateSpecification = cli.Specification{
	Options:           []cli.Option{{Name: optionStaged, Type: cli.OptionTypeBoolean}},
	AllowsPositionals: true,
}

// runGate lints the ticket and doc notes git has staged, or the notes the references name. Nothing staged passes.
func runGate(arguments []string, _ io.Reader, stdout io.Writer) error {
	parsed, err := cli.Parse(arguments, gateSpecification)
	if err != nil {
		return err
	}
	isStaged := parsed.IsEnabled(optionStaged)
	if isStaged == (len(parsed.Positionals) > 0) {
		return cli.NewUsageError(ErrExpectedReferencesOrStaged)
	}
	opened, err := openWorkspace()
	if err != nil {
		return err
	}
	ids, err := gateNotes(opened, parsed.Positionals, isStaged)
	if err != nil {
		return err
	}
	if len(ids) == 0 {
		return nil
	}
	return lintNotes(opened.store, ids, false, stdout)
}

// gateNotes returns the ids of the notes to gate: the ticket and doc notes git has staged, added or modified, or
// else the notes the references name.
func gateNotes(opened workspace, references []string, isStaged bool) ([]string, error) {
	if !isStaged {
		var ids []string
		for _, reference := range references {
			id, _, err := resolveEither(opened.store, reference)
			if err != nil {
				return nil, err
			}
			ids = append(ids, id)
		}
		return ids, nil
	}
	reader, isLintReader := opened.store.(backend.LintReader)
	if !isLintReader {
		return nil, ErrNoLint
	}
	output, err := cli.RunChildIn(opened.root, gitExecutable, stagedArguments, nil)
	if err != nil {
		return nil, err
	}
	staged, err := parseStaged(output)
	if err != nil {
		return nil, err
	}
	lintable, err := reader.LintNotes()
	if err != nil {
		return nil, err
	}
	return stagedNotes(staged, lintable), nil
}

// stagedNotes keeps the staged paths that are ticket or doc notes, as ids, in the order git listed them.
func stagedNotes(staged []string, lintable []backend.LintNote) []string {
	byPath := make(map[string]string, len(lintable))
	for _, note := range lintable {
		byPath[filepath.ToSlash(note.Path)] = note.ID
	}
	var ids []string
	for _, path := range staged {
		if id, isNote := byPath[path]; isNote {
			ids = append(ids, id)
		}
	}
	return ids
}

// parseStaged reads git diff --name-status -z: a status, then a path, each ended by a NUL, and returns the paths.
func parseStaged(output []byte) ([]string, error) {
	if len(output) == 0 {
		return nil, nil
	}
	fields := bytes.Split(bytes.TrimSuffix(output, []byte{nulSeparator}), []byte{nulSeparator})
	if len(fields)%2 != 0 {
		return nil, fmt.Errorf("%w: %q", ErrInvalidStaged, output)
	}
	var paths []string
	for position := 0; position < len(fields); position += 2 {
		status := string(fields[position])
		if status != statusAdded && status != statusModified {
			return nil, fmt.Errorf("%w: status %q", ErrInvalidStaged, status)
		}
		paths = append(paths, string(fields[position+1]))
	}
	return paths, nil
}
