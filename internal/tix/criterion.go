package tix

import (
	"fmt"
	"io"
	"strconv"
	"time"

	"github.com/theroymind/go-tix/contracts/ticket"
	"github.com/theroymind/go-tix/contracts/timestamp"
	"github.com/theroymind/go-tix/internal/tix/backend"
	"github.com/theroymind/railyard-cli"
)

var markSpecification = cli.Specification{
	Options:           []cli.Option{{Name: optionAll, Type: cli.OptionTypeBoolean}},
	AllowsPositionals: true,
}

var criterionSpecification = cli.Specification{
	Options:           []cli.Option{{Name: optionOpen, Type: cli.OptionTypeBoolean}},
	AllowsPositionals: true,
}

func runTick(arguments []string, _ io.Reader, _ io.Writer) error {
	return markCriteria(arguments, func(index int) backend.Change { return backend.CheckCriterion{Index: index} })
}

func runUntick(arguments []string, _ io.Reader, _ io.Writer) error {
	return markCriteria(arguments, func(index int) backend.Change { return backend.UncheckCriterion{Index: index} })
}

// markCriteria applies mark to each criterion number after the reference, or with --all to every criterion the
// ticket has, all in one write.
func markCriteria(arguments []string, mark func(index int) backend.Change) error {
	parsed, err := cli.Parse(arguments, markSpecification)
	if err != nil {
		return err
	}
	isAll := parsed.IsEnabled(optionAll)
	switch {
	case len(parsed.Positionals) == 0, len(parsed.Positionals) == 1 && !isAll:
		return cli.NewUsageError(ErrExpectedNumbers)
	case len(parsed.Positionals) > 1 && isAll:
		return cli.NewUsageError(ErrExpectedNumbersOrAll)
	}
	if isAll {
		return markAll(parsed.Positionals[0], mark)
	}
	changes := make([]backend.Change, 0, len(parsed.Positionals))
	for _, text := range parsed.Positionals[1:] {
		index, err := parseCriterionNumber(text)
		if err != nil {
			return err
		}
		changes = append(changes, mark(index))
	}
	return applyToReference(parsed.Positionals[0], changes)
}

// markAll applies mark to every criterion of the ticket the reference names, in one write. A ticket without
// criteria is left as it is.
func markAll(reference string, mark func(index int) backend.Change) error {
	store, err := openBackend()
	if err != nil {
		return err
	}
	id, err := store.Resolve(reference)
	if err != nil {
		return err
	}
	marked, err := store.Read(id)
	if err != nil {
		return err
	}
	if len(marked.Acceptance) == 0 {
		return nil
	}
	changes := make([]backend.Change, 0, len(marked.Acceptance)+1)
	for index := range marked.Acceptance {
		changes = append(changes, mark(index+1))
	}
	changes = append(changes, backend.SetField{Field: ticket.FieldUpdatedAt, Value: timestamp.Format(time.Now())})
	return store.Apply(id, changes)
}

// runAc dispatches on its first positional, since cli.Program has no nested commands: ac ls REF prints the
// criteria, or with --open only the unticked ones; ac set REF N TEXT replaces a criterion's text, and ac add REF
// TEXT appends one.
func runAc(arguments []string, _ io.Reader, stdout io.Writer) error {
	parsed, err := cli.Parse(arguments, criterionSpecification)
	if err != nil {
		return err
	}
	if len(parsed.Positionals) == 0 {
		return cli.NewUsageError(ErrExpectedSubcommand)
	}
	subcommand, rest := parsed.Positionals[0], parsed.Positionals[1:]
	if subcommand != subcommandLs && parsed.IsEnabled(optionOpen) {
		return cli.NewUsageError(ErrOpenNeedsLs)
	}
	switch subcommand {
	case subcommandLs:
		if len(rest) != 1 {
			return cli.NewUsageError(ErrExpectedLsCriteria)
		}
		return listCriteria(stdout, rest[0], parsed.IsEnabled(optionOpen))
	case subcommandSet:
		if len(rest) != 3 {
			return cli.NewUsageError(ErrExpectedSetCriterion)
		}
		index, err := parseCriterionNumber(rest[1])
		if err != nil {
			return err
		}
		return applyToReference(rest[0], []backend.Change{backend.SetCriterion{Index: index, Text: rest[2]}})
	case subcommandAdd:
		if len(rest) != 2 {
			return cli.NewUsageError(ErrExpectedAddCriterion)
		}
		return applyToReference(rest[0], []backend.Change{backend.AddCriterion{Text: rest[1]}})
	default:
		return cli.NewUsageError(fmt.Errorf("%w, got %s", ErrExpectedSubcommand, strconv.Quote(subcommand)))
	}
}

// listCriteria prints each criterion of the ticket the reference names as N. [x] TEXT or N. [ ] TEXT, numbered as
// tick takes them, or with isOpenOnly only the unticked ones. A backend whose criteria carry no done state shows
// each unticked.
func listCriteria(stdout io.Writer, reference string, isOpenOnly bool) error {
	store, err := openBackend()
	if err != nil {
		return err
	}
	id, err := store.Resolve(reference)
	if err != nil {
		return err
	}
	listed, err := store.Read(id)
	if err != nil {
		return err
	}
	for index, criterion := range listed.Acceptance {
		line := openCriterion
		if criterion.Done {
			if isOpenOnly {
				continue
			}
			line = checkedCriterion
		}
		if _, err := fmt.Fprintf(stdout, line, index+1, criterion.Text); err != nil {
			return err
		}
	}
	return nil
}

// parseCriterionNumber reads the 1-based number tix show prints. Whether the ticket has that many criteria is the
// backend's to say.
func parseCriterionNumber(text string) (int, error) {
	index, err := strconv.Atoi(text)
	if err != nil || index < 1 {
		return 0, cli.NewUsageError(fmt.Errorf("%s %w", strconv.Quote(text), ErrInvalidNumber))
	}
	return index, nil
}

// applyToReference applies the changes and updated_at to the ticket the reference names.
func applyToReference(reference string, changes []backend.Change) error {
	store, err := openBackend()
	if err != nil {
		return err
	}
	id, err := store.Resolve(reference)
	if err != nil {
		return err
	}
	changes = append(changes, backend.SetField{Field: ticket.FieldUpdatedAt, Value: timestamp.Format(time.Now())})
	return store.Apply(id, changes)
}
