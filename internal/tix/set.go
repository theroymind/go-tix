package tix

import (
	"fmt"
	"io"
	"strconv"
	"strings"
	"time"

	"go.yaml.in/yaml/v3"

	"github.com/theroymind/go-tix/contracts/ticket"
	"github.com/theroymind/go-tix/contracts/timestamp"
	"github.com/theroymind/go-tix/internal/tix/backend"
	"github.com/theroymind/railyard-cli"
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

// pairValue gives a value the type its field holds: a priority is an integer, and a value written as a flow list,
// such as [a, b], is a list of strings that replaces the field's value; [] is an empty one. A wikilink such as
// [[slug]] stays text. Tags take only a list. Dependencies and criteria have commands of their own, which check
// what a raw write can't, so set refuses them. Any other value stays text.
func pairValue(key, value string) (any, error) {
	isList := isListText(value)
	switch {
	case key == ticket.FieldPriority:
		return parsePriorityText(value)
	case key == ticket.FieldDependencies:
		return nil, fmt.Errorf("%s %w; %s", key, ErrListField, dependenciesHint)
	case key == ticket.FieldAcceptance:
		return nil, fmt.Errorf("%s %w; %s", key, ErrListField, criteriaHint)
	case key == ticket.FieldTags && !isList:
		return nil, fmt.Errorf("%s %w; %s", key, ErrListField, listHint)
	case isList:
		return parseList(key, value)
	default:
		return value, nil
	}
}

func isListText(value string) bool {
	isBracketed := strings.HasPrefix(value, listStart) && strings.HasSuffix(value, listEnd)
	isWikilink := strings.HasPrefix(value, wikilinkStart)
	return isBracketed && !isWikilink
}

// parseList reads a YAML flow list of strings.
func parseList(key, value string) ([]string, error) {
	items := []string{}
	if err := yaml.Unmarshal([]byte(value), &items); err != nil {
		return nil, fmt.Errorf("%s=%s %w", key, value, ErrInvalidList)
	}
	return items, nil
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
