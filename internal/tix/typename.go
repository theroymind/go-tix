package tix

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/theroymind/railyard/internal/cli"
	"github.com/theroymind/railyard/internal/contracts/ticket"
	"github.com/theroymind/railyard/internal/contracts/validation"
	"github.com/theroymind/railyard/internal/tix/backend"
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

// resolveType returns the ticket type value names, as parseType does, and a usage error for any other value.
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
	return "", cli.NewUsageError(fmt.Errorf("%s%s %q %w: %s%s", longOptionStart, optionType, value, ErrInvalidOption,
		validation.InvalidOption(ticket.Types), typeNamesHint(names)))
}

// typeNames returns the project's own names for the ticket types, or none when the backend has no such names.
func typeNames(store backend.Backend) []backend.TypeName {
	namer, hasNames := store.(backend.TypeNamer)
	if !hasNames {
		return nil
	}
	return namer.TypeNames()
}

// typeNamesHint is the part of an invalid --type error that lists the project's own names, such as
// ` or one of this project's "phase" (epic), "task" (task)`. It's empty when the project has none.
func typeNamesHint(names []backend.TypeName) string {
	if len(names) == 0 {
		return ""
	}
	listed := make([]string, len(names))
	for index, name := range names {
		listed[index] = fmt.Sprintf("%s (%s)", strconv.Quote(name.Name), name.Type)
	}
	return typeNamesHintStart + strings.Join(listed, typeNamesHintSeparator)
}
