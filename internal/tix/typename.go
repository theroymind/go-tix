package tix

import (
	"fmt"
	"strings"

	"github.com/theroymind/go-tix/contracts/ticket"
	"github.com/theroymind/go-tix/internal/tix/backend"
	"github.com/theroymind/railyard-cli"
)

// parseType returns the ticket type --type names, or "" when the option is absent. The value is a ticket type, or
// one of the project's own names for one when the backend has them, such as "phase" for an epic in a vault.
func parseType(parsed cli.Arguments, store backend.Backend) (ticket.Type, error) {
	value, exists := parsed.String(optionType)
	if !exists {
		return "", nil
	}
	return resolveType(value, store)
}

// resolveType returns the ticket type value names, as parseType does, and a usage error for any other value. The
// error lists the types, paired with the project's names when it has them: `default|project: epic|phase, story, task|task`.
func resolveType(value string, store backend.Backend) (ticket.Type, error) {
	if ticket.Type(value).IsValid() {
		return ticket.Type(value), nil
	}
	names := typeNames(store)
	for _, name := range names {
		if name.Name == value {
			return name.Type, nil
		}
	}
	return "", cli.NewUsageError(fmt.Errorf("%s%s %q %w: expected %s", longOptionStart, optionType, value, ErrInvalidOption, typeChoices(names)))
}

// typeChoices lists the ticket types, each followed by the project's names for it: `default|project: epic|phase,
// story, task|task`. With no project names it's `epic, story, task`.
func typeChoices(names []backend.TypeName) string {
	listed := make([]string, len(ticket.Types))
	for index, ticketType := range ticket.Types {
		listed[index] = string(ticketType)
		for _, name := range names {
			if name.Type == ticketType {
				listed[index] += typeNameSeparator + name.Name
			}
		}
	}
	choices := strings.Join(listed, typeNamesHintSeparator)
	if len(names) == 0 {
		return choices
	}
	return typeNamesLegend + choices
}

// typeNames returns the project's own names for the ticket types, or none when the backend has no such names.
func typeNames(store backend.Backend) []backend.TypeName {
	namer, hasNames := store.(backend.TypeNamer)
	if !hasNames {
		return nil
	}
	return namer.TypeNames()
}
