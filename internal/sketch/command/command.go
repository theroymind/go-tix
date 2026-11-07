// Package command provides the sketch commands: the explorer, which bare sketch opens on the repo tree; ls, which
// lists the repo's .sketch.md notes as a table or JSON; questions, which lists their open questions; and help, which
// documents the note format.
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

	tea "charm.land/bubbletea/v2"
	"github.com/charmbracelet/x/term"
	cli "github.com/theroymind/railyard-cli"

	"github.com/theroymind/go-tix/contracts/sketch"
	"github.com/theroymind/go-tix/internal/sketch/explorer"
	"github.com/theroymind/go-tix/internal/sketch/questions"
	"github.com/theroymind/go-tix/internal/sketch/refs"
	"github.com/theroymind/go-tix/internal/sketch/repo"
)

const usage = `usage: sketch [<command> [arguments]]

With no command, in a terminal, sketch opens the explorer: the repo tree with each note shown as a + ghost node
when the file it plans doesn't exist yet or as ~ on that file when it does, and the selected node's note or first
lines beside it. j/k or the arrows move, l/enter opens a folder, h closes one, n shows only notes and the folders
above them, R reloads the tree, which also happens when the terminal regains focus, and q quits. a asks for a
file name and adds a note planning that file in the selected folder, or beside the selected file, A attaches a
note to the selected file, and on a file with a note either opens it; e opens the note in $EDITOR, or
in the parent nvim when $NVIM is set, and d deletes it after asking. r on a note attached to a file lists the
names in that file and adds the one you pick to the note as a "- uses" line; f filters the tree to one feature's
notes, picked from tix's features and the ones notes name, and links new notes to it; and ? jumps to the next note
with an open question. A note that couldn't be read shows as ! with the reason in the status line and preview,
so e can open it to fix it. In a list, type to filter, the arrows or ctrl+n/ctrl+p move, enter picks and esc
cancels.

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
In a git repo, sketch sees the files git does, tracked or not but never ignored; outside one, it skips hidden
folders and node_modules. ls and questions list every note they can read, and then fail naming any they
couldn't. To quiet markdownlint's MD018 and MD041 on notes, see the README's .markdownlint-cli2.yaml override.
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

// Run runs sketch with arguments: the explorer when there are none and stdin and stdout are a terminal, otherwise
// the command they name, with no arguments printing the usage.
func Run(arguments []string, stdin io.Reader, stdout, stderr io.Writer) cli.ExitCode {
	if len(arguments) > 0 || !isTerminal(stdin) || !isTerminal(stdout) {
		return Program.Run(arguments, stdin, stdout, stderr)
	}
	if err := runExplorer(stdin, stdout); err != nil {
		_, _ = fmt.Fprintf(stderr, "%s: %v\n", Program.Name, err)
		return cli.ExitCodeFailure
	}
	return cli.ExitCodeSuccess
}

func isTerminal(stream any) bool {
	file, isFile := stream.(*os.File)
	return isFile && term.IsTerminal(file.Fd())
}

// runExplorer opens the explorer on the repo tree until the user quits, and then fails naming each note that
// didn't read when the tree was last loaded.
func runExplorer(stdin io.Reader, stdout io.Writer) error {
	root, err := repoRootFromWorkingDir()
	if err != nil {
		return err
	}
	model, err := explorer.New(root)
	if err != nil {
		return err
	}
	final, err := tea.NewProgram(model, tea.WithInput(stdin), tea.WithOutput(stdout)).Run()
	if err != nil {
		return err
	}
	return unreadable(final.(explorer.Model).Problems())
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
	loaded, err := repo.Load(root)
	if err != nil {
		return err
	}
	if err := flagUnknownFeatures(refs.New(root), loaded.Notes); err != nil {
		return err
	}
	if asJSON {
		err = cli.WriteJSON(stdout, loaded.Notes)
	} else {
		err = writeTable(stdout, loaded.Notes)
	}
	if err != nil {
		return err
	}
	return unreadable(loaded.Problems)
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
	loaded, err := repo.Load(root)
	if err != nil {
		return err
	}
	open := questions.Open(loaded.Notes)
	if asJSON {
		err = cli.WriteJSON(stdout, open)
	} else {
		err = writeQuestions(stdout, open)
	}
	if err != nil {
		return err
	}
	return unreadable(loaded.Problems)
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
