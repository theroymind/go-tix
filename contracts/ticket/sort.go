package ticket

import (
	"cmp"
	"slices"
)

// Sort orders tickets by priority, then created_at, then ID, the order every tix listing uses.
func Sort(tickets []Ticket) {
	slices.SortFunc(tickets, func(left, right Ticket) int {
		return cmp.Or(
			cmp.Compare(left.Priority, right.Priority),
			left.CreatedAt.Time().Compare(right.CreatedAt.Time()),
			cmp.Compare(left.ID, right.ID),
		)
	})
}
