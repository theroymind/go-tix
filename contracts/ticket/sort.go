package ticket

import (
	"cmp"
	"slices"
	"time"
)

// Sort orders tickets by priority, then created_at, then ID, the order every tix listing uses.
// An absent priority sorts as PriorityDefault, and an absent created_at sorts before any present one.
func Sort(tickets []Ticket) {
	slices.SortFunc(tickets, func(left, right Ticket) int {
		return cmp.Or(
			cmp.Compare(left.PriorityOrDefault(), right.PriorityOrDefault()),
			createdAt(left).Compare(createdAt(right)),
			cmp.Compare(left.ID, right.ID),
		)
	})
}

// createdAt is the zero time for an absent created_at, and for an invalid one, which Validate rejects before tickets are listed.
func createdAt(ticket Ticket) time.Time {
	moment, _ := ticket.CreatedAt.Time()
	return moment
}
