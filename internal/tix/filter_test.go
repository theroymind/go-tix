package tix

import (
	"slices"
	"testing"

	"github.com/theroymind/go-tix/contracts/ticket"
	"github.com/theroymind/go-tix/contracts/timestamp"
)

const filterCreatedAt timestamp.Timestamp = "2026-10-25T09:00:00Z"

var filterTickets = []ticket.Ticket{
	{ID: "t-0001", Type: ticket.TypeEpic, Status: ticket.StatusOpen, Priority: new(ticket.PriorityLowest), CreatedAt: filterCreatedAt},
	{
		ID: "t-0002", Type: ticket.TypeStory, Status: ticket.StatusInProgress, Priority: new(ticket.PriorityDefault),
		Tags: []string{"engine"}, Parent: "t-0001", CreatedAt: filterCreatedAt,
	},
	{ID: "t-0003", Type: ticket.TypeTask, Status: ticket.StatusCompleted, Priority: new(ticket.PriorityHighest), CreatedAt: filterCreatedAt},
	{ID: "t-0004", Type: ticket.TypeTask, Status: ticket.StatusCanceled, Priority: new(ticket.PriorityHighest), Parent: "t-0001", CreatedAt: filterCreatedAt},
}

func TestFilterApply(t *testing.T) {
	tests := []struct {
		name     string
		filter   filter
		expected []string
	}{
		{"Resolved tickets are hidden by default, and the rest sorted", filter{}, []string{"t-0002", "t-0001"}},
		{"All includes resolved tickets", filter{includesResolved: true}, []string{"t-0003", "t-0004", "t-0002", "t-0001"}},
		{"A resolved status shows its tickets", filter{status: ticket.StatusCompleted}, []string{"t-0003"}},
		{"An unresolved status shows only its tickets", filter{status: ticket.StatusOpen}, []string{"t-0001"}},
		{"Type filters", filter{ticketType: ticket.TypeTask, includesResolved: true}, []string{"t-0003", "t-0004"}},
		{"Tag filters", filter{tag: "engine"}, []string{"t-0002"}},
		{"Parent shows direct children", filter{parent: "t-0001"}, []string{"t-0002"}},
		{"Parent with all includes resolved children", filter{parent: "t-0001", includesResolved: true}, []string{"t-0004", "t-0002"}},
		{"No match is an empty list", filter{tag: "missing"}, []string{}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			matching := tt.filter.apply(slices.Clone(filterTickets))

			got := make([]string, len(matching))
			for index, value := range matching {
				got[index] = value.ID
			}
			if !slices.Equal(got, tt.expected) {
				t.Errorf("apply() = %v, want %v", got, tt.expected)
			}
		})
	}
}
