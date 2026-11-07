package questions

import (
	"slices"
	"testing"

	"github.com/theroymind/go-tix/contracts/sketch"
)

func TestOpen(t *testing.T) {
	tests := []struct {
		name     string
		notes    []sketch.Note
		expected []Question
	}{
		{"No notes", nil, []Question{}},
		{"Answered questions are left out", []sketch.Note{{Path: "a.go.sketch.md", Questions: []sketch.Question{
			{Text: "how fast", Answer: "1 rad/s", Line: 1},
			{Text: "who owns it", Line: 3},
		}}}, []Question{{Path: "a.go.sketch.md", Line: 3, Text: "who owns it"}}},
		{"Every question answered", []sketch.Note{{Path: "a.go.sketch.md", Questions: []sketch.Question{
			{Text: "how fast", Answer: "1 rad/s", Line: 1},
		}}}, []Question{}},
		{"Ordered by note path, then line", []sketch.Note{
			{Path: "web/app.ts.sketch.md", Questions: []sketch.Question{{Text: "c", Line: 2}}},
			{Path: "a.go.sketch.md", Questions: []sketch.Question{{Text: "b", Line: 9}, {Text: "a", Line: 4}}},
		}, []Question{
			{Path: "a.go.sketch.md", Line: 4, Text: "a"},
			{Path: "a.go.sketch.md", Line: 9, Text: "b"},
			{Path: "web/app.ts.sketch.md", Line: 2, Text: "c"},
		}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if actual := Open(tt.notes); !slices.Equal(actual, tt.expected) {
				t.Errorf("Open() = %+v, want %+v", actual, tt.expected)
			}
		})
	}
}

func TestAll(t *testing.T) {
	tests := []struct {
		name     string
		notes    []sketch.Note
		expected []Question
	}{
		{"No notes", nil, []Question{}},
		{"Answered questions keep their answers, by note path and then line", []sketch.Note{
			{Path: "web/app.ts.sketch.md", Questions: []sketch.Question{{Text: "c", Line: 2}}},
			{Path: "a.go.sketch.md", Questions: []sketch.Question{
				{Text: "how fast", Answer: "1 rad/s\nthen ease", Line: 9},
				{Text: "who owns it", Line: 3},
			}},
		}, []Question{
			{Path: "a.go.sketch.md", Line: 3, Text: "who owns it"},
			{Path: "a.go.sketch.md", Line: 9, Text: "how fast", Answer: "1 rad/s\nthen ease"},
			{Path: "web/app.ts.sketch.md", Line: 2, Text: "c"},
		}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if actual := All(tt.notes); !slices.Equal(actual, tt.expected) {
				t.Errorf("All() = %+v, want %+v", actual, tt.expected)
			}
		})
	}
}
