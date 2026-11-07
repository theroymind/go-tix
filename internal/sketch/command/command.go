// Package command provides the sketch commands: ls, which lists the repo's .sketch.md notes as a table or JSON,
// questions, which lists their open questions, and help, which documents the note format.
package command

import (
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"text/tabwriter"

	cli "github.com/theroymind/railyard-cli"

	"github.com/theroymind/go-tix/contracts/sketch"
	"github.com/theroymind/go-tix/internal/sketch/marks"
	"github.com/theroymind/go-tix/internal/sketch/note"
	"github.com/theroymind/go-tix/internal/sketch/questions"
	"github.com/theroymind/go-tix/internal/sketch/refs"
)

const usage = `usage: sketch <command> [arguments]

commands:
  ls [--json]          lists every note under the repo root with the file it plans, + when that file is new
                       or ~ when it exists, its feature and task, its items, each + when the file doesn't
                       hold its name yet or ~ when it does, and how many questions it asks; in a tix repo, a
                       feature tix has no ticket for is flagged (not in tix); --json prints an array of notes
                       with mark new or changed, and unknown_feature: true on a flagged one
  questions [--json]   lists each open question as NOTE:LINE: QUESTION, by note and then line; --json prints
                       an array of {path, line, text}
  help                 prints this

A note is a plain markdown file beside the file it plans, named after it with .sketch.md added:
src/useOrbit.ts.sketch.md plans src/useOrbit.ts, whether or not that file exists yet. Write whatever you like in
it; these lines give it structure, and every other line is free text, never an error:

  ---                         optional front matter, only at the very top, between two --- lines:
  feature: FEATURE            the tix feature the note belongs to
  task: TASK                  and the tix task
  ---
  #TYPE NAME [TEXT]           an item the file will hold, where TYPE is one of
                              fn, type, component, hook, flow, tune, rule or test, and TEXT is the rest of the
                              line, such as #fn useOrbit target Ref<Vec3>
  ? QUESTION                  an open question
  > ANSWER                    on the line right under a question, its answer

Markers start the line. A #word that isn't an item type, such as #todo, and a # heading are free text, and so is
a > line that isn't right under a question. The repo root is the nearest directory above holding .tix or .git.
ls and questions skip hidden folders and node_modules, list every note they can read, and then fail naming
any they couldn't.
`

const (
	optionJSON  = "json"
	tableHeader = "NOTE\tTARGET\tFEATURE\tTASK\tITEMS\tQUESTIONS"
	tablePad    = 2
	noValue     = "-"

	unknownFeatureMarker = " (not in tix)"
	itemSeparator        = ", "
)

var rootMarkers = []string{".tix", ".git"}

var ErrUnreadableNotes = errors.New("some notes couldn't be read")

var Program = cli.Program{
	Name:  "sketch",
	Usage: usage,
	Commands: map[string]cli.Command{
		"ls":        runList,
		"questions": runQuestions,
	},
}

// runList prints every note under the repo root that reads and validates, marked against its target and, in a
// tix repo, flagged when tix has no ticket for its feature, as a table or a JSON array. It then fails naming each
// note that didn't read, so one bad note never hides the rest.
func runList(arguments []string, _ io.Reader, stdout io.Writer) error {
	asJSON, err := parseJSONOption(arguments)
	if err != nil {
		return err
	}
	root, err := repoRootFromWorkingDir()
	if err != nil {
		return err
	}
	loaded, problems, err := loadNotes(root)
	if err != nil {
		return err
	}
	reader := marks.NewReader(root)
	notes := []sketch.Note{}
	for _, value := range loaded {
		if err := reader.MarkNote(&value); err != nil {
			problems = append(problems, fmt.Errorf("%s: %w", value.Path, err))
			continue
		}
		notes = append(notes, value)
	}
	if err := flagUnknownFeatures(refs.New(root), notes); err != nil {
		return err
	}
	if asJSON {
		err = cli.WriteJSON(stdout, notes)
	} else {
		err = writeTable(stdout, notes)
	}
	if err != nil {
		return err
	}
	return unreadable(problems)
}

// runQuestions prints each open question under the repo root with the note and line that asks it, as
// path:line: text lines or a JSON array, and then fails naming each note that didn't read.
func runQuestions(arguments []string, _ io.Reader, stdout io.Writer) error {
	asJSON, err := parseJSONOption(arguments)
	if err != nil {
		return err
	}
	root, err := repoRootFromWorkingDir()
	if err != nil {
		return err
	}
	notes, problems, err := loadNotes(root)
	if err != nil {
		return err
	}
	open := questions.Open(notes)
	if asJSON {
		err = cli.WriteJSON(stdout, open)
	} else {
		err = writeQuestions(stdout, open)
	}
	if err != nil {
		return err
	}
	return unreadable(problems)
}

// parseJSONOption parses a command's arguments, which take only --json, and reports whether it was given.
func parseJSONOption(arguments []string) (bool, error) {
	parsed, err := cli.Parse(arguments, cli.Specification{
		Options: []cli.Option{{Name: optionJSON, Type: cli.OptionTypeBoolean}},
	})
	if err != nil {
		return false, err
	}
	return parsed.IsEnabled(optionJSON), nil
}

// loadNotes returns every note under root that reads and validates, in path order, and an error for each one
// that doesn't. Only failing to walk root fails it.
func loadNotes(root string) ([]sketch.Note, []error, error) {
	paths, err := note.Find(root)
	if err != nil {
		return nil, nil, err
	}
	var notes []sketch.Note
	var problems []error
	for _, notePath := range paths {
		value, err := loadValid(root, notePath)
		if err != nil {
			problems = append(problems, err)
			continue
		}
		notes = append(notes, value)
	}
	return notes, problems, nil
}

// loadValid loads the note at notePath and checks it against the contract.
func loadValid(root, notePath string) (sketch.Note, error) {
	value, err := note.Load(root, notePath)
	if err != nil {
		return sketch.Note{}, err
	}
	if err := value.Validate(); err != nil {
		return sketch.Note{}, fmt.Errorf("%s: %w", notePath, err)
	}
	return value, nil
}

// flagUnknownFeatures sets UnknownFeature on each note whose feature tix has no ticket for. It asks tix once, and
// only in a tix repo where some note names a feature; outside one, any feature is accepted.
func flagUnknownFeatures(resolver refs.Resolver, notes []sketch.Note) error {
	namesFeature := func(value sketch.Note) bool { return value.Feature != "" }
	if !resolver.IsTixRepo() || !slices.ContainsFunc(notes, namesFeature) {
		return nil
	}
	tickets, err := resolver.Tickets()
	if err != nil {
		return err
	}
	for index := range notes {
		feature := notes[index].Feature
		notes[index].UnknownFeature = feature != "" && !tickets[feature]
	}
	return nil
}

// unreadable joins the errors of notes that didn't read under ErrUnreadableNotes, or returns nil when there are
// none.
func unreadable(problems []error) error {
	if len(problems) == 0 {
		return nil
	}
	return fmt.Errorf("%w:\n%w", ErrUnreadableNotes, errors.Join(problems...))
}

// writeTable writes one row per note: its target with + when the file is new or ~ when it exists, its feature
// marked when tix doesn't know it, and each item's name with its own + or ~. A feature, task or items the note
// doesn't have show as -.
func writeTable(writer io.Writer, notes []sketch.Note) error {
	table := tabwriter.NewWriter(writer, 0, 0, tablePad, ' ', 0)
	if _, err := fmt.Fprintln(table, tableHeader); err != nil {
		return err
	}
	for _, value := range notes {
		feature := orNoValue(value.Feature)
		if value.UnknownFeature {
			feature += unknownFeatureMarker
		}
		var items []string
		for _, item := range value.Items {
			items = append(items, item.Mark.Symbol()+item.Name)
		}
		_, err := fmt.Fprintf(table, "%s\t%s\t%s\t%s\t%s\t%d\n", value.Path, value.Mark.Symbol()+value.Target, feature,
			orNoValue(value.Task), orNoValue(strings.Join(items, itemSeparator)), len(value.Questions))
		if err != nil {
			return err
		}
	}
	return table.Flush()
}

// writeQuestions writes one path:line: text line per question, the form editors and grep use.
func writeQuestions(writer io.Writer, open []questions.Question) error {
	for _, question := range open {
		if _, err := fmt.Fprintf(writer, "%s:%d: %s\n", question.Path, question.Line, question.Text); err != nil {
			return err
		}
	}
	return nil
}

func orNoValue(text string) string {
	if text == "" {
		return noValue
	}
	return text
}

func repoRootFromWorkingDir() (string, error) {
	workingDir, err := os.Getwd()
	if err != nil {
		return "", err
	}
	return repoRoot(workingDir), nil
}

// repoRoot returns the nearest directory at or above dir that holds .tix or .git, or dir when none does.
func repoRoot(dir string) string {
	for candidate := dir; ; {
		for _, marker := range rootMarkers {
			if _, err := os.Stat(filepath.Join(candidate, marker)); err == nil {
				return candidate
			}
		}
		parent := filepath.Dir(candidate)
		if parent == candidate {
			return dir
		}
		candidate = parent
	}
}
