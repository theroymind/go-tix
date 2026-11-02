package tix

import (
	"fmt"
	"io"
	"regexp"
	"strconv"
	"time"

	"github.com/theroymind/railyard/internal/cli"
	"github.com/theroymind/railyard/internal/contracts/ticket"
	"github.com/theroymind/railyard/internal/contracts/timestamp"
	"github.com/theroymind/railyard/internal/contracts/validation"
)

var addSpecification = cli.Specification{
	Options: []cli.Option{
		{Name: optionType, Type: cli.OptionTypeString},
		{Name: optionPriority, Type: cli.OptionTypeString},
		{Name: optionTag, Type: cli.OptionTypeString, IsMultiple: true},
		{Name: optionAcceptance, Type: cli.OptionTypeString, IsMultiple: true},
		{Name: optionDescription, Type: cli.OptionTypeString},
		{Name: optionDraft, Type: cli.OptionTypeBoolean},
	},
	AllowsPositionals: true,
}

var integerExpression = regexp.MustCompile(integerPattern)

func parsePriority(parsed cli.Arguments) (ticket.Priority, error) {
	value, exists := parsed.String(optionPriority)
	if !exists {
		return ticket.PriorityDefault, nil
	}
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
	store, err := openBackend()
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
		Title:       parsed.Positionals[0],
		Type:        ticket.Type(ticketType),
		Status:      status,
		Priority:    &priority,
		Tags:        parsed.Strings(optionTag),
		Description: description,
		Acceptance:  ticket.NewCriteria(parsed.Strings(optionAcceptance)),
		CreatedAt:   now,
		UpdatedAt:   now,
	}
	id, err := store.Create(created)
	if err != nil {
		return err
	}
	_, err = fmt.Fprintln(stdout, id)
	return err
}
