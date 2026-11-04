// Package graph provides tix's rules across tickets, such as which tickets are ready to pick up next.
package graph

import (
	"slices"

	"github.com/theroymind/go-tix/contracts/ticket"
)

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

// DependencyPath returns the shortest chain of ids from one ticket to another following dependencies, starting with
// from and ending with to, or nil when to can't be reached. A ticket reaches itself by a path of one.
func DependencyPath(tickets []ticket.Ticket, from, to string) []string {
	dependencies := make(map[string][]string, len(tickets))
	for _, candidate := range tickets {
		dependencies[candidate.ID] = candidate.Dependencies
	}
	previous := map[string]string{from: ""}
	queue := []string{from}
	for len(queue) > 0 {
		current := queue[0]
		queue = queue[1:]
		if current == to {
			return pathTo(previous, from, to)
		}
		for _, next := range dependencies[current] {
			if _, isSeen := previous[next]; isSeen {
				continue
			}
			previous[next] = current
			queue = append(queue, next)
		}
	}
	return nil
}

// pathTo walks previous back from to, which was reached from from, and returns the path in order.
func pathTo(previous map[string]string, from, to string) []string {
	path := []string{to}
	for current := to; current != from; {
		current = previous[current]
		path = append(path, current)
	}
	slices.Reverse(path)
	return path
}
