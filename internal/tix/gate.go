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

// gateNote is a note a gate checks, and whether it is new, which only new notes face the judge for.
type gateNote struct {
	ID    string
	IsNew bool
}

// stagedFile is one path git has staged, and whether it is added rather than modified.
type stagedFile struct {
	Path    string
	IsAdded bool
}

// gates are the checks a note passes before it lands, in order: lint on every note, then the judge on the new ones.
type gates struct {
	lint  func(ids []string, stdout io.Writer) error
	judge func(ids []string, stdout io.Writer) error
}

// runGate lints the notes git has staged, or the notes the references name, and, when every one passes, asks the
// judge about the new ones. A reference counts as new. It stops at the first gate that fails.
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
	notes, err := gateNotes(opened, parsed.Positionals, isStaged)
	if err != nil {
		return err
	}
	checks := gates{
		lint: func(ids []string, stdout io.Writer) error {
			return lintNotes(opened.store, ids, false, stdout)
		},
		judge: func(ids []string, stdout io.Writer) error {
			session, err := openJudging(opened, stdout)
			if err != nil {
				return err
			}
			return session.judgeNotes(ids, stdout)
		},
	}
	return checks.run(notes, stdout)
}

// run passes every note through lint, then every new note through the judge, and returns the first gate's failure
// without running the gates after it. With no new notes the judge isn't asked.
func (checks gates) run(notes []gateNote, stdout io.Writer) error {
	if len(notes) == 0 {
		return nil
	}
	var all, added []string
	for _, note := range notes {
		all = append(all, note.ID)
		if note.IsNew {
			added = append(added, note.ID)
		}
	}
	if err := checks.lint(all, stdout); err != nil {
		return err
	}
	if len(added) == 0 {
		return nil
	}
	return checks.judge(added, stdout)
}

// gateNotes returns the notes to gate: the ticket and doc notes git has staged, added or modified, or else the notes
// the references name, each counted as new.
func gateNotes(opened workspace, references []string, isStaged bool) ([]gateNote, error) {
	if !isStaged {
		var notes []gateNote
		for _, reference := range references {
			id, _, err := resolveEither(opened.store, reference)
			if err != nil {
				return nil, err
			}
			notes = append(notes, gateNote{ID: id, IsNew: true})
		}
		return notes, nil
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

// stagedNotes keeps the staged files that are ticket or doc notes, as notes, in the order git listed them.
func stagedNotes(staged []stagedFile, lintable []backend.LintNote) []gateNote {
	byPath := make(map[string]string, len(lintable))
	for _, note := range lintable {
		byPath[filepath.ToSlash(note.Path)] = note.ID
	}
	var notes []gateNote
	for _, file := range staged {
		if id, isNote := byPath[file.Path]; isNote {
			notes = append(notes, gateNote{ID: id, IsNew: file.IsAdded})
		}
	}
	return notes
}

// parseStaged reads git diff --name-status -z: a status, then a path, each ended by a NUL.
func parseStaged(output []byte) ([]stagedFile, error) {
	if len(output) == 0 {
		return nil, nil
	}
	fields := bytes.Split(bytes.TrimSuffix(output, []byte{nulSeparator}), []byte{nulSeparator})
	if len(fields)%2 != 0 {
		return nil, fmt.Errorf("%w: %q", ErrInvalidStaged, output)
	}
	var files []stagedFile
	for position := 0; position < len(fields); position += 2 {
		status := string(fields[position])
		if status != statusAdded && status != statusModified {
			return nil, fmt.Errorf("%w: status %q", ErrInvalidStaged, status)
		}
		files = append(files, stagedFile{Path: string(fields[position+1]), IsAdded: status == statusAdded})
	}
	return files, nil
}
