package graph

import (
	"slices"
	"testing"

	"github.com/theroymind/go-tix/contracts/ticket"
	"github.com/theroymind/go-tix/contracts/timestamp"
)

const createdAt timestamp.Timestamp = "2026-10-25T09:00:00Z"

func testTicket(id string, status ticket.Status) ticket.Ticket {
	return ticket.Ticket{ID: id, Status: status, Priority: new(ticket.PriorityDefault), CreatedAt: createdAt}
}

func withDependencies(value ticket.Ticket, dependencies ...string) ticket.Ticket {
	value.Dependencies = dependencies
	return value
}

func withParent(value ticket.Ticket, parent string) ticket.Ticket {
	value.Parent = parent
	return value
}

func withPriority(value ticket.Ticket, priority ticket.Priority) ticket.Ticket {
	value.Priority = &priority
	return value
}

func TestReady(t *testing.T) {
	tests := []struct {
		name     string
		input    []ticket.Ticket
		expected []string
	}{
		{"No tickets means none ready", nil, []string{}},
		{
			"Only open tickets are ready",
			[]ticket.Ticket{
				testTicket("t-0001", ticket.StatusOpen),
				testTicket("t-0002", ticket.StatusDraft),
				testTicket("t-0003", ticket.StatusInProgress),
				testTicket("t-0004", ticket.StatusCompleted),
				testTicket("t-0005", ticket.StatusCanceled),
			},
			[]string{"t-0001"},
		},
		{
			"An unresolved dependency blocks",
			[]ticket.Ticket{
				testTicket("t-000a", ticket.StatusOpen),
				withDependencies(testTicket("t-000b", ticket.StatusOpen), "t-000a"),
			},
			[]string{"t-000a"},
		},
		{
			"A completed dependency is satisfied",
			[]ticket.Ticket{
				testTicket("t-000a", ticket.StatusCompleted),
				withDependencies(testTicket("t-000b", ticket.StatusOpen), "t-000a"),
			},
			[]string{"t-000b"},
		},
		{
			"A canceled dependency is satisfied",
			[]ticket.Ticket{
				testTicket("t-000a", ticket.StatusCanceled),
				withDependencies(testTicket("t-000b", ticket.StatusOpen), "t-000a"),
			},
			[]string{"t-000b"},
		},
		{
			"A dangling dependency blocks",
			[]ticket.Ticket{withDependencies(testTicket("t-000b", ticket.StatusOpen), "t-zzzz")},
			[]string{},
		},
		{
			"A parent with an unresolved child waits for it",
			[]ticket.Ticket{
				testTicket("t-000e", ticket.StatusOpen),
				withParent(testTicket("t-000s", ticket.StatusOpen), "t-000e"),
			},
			[]string{"t-000s"},
		},
		{
			"A parent whose children are resolved is ready",
			[]ticket.Ticket{
				testTicket("t-000e", ticket.StatusOpen),
				withParent(testTicket("t-000s", ticket.StatusCompleted), "t-000e"),
				withParent(testTicket("t-000t", ticket.StatusCanceled), "t-000e"),
			},
			[]string{"t-000e"},
		},
		{
			"Ready tickets come back sorted",
			[]ticket.Ticket{
				withPriority(testTicket("t-0001", ticket.StatusOpen), ticket.PriorityLowest),
				withPriority(testTicket("t-0002", ticket.StatusOpen), ticket.PriorityHighest),
			},
			[]string{"t-0002", "t-0001"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ready := Ready(tt.input)

			got := make([]string, len(ready))
			for index, value := range ready {
				got[index] = value.ID
			}
			if !slices.Equal(got, tt.expected) {
				t.Errorf("Ready() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestDependencyPath(t *testing.T) {
	open := ticket.StatusOpen
	chain := []ticket.Ticket{
		withDependencies(testTicket("a", open), "b"),
		withDependencies(testTicket("b", open), "c"),
		testTicket("c", open),
	}
	diamond := []ticket.Ticket{
		withDependencies(testTicket("a", open), "long", "short"),
		withDependencies(testTicket("long", open), "middle"),
		withDependencies(testTicket("middle", open), "end"),
		withDependencies(testTicket("short", open), "end"),
		testTicket("end", open),
	}
	looped := []ticket.Ticket{
		withDependencies(testTicket("a", open), "b"),
		withDependencies(testTicket("b", open), "a"),
		testTicket("c", open),
	}
	tests := []struct {
		name     string
		tickets  []ticket.Ticket
		from     string
		to       string
		expected []string
	}{
		{"A ticket reaches itself by a path of one", chain, "a", "a", []string{"a"}},
		{"A direct dependency is a path of two", chain, "a", "b", []string{"a", "b"}},
		{"Dependencies are followed through", chain, "a", "c", []string{"a", "b", "c"}},
		{"Dependencies aren't followed backwards", chain, "c", "a", nil},
		{"The shortest path wins", diamond, "a", "end", []string{"a", "short", "end"}},
		{"A dangling dependency leads nowhere", []ticket.Ticket{withDependencies(testTicket("a", open), "gone")}, "a", "b", nil},
		{"An existing cycle doesn't stop the search ending", looped, "a", "c", nil},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := DependencyPath(tt.tickets, tt.from, tt.to); !slices.Equal(got, tt.expected) {
				t.Errorf("DependencyPath(%q, %q) = %v, want %v", tt.from, tt.to, got, tt.expected)
			}
		})
	}
}
