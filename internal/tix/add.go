package tix

import (
	"fmt"
	"io"
	"regexp"
	"slices"
	"strconv"
	"time"

	"github.com/theroymind/railyard/internal/cli"
	"github.com/theroymind/railyard/internal/contracts/ticket"
	"github.com/theroymind/railyard/internal/contracts/timestamp"
	"github.com/theroymind/railyard/internal/contracts/validation"
	"github.com/theroymind/railyard/internal/tix/backend"
)

var addSpecification = cli.Specification{
	Options: []cli.Option{
		{Name: optionType, Type: cli.OptionTypeString},
		{Name: optionPriority, Type: cli.OptionTypeString},
		{Name: optionTag, Type: cli.OptionTypeString, IsMultiple: true},
		{Name: optionAcceptance, Type: cli.OptionTypeString, IsMultiple: true},
		{Name: optionDescription, Type: cli.OptionTypeString},
		{Name: optionDraft, Type: cli.OptionTypeBoolean},
		{Name: optionParent, Type: cli.OptionTypeString},
		{Name: optionDependency, Type: cli.OptionTypeString, IsMultiple: true},
	},
	AllowsPositionals: true,
}

var integerExpression = regexp.MustCompile(integerPattern)

func parsePriority(parsed cli.Arguments) (ticket.Priority, error) {
	value, exists := parsed.String(optionPriority)
	if !exists {
		return ticket.PriorityDefault, nil
	}
	return parsePriorityText(value)
}

// parsePriorityText reads a priority written as a base-10 integer.
func parsePriorityText(value string) (ticket.Priority, error) {
	if !integerExpression.MatchString(value) {
		return 0, invalidPriority(value)
	}
	priority, err := strconv.Atoi(value)
	if err != nil {
		return 0, invalidPriority(value)
	}
	return ticket.Priority(priority), nil
}

func invalidPriority(value string) error {
	var problems validation.Errors
	problems.Add(ticket.FieldPriority, problemPriority+strconv.Quote(value))
	return problems
}

func runAdd(arguments []string, _ io.Reader, stdout io.Writer) error {
	parsed, err := cli.Parse(arguments, addSpecification)
	if err != nil {
		return err
	}
	if len(parsed.Positionals) != 1 {
		return cli.NewUsageError(ErrExpectedOneTitle)
	}
	ticketType, err := parsed.RequiredString(optionType)
	if err != nil {
		return err
	}
	title, err := ticket.NormalizeTitle(parsed.Positionals[0])
	if err != nil {
		return err
	}
	store, err := openBackend()
	if err != nil {
		return err
	}
	parent, err := resolveParent(store, parsed)
	if err != nil {
		return err
	}
	dependencies, err := resolveAll(store, parsed.Strings(optionDependency))
	if err != nil {
		return err
	}
	priority, err := parsePriority(parsed)
	if err != nil {
		return err
	}
	status := ticket.StatusOpen
	if parsed.IsEnabled(optionDraft) {
		status = ticket.StatusDraft
	}
	description, _ := parsed.String(optionDescription)
	now := timestamp.Format(time.Now())
	created := ticket.Ticket{
		Title:        title,
		Type:         ticket.Type(ticketType),
		Status:       status,
		Priority:     &priority,
		Tags:         parsed.Strings(optionTag),
		Dependencies: dependencies,
		Parent:       parent,
		Description:  description,
		Acceptance:   ticket.NewCriteria(parsed.Strings(optionAcceptance)),
		CreatedAt:    now,
		UpdatedAt:    now,
	}
	id, err := store.Create(created)
	if err != nil {
		return err
	}
	_, err = fmt.Fprintln(stdout, id)
	return err
}

// resolveParent resolves the --parent reference, or returns "" when there is none.
func resolveParent(store backend.Backend, parsed cli.Arguments) (string, error) {
	reference, exists := parsed.String(optionParent)
	if !exists {
		return "", nil
	}
	return store.Resolve(reference)
}

// resolveAll resolves each reference to an id, keeping the first occurrence of each id in order, and fails on the
// first reference that matches no ticket or several.
func resolveAll(store backend.Backend, references []string) ([]string, error) {
	var ids []string
	for _, reference := range references {
		id, err := store.Resolve(reference)
		if err != nil {
			return nil, err
		}
		if !slices.Contains(ids, id) {
			ids = append(ids, id)
		}
	}
	return ids, nil
}
