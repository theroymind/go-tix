package ticket

import (
	"slices"
	"testing"

	"github.com/theroymind/railyard/internal/contracts/timestamp"
)

const (
	sortEarlier timestamp.Timestamp = "2026-10-25T09:00:00Z"
	sortLater   timestamp.Timestamp = "2026-10-25T06:00:00-04:00"
)

func TestSort(t *testing.T) {
	tests := []struct {
		name     string
		input    []Ticket
		expected []string
	}{
		{
			"Priority comes first",
			[]Ticket{
				{ID: "t-0001", Priority: new(PriorityLowest), CreatedAt: sortEarlier},
				{ID: "t-0002", Priority: new(PriorityHighest), CreatedAt: sortLater},
			},
			[]string{"t-0002", "t-0001"},
		},
		{
			"Created at breaks a priority tie, comparing instants across offsets",
			[]Ticket{
				{ID: "t-0001", Priority: new(PriorityDefault), CreatedAt: sortLater},
				{ID: "t-0002", Priority: new(PriorityDefault), CreatedAt: sortEarlier},
			},
			[]string{"t-0002", "t-0001"},
		},
		{
			"An absent priority sorts as the default",
			[]Ticket{
				{ID: "t-0001", Priority: new(PriorityLowest), CreatedAt: sortEarlier},
				{ID: "t-0002", CreatedAt: sortLater},
				{ID: "t-0003", Priority: new(PriorityHighest), CreatedAt: sortLater},
			},
			[]string{"t-0003", "t-0002", "t-0001"},
		},
		{
			"An absent created_at sorts before a present one",
			[]Ticket{
				{ID: "t-0001", Priority: new(PriorityDefault), CreatedAt: sortEarlier},
				{ID: "t-0002", Priority: new(PriorityDefault)},
			},
			[]string{"t-0002", "t-0001"},
		},
		{
			"ID breaks a full tie",
			[]Ticket{
				{ID: "t-0002", Priority: new(PriorityDefault), CreatedAt: sortEarlier},
				{ID: "t-0001", Priority: new(PriorityDefault), CreatedAt: sortEarlier},
			},
			[]string{"t-0001", "t-0002"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			Sort(tt.input)

			got := make([]string, len(tt.input))
			for index, sorted := range tt.input {
				got[index] = sorted.ID
			}
			if !slices.Equal(got, tt.expected) {
				t.Errorf("Sort() = %v, want %v", got, tt.expected)
			}
		})
	}
}
