package tix

import (
	"errors"
	"fmt"
	"io"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/theroymind/railyard/internal/cli"
	"github.com/theroymind/railyard/internal/contracts/ticket"
	"github.com/theroymind/railyard/internal/contracts/timestamp"
	"github.com/theroymind/railyard/internal/tix/backend"
	"github.com/theroymind/railyard/internal/tix/graph"
)

var dependencySpecification = cli.Specification{AllowsPositionals: true}

// runDep adds each dependency the ticket doesn't have yet, refusing one that would close a cycle. Every dependency
// must resolve to a ticket, and nothing is written unless all of them do.
func runDep(arguments []string, _ io.Reader, _ io.Writer) error {
	store, dependent, references, err := openDependent(arguments)
	if err != nil {
		return err
	}
	tickets, err := store.List()
	if err != nil {
		return err
	}
	dependencies := slices.Clone(dependent.Dependencies)
	for _, reference := range references {
		id, err := store.Resolve(reference)
		if err != nil {
			return err
		}
		if slices.Contains(dependencies, id) {
			continue
		}
		pending := withDependencies(tickets, dependent.ID, dependencies)
		if path := graph.DependencyPath(pending, id, dependent.ID); path != nil {
			return dependencyCycle(append([]string{dependent.ID}, path...))
		}
		dependencies = append(dependencies, id)
	}
	if len(dependencies) == len(dependent.Dependencies) {
		return nil
	}
	return setDependencies(store, dependent.ID, dependencies)
}

// runUndep removes each dependency named, by the exact id the ticket holds or by a reference to a ticket. A
// dependency on a ticket that no longer exists is removed by its exact id.
func runUndep(arguments []string, _ io.Reader, _ io.Writer) error {
	store, dependent, references, err := openDependent(arguments)
	if err != nil {
		return err
	}
	dependencies := slices.Clone(dependent.Dependencies)
	for _, reference := range references {
		id, err := heldDependency(store, dependent, reference)
		if err != nil {
			return err
		}
		dependencies = slices.DeleteFunc(dependencies, func(held string) bool { return held == id })
	}
	return setDependencies(store, dependent.ID, dependencies)
}

// openDependent parses REF DEPENDENCY_REF... and reads the ticket REF names.
func openDependent(arguments []string) (backend.Backend, ticket.Ticket, []string, error) {
	parsed, err := cli.Parse(arguments, dependencySpecification)
	if err != nil {
		return nil, ticket.Ticket{}, nil, err
	}
	if len(parsed.Positionals) < 2 {
		return nil, ticket.Ticket{}, nil, cli.NewUsageError(ErrExpectedDependencies)
	}
	store, err := openBackend()
	if err != nil {
		return nil, ticket.Ticket{}, nil, err
	}
	id, err := store.Resolve(parsed.Positionals[0])
	if err != nil {
		return nil, ticket.Ticket{}, nil, err
	}
	dependent, err := store.Read(id)
	if err != nil {
		return nil, ticket.Ticket{}, nil, err
	}
	return store, dependent, parsed.Positionals[1:], nil
}

// heldDependency returns the dependency of dependent that reference names: an id it holds exactly, else the ticket
// reference resolves to, which must be one it holds.
func heldDependency(store backend.Backend, dependent ticket.Ticket, reference string) (string, error) {
	if slices.Contains(dependent.Dependencies, reference) {
		return reference, nil
	}
	id, err := store.Resolve(reference)
	if errors.Is(err, backend.ErrNoMatch) {
		return "", notDependency(dependent.ID, reference)
	}
	if err != nil {
		return "", err
	}
	if !slices.Contains(dependent.Dependencies, id) {
		return "", notDependency(dependent.ID, id)
	}
	return id, nil
}

// withDependencies returns a copy of tickets in which the ticket with id has dependencies instead of its own.
func withDependencies(tickets []ticket.Ticket, id string, dependencies []string) []ticket.Ticket {
	replaced := slices.Clone(tickets)
	for index := range replaced {
		if replaced[index].ID == id {
			replaced[index].Dependencies = dependencies
		}
	}
	return replaced
}

func setDependencies(store backend.Backend, id string, dependencies []string) error {
	changes := []backend.Change{
		backend.SetField{Field: ticket.FieldDependencies, Value: dependencies},
		backend.SetField{Field: ticket.FieldUpdatedAt, Value: timestamp.Format(time.Now())},
	}
	return store.Apply(id, changes)
}

func dependencyCycle(path []string) error {
	return fmt.Errorf("%w: %s", ErrDependencyCycle, strings.Join(path, dependencyPathSeparator))
}

func notDependency(id, reference string) error {
	return fmt.Errorf("%s %w %s", id, ErrNotDependency, strconv.Quote(reference))
}
