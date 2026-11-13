package tix

import (
	"slices"
	"testing"

	"github.com/theroymind/go-tix/contracts/ticket"
)

func TestNest(t *testing.T) {
	tests := []struct {
		name    string
		tickets []ticket.Ticket
		ids     []string
		depths  []int
	}{
		{
			"Each child follows its parent, a level deeper",
			[]ticket.Ticket{{ID: "epic"}, {ID: "other"}, {ID: "task", Parent: "epic"}, {ID: "sub", Parent: "task"}},
			[]string{"epic", "task", "sub", "other"},
			[]int{0, 1, 2, 0},
		},
		{
			"A ticket whose parent isn't listed is a root",
			[]ticket.Ticket{{ID: "task", Parent: "hidden"}, {ID: "sub", Parent: "task"}},
			[]string{"task", "sub"},
			[]int{0, 1},
		},
		{
			"A parent cycle is listed flat rather than dropped",
			[]ticket.Ticket{{ID: "a", Parent: "b"}, {ID: "b", Parent: "a"}, {ID: "self", Parent: "self"}},
			[]string{"self", "a", "b"},
			[]int{0, 0, 1},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ordered, depths := nest(tt.tickets)
			var ids []string
			var gotDepths []int
			for _, row := range ordered {
				ids = append(ids, row.ID)
				gotDepths = append(gotDepths, depths[row.ID])
			}
			if !slices.Equal(ids, tt.ids) || !slices.Equal(gotDepths, tt.depths) {
				t.Errorf("nest() = %v at depths %v, want %v at %v", ids, gotDepths, tt.ids, tt.depths)
			}
		})
	}
}
