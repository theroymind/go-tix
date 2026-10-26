// Package graph provides tix's rules across tickets, such as which tickets are ready to pick up next.
package graph

import "github.com/theroymind/railyard/internal/contracts/ticket"

// Ready returns, sorted, the open tickets whose dependencies are all resolved and whose children are all resolved.
// A dependency on a ticket that doesn't exist is unsatisfied, so a dangling ID never makes a ticket ready.
func Ready(tickets []ticket.Ticket) []ticket.Ticket {
	statuses := make(map[string]ticket.Status, len(tickets))
	hasUnresolvedChild := map[string]bool{}
	for _, candidate := range tickets {
		statuses[candidate.ID] = candidate.Status
		if candidate.Parent != "" && !candidate.Status.IsResolved() {
			hasUnresolvedChild[candidate.Parent] = true
		}
	}
	ready := []ticket.Ticket{}
	for _, candidate := range tickets {
		if candidate.Status != ticket.StatusOpen || hasUnresolvedChild[candidate.ID] {
			continue
		}
		if isSatisfied(candidate.Dependencies, statuses) {
			ready = append(ready, candidate)
		}
	}
	ticket.Sort(ready)
	return ready
}

func isSatisfied(dependencies []string, statuses map[string]ticket.Status) bool {
	for _, dependency := range dependencies {
		status, exists := statuses[dependency]
		if !exists || !status.IsResolved() {
			return false
		}
	}
	return true
}
