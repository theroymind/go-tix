// Package command provides the sketch commands: the explorer, which bare sketch opens on the repo tree; ls, which
// lists the repo's .sketch.md notes as a table or JSON; questions, which lists their questions; and help, which
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
  ls [--json]                lists every note under the repo root with the file it plans, + when that file is
                             new or ~ when it exists, its feature and status, how many links it has and how
                             many questions it asks; in a tix repo, a feature tix has no ticket for is flagged
                             (not in tix); --json prints an array of notes, each with its whole body, its links
                             as edges, + (new) when a note plans the linked file or ~ (changed) when it exists,
                             and its questions
  questions [--json] [--all] lists each open question as NOTE:LINE: QUESTION, by note and then line; --all adds
                             answered ones, each answer line below its question after > ; --json prints an array
                             of {path, line, text, answer}
  help                       prints this

A note is a plain markdown file beside the file it plans, named after it with .sketch.md added:
src/useOrbit.ts.sketch.md plans src/useOrbit.ts, whether or not that file exists yet. It's a blueprint for
whoever builds the file, person or agent, and its whole body is read as written. sketch reads only these lines;
every other line is free text, never an error:

  ---                         optional front matter, only at the very top, between two --- lines:
  feature: FEATURE            the tix feature the note belongs to
  status: draft|ready         whether the note is still being written or is ready to build from
  ---
  -> PATH [: LABEL]           a link to another file, from the repo root, or from the note's folder when PATH
                              starts with ./ or ../, such as -> ./camera.ts : reads the target
  ? QUESTION                  an open question
  > ANSWER                    answers the question above it, on the line right under it or under one blank
                              line, and runs on over the > lines that follow

Markers start the line. A > line that isn't under a question is free text, and so is a link that leaves the repo.
To describe what the file holds, these markers are a suggested, language-neutral convention; sketch doesn't
parse them: #fn for a function, #type for a type, #component for a UI component, #flow for a sequence of steps,
and #test for a behaviour to test. For example, src/useOrbit.ts.sketch.md:

  ---
  feature: camera
  status: draft
  ---
  Orbit the camera around the fight.

  #fn useOrbit(target: Ref<Vec3>) -> { yaw, pitch }
  #flow drag -> update yaw and pitch -> ease back when released
  #test releasing the drag eases back to rest
  -> ./camera.ts : reads the camera's target

  ? how fast does it ease back
  > about 1 rad/s,
  > slower when zoomed in

The repo root is the nearest directory above holding .tix or .git. In a git repo, sketch sees the files git does,
tracked or not but never ignored; outside one, it skips hidden folders and node_modules. ls and questions list
every note they can read, and then fail naming any they couldn't. In a tix repo, when tix can't be run, ls warns
and lists the notes without checking their features. To quiet markdownlint's MD018 and MD041 on notes, see the
README's .markdownlint-cli2.yaml override.
`

const (
	optionJSON  = "json"
	optionAll   = "all"
	tableHeader = "NOTE\tTARGET\tFEATURE\tSTATUS\tLINKS\tQUESTIONS"
	tablePad    = 2
	noValue     = "-"

	unknownFeatureMarker = " (not in tix)"
	answerPrefix         = "  > "
	warningFormat        = "%s: warning: not checking features: %v\n"
)

const programName = "sketch"

var rootMarkers = []string{".tix", ".git"}

var ErrUnreadableNotes = errors.New("some notes couldn't be read")

// newProgram returns sketch's commands, which print warnings on stderr.
func newProgram(stderr io.Writer) cli.Program {
	return cli.Program{
		Name:  programName,
		Usage: usage,
		Commands: map[string]cli.Command{
			"ls": func(arguments []string, _ io.Reader, stdout io.Writer) error {
				return runList(arguments, stdout, stderr)
			},
			"questions": runQuestions,
		},
	}
}

// Run runs sketch with arguments: the explorer when there are none and stdin and stdout are a terminal, otherwise
// the command they name, with no arguments printing the usage.
func Run(arguments []string, stdin io.Reader, stdout, stderr io.Writer) cli.ExitCode {
	if len(arguments) > 0 || !isTerminal(stdin) || !isTerminal(stdout) {
		return newProgram(stderr).Run(arguments, stdin, stdout, stderr)
	}
	if err := runExplorer(stdin, stdout); err != nil {
		_, _ = fmt.Fprintf(stderr, "%s: %v\n", programName, err)
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

// runList prints every note under the repo root that reads and validates, marked against the repo, as a table
// or a JSON array; in a tix repo, the table flags a feature tix has no ticket for. It then fails naming each note
// that didn't read, so one bad note never hides the rest.
func runList(arguments []string, stdout, stderr io.Writer) error {
	parsed, err := parseOptions(arguments, optionJSON)
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
	unknown := unknownFeatures(refs.New(root), loaded.Notes, stderr)
	if parsed.IsEnabled(optionJSON) {
		err = cli.WriteJSON(stdout, loaded.Notes)
	} else {
		err = writeTable(stdout, loaded.Notes, unknown)
	}
	if err != nil {
		return err
	}
	return unreadable(loaded.Problems)
}

// runQuestions prints each open question under the repo root, or with --all every question, with the note and
// line that asks it, as path:line: text lines or a JSON array, and then fails naming each note that didn't read.
func runQuestions(arguments []string, _ io.Reader, stdout io.Writer) error {
	parsed, err := parseOptions(arguments, optionJSON, optionAll)
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
	listed := questions.Open(loaded.Notes)
	if parsed.IsEnabled(optionAll) {
		listed = questions.All(loaded.Notes)
	}
	if parsed.IsEnabled(optionJSON) {
		err = cli.WriteJSON(stdout, listed)
	} else {
		err = writeQuestions(stdout, listed)
	}
	if err != nil {
		return err
	}
	return unreadable(loaded.Problems)
}

// parseOptions parses a command's arguments, which take only the named boolean options.
func parseOptions(arguments []string, names ...string) (cli.Arguments, error) {
	var options []cli.Option
	for _, name := range names {
		options = append(options, cli.Option{Name: name, Type: cli.OptionTypeBoolean})
	}
	return cli.Parse(arguments, cli.Specification{Options: options})
}

// unknownFeatures returns the set of features notes name that tix has no ticket for. It asks tix once, and only
// in a tix repo where some note names a feature; outside one, any feature is accepted. When tix can't be run, it
// warns on stderr and accepts every feature, so the notes still list.
func unknownFeatures(resolver refs.Resolver, notes []sketch.Note, stderr io.Writer) map[string]bool {
	namesFeature := func(value sketch.Note) bool { return value.Feature != "" }
	if !resolver.IsTixRepo() || !slices.ContainsFunc(notes, namesFeature) {
		return nil
	}
	tickets, err := resolver.Tickets()
	if err != nil {
		_, _ = fmt.Fprintf(stderr, warningFormat, programName, err)
		return nil
	}
	unknown := map[string]bool{}
	for _, value := range notes {
		if value.Feature != "" && !tickets[value.Feature] {
			unknown[value.Feature] = true
		}
	}
	return unknown
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
// marked when it's among unknown, its status, and how many links and questions it has. A feature or status the
// note doesn't have shows as -.
func writeTable(writer io.Writer, notes []sketch.Note, unknown map[string]bool) error {
	table := tabwriter.NewWriter(writer, 0, 0, tablePad, ' ', 0)
	if _, err := fmt.Fprintln(table, tableHeader); err != nil {
		return err
	}
	for _, value := range notes {
		feature := orNoValue(value.Feature)
		if unknown[value.Feature] {
			feature += unknownFeatureMarker
		}
		_, err := fmt.Fprintf(table, "%s\t%s\t%s\t%s\t%d\t%d\n", value.Path, value.Mark.Symbol()+value.Target, feature,
			orNoValue(string(value.Status)), len(value.Edges), len(value.Questions))
		if err != nil {
			return err
		}
	}
	return table.Flush()
}

// writeQuestions writes one path:line: text line per question, the form editors and grep use, followed by each
// line of its answer, if it has one, after > .
func writeQuestions(writer io.Writer, listed []questions.Question) error {
	for _, question := range listed {
		if _, err := fmt.Fprintf(writer, "%s:%d: %s\n", question.Path, question.Line, question.Text); err != nil {
			return err
		}
		if question.Answer == "" {
			continue
		}
		for _, line := range strings.Split(question.Answer, "\n") {
			if _, err := fmt.Fprintln(writer, answerPrefix+line); err != nil {
				return err
			}
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
