package tix

import (
	"fmt"
	"io"
	"strconv"
	"strings"
	"time"

	"github.com/theroymind/railyard/internal/cli"
	"github.com/theroymind/railyard/internal/contracts/ticket"
	"github.com/theroymind/railyard/internal/contracts/timestamp"
	"github.com/theroymind/railyard/internal/tix/backend"
)

var setSpecification = cli.Specification{AllowsPositionals: true}

func runSet(arguments []string, _ io.Reader, _ io.Writer) error {
	parsed, err := cli.Parse(arguments, setSpecification)
	if err != nil {
		return err
	}
	if len(parsed.Positionals) < 2 {
		return cli.NewUsageError(ErrExpectedPairs)
	}
	changes, err := parsePairs(parsed.Positionals[1:])
	if err != nil {
		return err
	}
	store, err := openBackend()
	if err != nil {
		return err
	}
	id, apply, err := resolveAny(store, parsed.Positionals[0])
	if err != nil {
		return err
	}
	return apply(id, changes)
}

// pairValue gives a value the type its field holds: a priority is an integer, and a list field is refused, since
// one value can't say whether to replace the list or add to it. Any other value stays text.
func pairValue(key, value string) (any, error) {
	switch key {
	case ticket.FieldPriority:
		return parsePriorityText(value)
	case ticket.FieldDependencies:
		return nil, fmt.Errorf("%s %w; %s", key, ErrListField, dependenciesHint)
	case ticket.FieldTags, ticket.FieldAcceptance:
		return nil, fmt.Errorf("%s %w", key, ErrListField)
	default:
		return value, nil
	}
}

// parsePairs turns each KEY=VALUE into a SetField, splitting at the first "=", and adds updated_at unless a pair
// sets it, since every write changes it.
func parsePairs(pairs []string) ([]backend.Change, error) {
	changes := make([]backend.Change, 0, len(pairs)+1)
	setsUpdatedAt := false
	for _, pair := range pairs {
		key, value, hasSeparator := strings.Cut(pair, pairSeparator)
		if !hasSeparator || key == "" {
			return nil, cli.NewUsageError(fmt.Errorf("%w, got %s", ErrExpectedPair, strconv.Quote(pair)))
		}
		setsUpdatedAt = setsUpdatedAt || key == ticket.FieldUpdatedAt
		typed, err := pairValue(key, value)
		if err != nil {
			return nil, cli.NewUsageError(err)
		}
		changes = append(changes, backend.SetField{Field: key, Value: typed})
	}
	if !setsUpdatedAt {
		changes = append(changes, backend.SetField{Field: ticket.FieldUpdatedAt, Value: timestamp.Format(time.Now())})
	}
	return changes, nil
}
