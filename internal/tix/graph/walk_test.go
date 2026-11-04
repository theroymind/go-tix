package graph

import (
	"reflect"
	"testing"
)

const testRelation = "link"

func TestWalk(t *testing.T) {
	const defaultDepth = 2
	chain := map[string][]string{"a": {"b"}, "b": {"c"}, "c": {"d"}}
	tests := []struct {
		name     string
		edges    map[string][]string
		start    string
		depth    int
		expected []Reached
	}{
		{"The start alone is reached at distance 0", nil, "a", defaultDepth, []Reached{{"a", 0, StartRelation}}},
		{"Depth 0 reaches only the start", chain, "a", 0, []Reached{{"a", 0, StartRelation}}},
		{"The walk stops at the depth given", chain, "a", defaultDepth,
			[]Reached{{"a", 0, StartRelation}, {"b", 1, testRelation}, {"c", 2, testRelation}}},
		{"A deeper walk reaches further", chain, "a", defaultDepth + 1,
			[]Reached{{"a", 0, StartRelation}, {"b", 1, testRelation}, {"c", 2, testRelation}, {"d", 3, testRelation}}},
		{"Two paths to one note keep the shorter distance",
			map[string][]string{"a": {"b", "c"}, "b": {"c"}, "c": {"d"}}, "a", defaultDepth,
			[]Reached{{"a", 0, StartRelation}, {"b", 1, testRelation}, {"c", 1, testRelation}, {"d", 2, testRelation}}},
		{"The longer path found first still gives way to the shorter",
			map[string][]string{"a": {"b", "d"}, "b": {"c"}, "c": {"d"}}, "a", defaultDepth + 1,
			[]Reached{{"a", 0, StartRelation}, {"b", 1, testRelation}, {"d", 1, testRelation}, {"c", 2, testRelation}}},
		{"A cycle between two notes ends the walk", map[string][]string{"a": {"b"}, "b": {"a"}}, "a", defaultDepth * 10,
			[]Reached{{"a", 0, StartRelation}, {"b", 1, testRelation}}},
		{"A note linking to itself is listed once", map[string][]string{"a": {"a", "b"}}, "a", defaultDepth,
			[]Reached{{"a", 0, StartRelation}, {"b", 1, testRelation}}},
		{"Notes at one distance are ordered by id", map[string][]string{"a": {"z", "m", "b"}}, "a", defaultDepth,
			[]Reached{{"a", 0, StartRelation}, {"b", 1, testRelation}, {"m", 1, testRelation}, {"z", 1, testRelation}}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			neighbours := func(id string) []Neighbour {
				var named []Neighbour
				for _, neighbour := range tt.edges[id] {
					named = append(named, Neighbour{ID: neighbour, Relation: testRelation})
				}
				return named
			}

			reached := Walk(tt.start, tt.depth, neighbours)

			if !reflect.DeepEqual(reached, tt.expected) {
				t.Errorf("Walk() = %v, want %v", reached, tt.expected)
			}
		})
	}
}

func TestWalkKeepsTheFirstRelationThatReachesANote(t *testing.T) {
	neighbours := func(id string) []Neighbour {
		if id != "a" {
			return nil
		}
		return []Neighbour{{"b", "parent"}, {"b", "wikilink"}, {"c", "backlink"}}
	}

	reached := Walk("a", 1, neighbours)

	expected := []Reached{{"a", 0, StartRelation}, {"b", 1, "parent"}, {"c", 1, "backlink"}}
	if !reflect.DeepEqual(reached, expected) {
		t.Errorf("Walk() = %v, want %v", reached, expected)
	}
}
