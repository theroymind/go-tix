package graph

import (
	"cmp"
	"slices"
)

// Neighbour is a note one hop from another, and the relation that hop follows.
type Neighbour struct {
	ID       string
	Relation string
}

// Reached is a note a walk reached, the fewest hops it took to reach it, and the relation of the first hop that
// reached it at that distance.
type Reached struct {
	ID       string
	Distance int
	Relation string
}

// StartRelation is the relation of the note a walk starts at.
const StartRelation = "start"

// Walk visits notes breadth first from start, following neighbours, and returns each note it reaches within depth
// hops once, at its shortest distance, ordered by distance and then by id. The start is reached at distance 0, a
// cycle ends where it meets a note already reached, and a note reached twice at one distance keeps the relation of
// the first neighbour that named it, so a note lists its strongest relations first.
func Walk(start string, depth int, neighbours func(id string) []Neighbour) []Reached {
	distances := map[string]int{start: 0}
	reached := []Reached{{ID: start, Distance: 0, Relation: StartRelation}}
	frontier := []string{start}
	for distance := 1; distance <= depth && len(frontier) > 0; distance++ {
		var next []string
		for _, current := range frontier {
			for _, neighbour := range neighbours(current) {
				if _, isReached := distances[neighbour.ID]; isReached {
					continue
				}
				distances[neighbour.ID] = distance
				reached = append(reached, Reached{ID: neighbour.ID, Distance: distance, Relation: neighbour.Relation})
				next = append(next, neighbour.ID)
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
