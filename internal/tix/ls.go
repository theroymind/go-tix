package tix

import (
	"fmt"
	"io"

	"github.com/theroymind/go-tix/contracts/ticket"
	"github.com/theroymind/go-tix/contracts/validation"
	"github.com/theroymind/railyard-cli"
)

var lsSpecification = cli.Specification{
	Options: []cli.Option{
		{Name: optionStatus, Type: cli.OptionTypeString},
		{Name: optionType, Type: cli.OptionTypeString},
		{Name: optionTag, Type: cli.OptionTypeString},
		{Name: optionParent, Type: cli.OptionTypeString},
		{Name: optionAll, Type: cli.OptionTypeBoolean},
		{Name: optionJSON, Type: cli.OptionTypeBoolean},
	},
}

func runLs(arguments []string, _ io.Reader, stdout io.Writer) error {
	parsed, err := cli.Parse(arguments, lsSpecification)
	if err != nil {
		return err
	}
	status, err := parseOption(parsed, optionStatus, ticket.Statuses)
	if err != nil {
		return err
	}
	tag, _ := parsed.String(optionTag)
	store, err := openBackend()
	if err != nil {
		return err
	}
	ticketType, err := parseType(parsed, store)
	if err != nil {
		return err
	}
	var parent string
	if reference, exists := parsed.String(optionParent); exists {
		if parent, err = store.Resolve(reference); err != nil {
			return err
		}
	}
	tickets, err := store.List()
	if err != nil {
		return err
	}
	listed := filter{
		status:           status,
		ticketType:       ticketType,
		tag:              tag,
		parent:           parent,
		includesResolved: parsed.IsEnabled(optionAll),
	}.apply(tickets)
	properties, err := readPropertyColumns(store)
	if err != nil {
		return err
	}
	if parsed.IsEnabled(optionJSON) {
		return writeTickets(stdout, listed, properties, true)
	}
	ordered, depths := nest(listed)
	return writeTable(stdout, ordered, depths, properties)
}

// nest orders the tickets as a tree, each child after its parent in the order it was listed, and returns each
// ticket's depth by id. A ticket whose parent isn't listed is a root, so a filter never hides a match.
func nest(tickets []ticket.Ticket) ([]ticket.Ticket, map[string]int) {
	listed := make(map[string]bool, len(tickets))
	for _, row := range tickets {
		listed[row.ID] = true
	}
	children := map[string][]ticket.Ticket{}
	var roots []ticket.Ticket
	for _, row := range tickets {
		if row.Parent != "" && row.Parent != row.ID && listed[row.Parent] {
			children[row.Parent] = append(children[row.Parent], row)
		} else {
			roots = append(roots, row)
		}
	}
	ordered := make([]ticket.Ticket, 0, len(tickets))
	depths := make(map[string]int, len(tickets))
	var visit func(row ticket.Ticket, depth int)
	visit = func(row ticket.Ticket, depth int) {
		if _, seen := depths[row.ID]; seen {
			return
		}
		depths[row.ID] = depth
		ordered = append(ordered, row)
		for _, child := range children[row.ID] {
			visit(child, depth+1)
		}
	}
	for _, row := range roots {
		visit(row, 0)
	}
	// A parent cycle has no root; list its tickets flat rather than drop them.
	for _, row := range tickets {
		visit(row, 0)
	}
	return ordered, depths
}

// parseOption returns the option's value, or "" when it's absent, and rejects a value outside options as a usage error.
func parseOption[Option ~string](parsed cli.Arguments, name string, options []Option) (Option, error) {
	value, exists := parsed.String(name)
	if !exists {
		return "", nil
	}
	for _, option := range options {
		if Option(value) == option {
			return option, nil
		}
	}
	return "", cli.NewUsageError(fmt.Errorf("%s%s %q %w: %s", longOptionStart, name, value, ErrInvalidOption, validation.InvalidOption(options)))
}
