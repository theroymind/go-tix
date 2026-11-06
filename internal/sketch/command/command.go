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
the repo root, the nearest directory above holding .tix or .git.
`

const (
	optionJSON     = "json"
	optionMarkdown = "md"
	contextOption  = "-C"
)

var rootMarkers = []string{".tix", ".git"}

var (
	ErrBothFormats  = errors.New("--json and --md can't both be given")
	ErrNeedsFeature = errors.New("needs exactly one FEATURE")
	ErrNeedsCommand = errors.New("needs a context and a command")
)

var Program = cli.Program{
	Name:  "sketch",
	Usage: usage,
	Commands: map[string]cli.Command{
		contextOption: runEdit,
		"show":        runShow,
	},
}

// runEdit runs one grammar command at the context its first argument names. The command's words are joined
// with single spaces, so a quoted argument and separate words read the same.
func runEdit(arguments []string, _ io.Reader, stdout io.Writer) error {
	if len(arguments) < 2 {
		return cli.NewUsageError(ErrNeedsCommand)
	}
	at, err := grammar.ParseContext(arguments[0])
	if err != nil {
		return cli.NewUsageError(err)
	}
	dir, err := featureDir(at.Feature)
	if err != nil {
		return err
	}
	line := strings.Join(arguments[1:], " ")
	var next grammar.Context
	err = store.Update(dir, func(value sketch.Sketch) (sketch.Sketch, error) {
		var edited sketch.Sketch
		var err error
		edited, next, err = grammar.Apply(value, at, line)
		return edited, err
	})
	if err != nil {
		return err
	}
	_, err = fmt.Fprintln(stdout, next)
	return err
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
	dir, err := featureDir(at.Feature)
	if err != nil {
		return err
	}
	value, err := store.Load(dir)
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

func featureDir(feature string) (string, error) {
	workingDir, err := os.Getwd()
	if err != nil {
		return "", err
	}
	return store.FeatureDir(repoRoot(workingDir), feature), nil
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
