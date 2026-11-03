package tix

import (
	"fmt"
	"io"
	"strconv"
	"time"

	"github.com/theroymind/railyard/internal/cli"
	"github.com/theroymind/railyard/internal/contracts/ticket"
	"github.com/theroymind/railyard/internal/contracts/timestamp"
	"github.com/theroymind/railyard/internal/tix/backend"
)

var criterionSpecification = cli.Specification{AllowsPositionals: true}

func runTick(arguments []string, _ io.Reader, _ io.Writer) error {
	return markCriteria(arguments, func(index int) backend.Change { return backend.CheckCriterion{Index: index} })
}

func runUntick(arguments []string, _ io.Reader, _ io.Writer) error {
	return markCriteria(arguments, func(index int) backend.Change { return backend.UncheckCriterion{Index: index} })
}

// markCriteria applies mark to each criterion number after the reference, all in one write.
func markCriteria(arguments []string, mark func(index int) backend.Change) error {
	parsed, err := cli.Parse(arguments, criterionSpecification)
	if err != nil {
		return err
	}
	if len(parsed.Positionals) < 2 {
		return cli.NewUsageError(ErrExpectedNumbers)
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

// runAc dispatches on its first positional, since cli.Program has no nested commands: ac set REF N TEXT replaces
// a criterion's text, and ac add REF TEXT appends one.
func runAc(arguments []string, _ io.Reader, _ io.Writer) error {
	parsed, err := cli.Parse(arguments, criterionSpecification)
	if err != nil {
		return err
	}
	if len(parsed.Positionals) == 0 {
		return cli.NewUsageError(ErrExpectedSubcommand)
	}
	subcommand, rest := parsed.Positionals[0], parsed.Positionals[1:]
	switch subcommand {
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
