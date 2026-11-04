package graph

import (
	"reflect"
	"testing"
)

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
		{"The start alone is reached at distance 0", nil, "a", defaultDepth, []Reached{{"a", 0}}},
		{"Depth 0 reaches only the start", chain, "a", 0, []Reached{{"a", 0}}},
		{"The walk stops at the depth given", chain, "a", defaultDepth, []Reached{{"a", 0}, {"b", 1}, {"c", 2}}},
		{"A deeper walk reaches further", chain, "a", defaultDepth + 1, []Reached{{"a", 0}, {"b", 1}, {"c", 2}, {"d", 3}}},
		{"Two paths to one note keep the shorter distance",
			map[string][]string{"a": {"b", "c"}, "b": {"c"}, "c": {"d"}}, "a", defaultDepth,
			[]Reached{{"a", 0}, {"b", 1}, {"c", 1}, {"d", 2}}},
		{"The longer path found first still gives way to the shorter",
			map[string][]string{"a": {"b", "d"}, "b": {"c"}, "c": {"d"}}, "a", defaultDepth + 1,
			[]Reached{{"a", 0}, {"b", 1}, {"d", 1}, {"c", 2}}},
		{"A cycle between two notes ends the walk", map[string][]string{"a": {"b"}, "b": {"a"}}, "a", defaultDepth * 10,
			[]Reached{{"a", 0}, {"b", 1}}},
		{"A note linking to itself is listed once", map[string][]string{"a": {"a", "b"}}, "a", defaultDepth,
			[]Reached{{"a", 0}, {"b", 1}}},
		{"Notes at one distance are ordered by id", map[string][]string{"a": {"z", "m", "b"}}, "a", defaultDepth,
			[]Reached{{"a", 0}, {"b", 1}, {"m", 1}, {"z", 1}}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			neighbours := func(id string) []string { return tt.edges[id] }

			reached := Walk(tt.start, tt.depth, neighbours)

			if !reflect.DeepEqual(reached, tt.expected) {
				t.Errorf("Walk() = %v, want %v", reached, tt.expected)
			}
		})
	}
}
