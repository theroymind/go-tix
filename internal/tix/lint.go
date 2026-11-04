package tix

import (
	"fmt"
	"io"
	"slices"

	"github.com/theroymind/railyard/internal/cli"
	"github.com/theroymind/railyard/internal/tix/backend"
	"github.com/theroymind/railyard/internal/tix/lint"
)

var lintSpecification = cli.Specification{
	Options:           []cli.Option{{Name: optionAll, Type: cli.OptionTypeBoolean}},
	AllowsPositionals: true,
}

// runLint checks the notes the references name, or every note with --all, and prints a line for each rule a note
// breaks. It fails when any note breaks one.
func runLint(arguments []string, _ io.Reader, stdout io.Writer) error {
	parsed, err := cli.Parse(arguments, lintSpecification)
	if err != nil {
		return err
	}
	isAll := parsed.IsEnabled(optionAll)
	if isAll == (len(parsed.Positionals) > 0) {
		return cli.NewUsageError(ErrExpectedReferencesOrAll)
	}
	store, err := openBackend()
	if err != nil {
		return err
	}
	var ids []string
	for _, reference := range parsed.Positionals {
		id, _, err := resolveEither(store, reference)
		if err != nil {
			return err
		}
		ids = append(ids, id)
	}
	return lintNotes(store, ids, isAll, stdout)
}

// lintNotes checks the notes ids name, or every note when isAll, prints each finding as PATH: RULE: MESSAGE, and
// fails with ErrLintFailed when there is any.
func lintNotes(store backend.Backend, ids []string, isAll bool, stdout io.Writer) error {
	reader, isLintReader := store.(backend.LintReader)
	if !isLintReader {
		return ErrNoLint
	}
	notes, err := reader.LintNotes()
	if err != nil {
		return err
	}
	targetList, err := reader.LinkTargets()
	if err != nil {
		return err
	}
	targets := make(map[string]bool, len(targetList))
	for _, target := range targetList {
		targets[target] = true
	}
	var findings []lint.Finding
	for _, note := range notes {
		if isAll || slices.Contains(ids, note.ID) {
			findings = append(findings, lint.Check(note, targets)...)
		}
	}
	for _, found := range findings {
		if _, err := fmt.Fprintf(stdout, findingLine, found.Path, found.Rule, found.Message); err != nil {
			return err
		}
	}
	if len(findings) == 0 {
		return nil
	}
	return lintFailed(findings)
}

func lintFailed(findings []lint.Finding) error {
	var failing []string
	for _, found := range findings {
		if !slices.Contains(failing, found.ID) {
			failing = append(failing, found.ID)
		}
	}
	return fmt.Errorf("%w: %d problems in %d notes", ErrLintFailed, len(findings), len(failing))
}
