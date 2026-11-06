// Package command provides the sketch commands: a one-shot edit under -C, which runs one grammar command on a
// feature's sketch and saves it, and show, which prints the sketch as a table, JSON or markdown.
package command

import (
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	cli "github.com/theroymind/railyard-cli"

	"github.com/theroymind/go-tix/contracts/sketch"
	"github.com/theroymind/go-tix/internal/sketch/grammar"
	"github.com/theroymind/go-tix/internal/sketch/show"
	"github.com/theroymind/go-tix/internal/sketch/store"
)

const usage = `usage: sketch <command> [arguments]

commands:
  -C FEATURE[/TASK[/FILE[:ITEM]]] VERB [ARGS]
                         (runs one command at that context, saves, and prints the context it leaves)
  -C FEATURE[/TASK[/FILE[:ITEM]]] -
                         (runs each line of stdin as a command, each at the context the last one left, and
                         saves once, only when every line succeeds)
  show FEATURE [--json | --md]
  help

verbs:
  task NAME, file PATH   add the task or file, or open it
  fn (f), type (t), component, hook, flow, tune, rule, test NAME
                         add an item to the file, or open the one of that name
  in (i), out (o), case (c), ask (a), rule (r), field, prop TEXT
                         add a line to the item; rule adds a line inside an item and an item at file level
  mv NAME                rename the item, else the file, else the task
  rm [N]                 remove line N of the item, else the item, file or task
  set N TEXT             replace line N's text

Lines are numbered from 1, as show --md prints them. A sketch lives in docs/features/FEATURE/sketch.yml from
the repo root, the nearest directory above holding .tix or .git. Send several commands as one stdin batch
rather than as parallel calls: a batch lands whole or not at all.
`

const (
	optionJSON     = "json"
	optionMarkdown = "md"
	contextOption  = "-C"
	stdinCommand   = "-"
)

var rootMarkers = []string{".tix", ".git"}

var (
	ErrBothFormats  = errors.New("--json and --md can't both be given")
	ErrNeedsFeature = errors.New("needs exactly one FEATURE")
	ErrNeedsCommand = errors.New("needs a context and a command")
	ErrNoCommands   = errors.New("stdin holds no commands")
)

var Program = cli.Program{
	Name:  "sketch",
	Usage: usage,
	Commands: map[string]cli.Command{
		contextOption: runEdit,
		"show":        runShow,
	},
}

// runEdit runs one grammar command at the context its first argument names, or, when the command is -, every
// line of stdin in order, each at the context the line before it left. The command's words are joined with
// single spaces, so a quoted argument and separate words read the same. The sketch is saved once, only when
// every command succeeds, and the context the last one leaves is printed.
func runEdit(arguments []string, stdin io.Reader, stdout io.Writer) error {
	if len(arguments) < 2 {
		return cli.NewUsageError(ErrNeedsCommand)
	}
	at, err := grammar.ParseContext(arguments[0])
	if err != nil {
		return cli.NewUsageError(err)
	}
	isBatch := len(arguments) == 2 && arguments[1] == stdinCommand
	var lines []string
	if isBatch {
		if lines, err = readLines(stdin); err != nil {
			return err
		}
	}
	root, err := repoRootFromWorkingDir()
	if err != nil {
		return err
	}
	next := at
	err = store.Update(root, store.FeatureDir(root, at.Feature), func(value sketch.Sketch) (sketch.Sketch, error) {
		if isBatch {
			return applyBatch(value, at, lines, &next)
		}
		edited, leaves, err := grammar.Apply(value, at, strings.Join(arguments[1:], " "))
		next = leaves
		return edited, err
	})
	if err != nil {
		return err
	}
	_, err = fmt.Fprintln(stdout, next)
	return err
}

// applyBatch applies each line in turn, carrying the context, and names the 1-based line that fails. Blank lines
// are skipped, and a batch of none is refused. next receives the context the last line leaves.
func applyBatch(value sketch.Sketch, at grammar.Context, lines []string, next *grammar.Context) (sketch.Sketch, error) {
	applied := 0
	for index, line := range lines {
		if strings.TrimSpace(line) == "" {
			continue
		}
		var err error
		if value, at, err = grammar.Apply(value, at, line); err != nil {
			return value, fmt.Errorf("line %d: %w", index+1, err)
		}
		applied++
	}
	if applied == 0 {
		return value, cli.NewUsageError(ErrNoCommands)
	}
	*next = at
	return value, nil
}

func readLines(stdin io.Reader) ([]string, error) {
	text, err := io.ReadAll(stdin)
	if err != nil {
		return nil, err
	}
	return strings.Split(strings.ReplaceAll(string(text), "\r\n", "\n"), "\n"), nil
}

func runShow(arguments []string, _ io.Reader, stdout io.Writer) error {
	parsed, err := cli.Parse(arguments, cli.Specification{
		Options: []cli.Option{
			{Name: optionJSON, Type: cli.OptionTypeBoolean},
			{Name: optionMarkdown, Type: cli.OptionTypeBoolean},
		},
		AllowsPositionals: true,
	})
	if err != nil {
		return err
	}
	if len(parsed.Positionals) != 1 {
		return cli.NewUsageError(ErrNeedsFeature)
	}
	asJSON, asMarkdown := parsed.IsEnabled(optionJSON), parsed.IsEnabled(optionMarkdown)
	if asJSON && asMarkdown {
		return cli.NewUsageError(ErrBothFormats)
	}
	at, err := grammar.ParseContext(parsed.Positionals[0])
	if err != nil || at.Task != "" {
		return cli.NewUsageError(fmt.Errorf("%w: %q", ErrNeedsFeature, parsed.Positionals[0]))
	}
	root, err := repoRootFromWorkingDir()
	if err != nil {
		return err
	}
	value, err := store.Load(store.FeatureDir(root, at.Feature))
	if err != nil {
		return err
	}
	switch {
	case asJSON:
		return cli.WriteJSON(stdout, value)
	case asMarkdown:
		return show.Markdown(stdout, at.Feature, value)
	default:
		return show.Table(stdout, value)
	}
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
