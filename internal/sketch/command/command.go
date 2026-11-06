// Package command provides the sketch commands: an edit under -C, which runs one grammar command or a stdin batch
// on a feature's sketch and saves it, show, which prints the sketch as a table, JSON or markdown, and questions and
// answer, which list a sketch's open questions and record their answers.
package command

import (
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	cli "github.com/theroymind/railyard-cli"

	"github.com/theroymind/go-tix/contracts/sketch"
	"github.com/theroymind/go-tix/internal/sketch/grammar"
	"github.com/theroymind/go-tix/internal/sketch/marks"
	"github.com/theroymind/go-tix/internal/sketch/questions"
	"github.com/theroymind/go-tix/internal/sketch/refs"
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
                         (marks each item ~ when its file already names it, + when it's new)
  questions FEATURE [--json]
                         (the open questions, each with its number and the context of its item)
  answer FEATURE N TEXT  (records TEXT as question N's answer and closes it)
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

Lines are numbered from 1, as show --md prints them, and questions from 1 across the whole sketch, answered ones
included, so answering one never renumbers the rest. The repo root is the nearest directory above holding .tix
or .git. Inside a tix repo, FEATURE must be a tix ticket, and its sketch.yml sits beside the ticket's note;
outside one, any name works, and it lives in docs/features/FEATURE. Send several commands as one stdin batch
rather than as parallel calls: a batch lands whole or not at all.
`

const (
	optionJSON     = "json"
	optionMarkdown = "md"
	contextOption  = "-C"
	stdinCommand   = "-"
)

var rootMarkers = []string{".tix", ".git"}

// runChild runs tix for refs, a variable so tests can stand in for the tix on PATH.
var runChild refs.Runner = cli.RunChildIn

var (
	ErrBothFormats  = errors.New("--json and --md can't both be given")
	ErrNeedsFeature = errors.New("needs exactly one FEATURE")
	ErrNeedsCommand = errors.New("needs a context and a command")
	ErrNoCommands   = errors.New("stdin holds no commands")
	ErrNeedsAnswer  = errors.New("needs FEATURE, a question number and its answer")
)

var Program = cli.Program{
	Name:  "sketch",
	Usage: usage,
	Commands: map[string]cli.Command{
		contextOption: runEdit,
		"show":        runShow,
		"questions":   runQuestions,
		"answer":      runAnswer,
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
	root, dir, err := locate(at.Feature)
	if err != nil {
		return err
	}
	next := at
	err = store.Update(root, dir, func(value sketch.Sketch) (sketch.Sketch, error) {
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
	feature, root, value, err := loadFeature(parsed.Positionals[0])
	if err != nil {
		return err
	}
	reader := marks.NewReader(root)
	mark := func(path, name string) (string, error) {
		itemMark, err := reader.Mark(path, name)
		return string(itemMark), err
	}
	switch {
	case asJSON:
		return cli.WriteJSON(stdout, value)
	case asMarkdown:
		return show.Markdown(stdout, feature, value, mark)
	default:
		return show.Table(stdout, value, mark)
	}
}

func runQuestions(arguments []string, _ io.Reader, stdout io.Writer) error {
	parsed, err := cli.Parse(arguments, cli.Specification{
		Options:           []cli.Option{{Name: optionJSON, Type: cli.OptionTypeBoolean}},
		AllowsPositionals: true,
	})
	if err != nil {
		return err
	}
	if len(parsed.Positionals) != 1 {
		return cli.NewUsageError(ErrNeedsFeature)
	}
	feature, _, value, err := loadFeature(parsed.Positionals[0])
	if err != nil {
		return err
	}
	open := questions.Open(feature, value)
	if parsed.IsEnabled(optionJSON) {
		return cli.WriteJSON(stdout, open)
	}
	return show.Questions(stdout, open)
}

// runAnswer records an answer. Its words are joined with single spaces, as an edit's are.
func runAnswer(arguments []string, _ io.Reader, _ io.Writer) error {
	if len(arguments) < 3 {
		return cli.NewUsageError(ErrNeedsAnswer)
	}
	feature, err := parseFeature(arguments[0])
	if err != nil {
		return err
	}
	number, err := strconv.Atoi(arguments[1])
	if err != nil {
		return cli.NewUsageError(fmt.Errorf("%w: %q isn't a question number", ErrNeedsAnswer, arguments[1]))
	}
	root, dir, err := locate(feature)
	if err != nil {
		return err
	}
	answer := strings.Join(arguments[2:], " ")
	return store.Update(root, dir, func(value sketch.Sketch) (sketch.Sketch, error) {
		return questions.Answer(value, number, answer)
	})
}

// parseFeature reads a FEATURE argument, refusing a deeper context.
func parseFeature(argument string) (string, error) {
	at, err := grammar.ParseContext(argument)
	if err != nil || at.Task != "" {
		return "", cli.NewUsageError(fmt.Errorf("%w: %q", ErrNeedsFeature, argument))
	}
	return at.Feature, nil
}

// loadFeature reads a FEATURE argument and loads its sketch, returning the feature, the repo root and the sketch.
func loadFeature(argument string) (string, string, sketch.Sketch, error) {
	feature, err := parseFeature(argument)
	if err != nil {
		return "", "", sketch.Sketch{}, err
	}
	root, dir, err := locate(feature)
	if err != nil {
		return "", "", sketch.Sketch{}, err
	}
	value, err := store.Load(dir)
	return feature, root, value, err
}

// locate returns the repo root and the directory the feature's sketch lives in.
func locate(feature string) (string, string, error) {
	root, err := repoRootFromWorkingDir()
	if err != nil {
		return "", "", err
	}
	dir, err := refs.Resolver{Root: root, Run: runChild}.FeatureDir(feature)
	return root, dir, err
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
