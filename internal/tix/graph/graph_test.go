package graph

import (
	"slices"
	"testing"

	"github.com/theroymind/railyard/internal/contracts/ticket"
	"github.com/theroymind/railyard/internal/contracts/timestamp"
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
