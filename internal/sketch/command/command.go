// Package command provides the sketch commands: ls, which lists the repo's .sketch.md notes as a table or JSON,
// and help, which documents the note format.
package command

import (
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"text/tabwriter"

	cli "github.com/theroymind/railyard-cli"

	"github.com/theroymind/go-tix/contracts/sketch"
	"github.com/theroymind/go-tix/internal/sketch/note"
)

const usage = `usage: sketch <command> [arguments]

commands:
  ls [--json]   lists every note under the repo root with the file it plans, its feature and task, and how many
                items and questions it holds; --json prints them as an array of notes
  help          prints this

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
ls skips hidden folders and node_modules, lists every note it can read, and then fails naming any it couldn't.
`

const (
	optionJSON  = "json"
	tableHeader = "NOTE\tTARGET\tFEATURE\tTASK\tITEMS\tQUESTIONS"
	tablePad    = 2
	noValue     = "-"
)

var rootMarkers = []string{".tix", ".git"}

var ErrUnreadableNotes = errors.New("some notes couldn't be read")

var Program = cli.Program{
	Name:  "sketch",
	Usage: usage,
	Commands: map[string]cli.Command{
		"ls": runList,
	},
}

// runList prints every note under the repo root that reads and validates, as a table or a JSON array, and then
// fails naming each note that didn't, so one bad note never hides the rest.
func runList(arguments []string, _ io.Reader, stdout io.Writer) error {
	parsed, err := cli.Parse(arguments, cli.Specification{
		Options: []cli.Option{{Name: optionJSON, Type: cli.OptionTypeBoolean}},
	})
	if err != nil {
		return err
	}
	root, err := repoRootFromWorkingDir()
	if err != nil {
		return err
	}
	paths, err := note.Find(root)
	if err != nil {
		return err
	}
	notes := []sketch.Note{}
	var problems []error
	for _, notePath := range paths {
		value, err := loadValid(root, notePath)
		if err != nil {
			problems = append(problems, err)
			continue
		}
		notes = append(notes, value)
	}
	if parsed.IsEnabled(optionJSON) {
		err = cli.WriteJSON(stdout, notes)
	} else {
		err = writeTable(stdout, notes)
	}
	if err != nil {
		return err
	}
	if len(problems) > 0 {
		return fmt.Errorf("%w:\n%w", ErrUnreadableNotes, errors.Join(problems...))
	}
	return nil
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

// writeTable writes one row per note, with - for a feature or task the note doesn't name.
func writeTable(writer io.Writer, notes []sketch.Note) error {
	table := tabwriter.NewWriter(writer, 0, 0, tablePad, ' ', 0)
	if _, err := fmt.Fprintln(table, tableHeader); err != nil {
		return err
	}
	for _, value := range notes {
		_, err := fmt.Fprintf(table, "%s\t%s\t%s\t%s\t%d\t%d\n", value.Path, value.Target, orNoValue(value.Feature),
			orNoValue(value.Task), len(value.Items), len(value.Questions))
		if err != nil {
			return err
		}
	}
	return table.Flush()
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
