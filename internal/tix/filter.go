package tix

import (
	"slices"

	"github.com/theroymind/railyard/internal/contracts/ticket"
)

// filter holds ls's options; an empty field matches every ticket.
type filter struct {
	status           ticket.Status
	ticketType       ticket.Type
	tag              string
	parent           string
	includesResolved bool
}

func (filter filter) matches(candidate ticket.Ticket) bool {
	if filter.status != "" && candidate.Status != filter.status {
		return false
	}
	if filter.status == "" && !filter.includesResolved && candidate.Status.IsResolved() {
		return false
	}
	if filter.ticketType != "" && candidate.Type != filter.ticketType {
		return false
	}
	if filter.tag != "" && !slices.Contains(candidate.Tags, filter.tag) {
		return false
	}
	return filter.parent == "" || candidate.Parent == filter.parent
}

func (filter filter) apply(tickets []ticket.Ticket) []ticket.Ticket {
	matching := []ticket.Ticket{}
	for _, candidate := range tickets {
		if filter.matches(candidate) {
			matching = append(matching, candidate)
		}
	}
	ticket.Sort(matching)
	return matching
}
