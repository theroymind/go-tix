package rankeval

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strconv"

	"github.com/theroymind/railyard/internal/cli"
)

var runSpecification = cli.Specification{
	Options: []cli.Option{
		{Name: optionVault, Type: cli.OptionTypeString},
		{Name: optionModel, Type: cli.OptionTypeString, IsMultiple: true},
		{Name: optionEndpoint, Type: cli.OptionTypeString},
		{Name: optionDepth, Type: cli.OptionTypeString},
		{Name: optionTop, Type: cli.OptionTypeString},
		{Name: optionRuns, Type: cli.OptionTypeString},
		{Name: optionNotes, Type: cli.OptionTypeString},
	},
}

type runOptions struct {
	vault    string
	models   []string
	endpoint string
	depth    int
	top      int
	runs     int
	notes    string
}

func runRun(arguments []string, _ io.Reader, stdout io.Writer) error {
	options, err := parseRunOptions(arguments)
	if err != nil {
		return err
	}
	store := vault{root: options.vault}
	repo := repository{root: options.vault}
	cases, skipped, err := buildCases(store, repo, options.depth, options.notes)
	if err != nil {
		return err
	}
	result := report{
		Vault:    options.vault,
		Depth:    options.depth,
		Top:      options.top,
		Cases:    len(cases),
		Decisive: decisiveCases(cases, options.top),
		Skipped:  skipped,
		Models:   []modelReport{},
	}
	for _, name := range options.models {
		ranker := newRanker(name, options.endpoint)
		model := modelReport{Model: name, Runs: []runReport{}}
		for run := 0; run < options.runs && model.Error == ""; run++ {
			fmt.Fprintf(os.Stderr, "rankeval: %s run %d of %d over %d cases\n", name, run+1, options.runs, len(cases))
			ran, err := runCases(ranker, cases, options.top)
			if err != nil {
				fmt.Fprintf(os.Stderr, "rankeval: %s: %v\n", name, err)
				model.Error = err.Error()
				continue
			}
			model.Runs = append(model.Runs, ran)
		}
		model.PrecisionSpread = precisionSpread(model.Runs)
		result.Models = append(result.Models, model)
	}
	return cli.WriteJSON(stdout, result)
}

func parseRunOptions(arguments []string) (runOptions, error) {
	parsed, err := cli.Parse(arguments, runSpecification)
	if err != nil {
		return runOptions{}, err
	}
	vaultPath, err := parsed.RequiredString(optionVault)
	if err != nil {
		return runOptions{}, err
	}
	absolute, err := filepath.Abs(vaultPath)
	if err != nil {
		return runOptions{}, err
	}
	depth, err := parseCount(parsed, optionDepth, DefaultDepth, 0)
	if err != nil {
		return runOptions{}, err
	}
	top, err := parseCount(parsed, optionTop, DefaultTop, 1)
	if err != nil {
		return runOptions{}, err
	}
	runs, err := parseCount(parsed, optionRuns, DefaultRuns, 1)
	if err != nil {
		return runOptions{}, err
	}
	options := runOptions{
		vault:    absolute,
		models:   parsed.Strings(optionModel),
		endpoint: DefaultEndpoint,
		depth:    depth,
		top:      top,
		runs:     runs,
		notes:    DefaultNotesDirectory,
	}
	if len(options.models) == 0 {
		options.models = []string{ModelNone}
	}
	if endpoint, exists := parsed.String(optionEndpoint); exists {
		options.endpoint = endpoint
	}
	if notes, exists := parsed.String(optionNotes); exists {
		options.notes = notes
	}
	return options, nil
}

// parseCount reads a whole-number option no smaller than minimum, or fallback when it is absent.
func parseCount(parsed cli.Arguments, name string, fallback, minimum int) (int, error) {
	text, exists := parsed.String(name)
	if !exists {
		return fallback, nil
	}
	value, err := strconv.Atoi(text)
	if err != nil || value < minimum {
		return 0, cli.NewUsageError(fmt.Errorf("--%s %s %w %d", name, strconv.Quote(text), ErrInvalidNumber, minimum))
	}
	return value, nil
}
