package graph

import (
	"cmp"
	"slices"
)

// Reached is a note a walk reached and the fewest hops it took to reach it.
type Reached struct {
	ID       string
	Distance int
}

// Walk visits notes breadth first from start, following neighbours, and returns each note it reaches within depth
// hops once, at its shortest distance, ordered by distance and then by id. The start is reached at distance 0, and
// a cycle ends where it meets a note already reached.
func Walk(start string, depth int, neighbours func(id string) []string) []Reached {
	distances := map[string]int{start: 0}
	reached := []Reached{{ID: start, Distance: 0}}
	frontier := []string{start}
	for distance := 1; distance <= depth && len(frontier) > 0; distance++ {
		var next []string
		for _, current := range frontier {
			for _, neighbour := range neighbours(current) {
				if _, isReached := distances[neighbour]; isReached {
					continue
				}
				distances[neighbour] = distance
				reached = append(reached, Reached{ID: neighbour, Distance: distance})
				next = append(next, neighbour)
			}
		}
		frontier = next
	}
	slices.SortStableFunc(reached, compareReached)
	return reached
}

func compareReached(first, second Reached) int {
	if byDistance := cmp.Compare(first.Distance, second.Distance); byDistance != 0 {
		return byDistance
	}
	return cmp.Compare(first.ID, second.ID)
}
