package tix

import (
	"errors"
	"reflect"
	"testing"

	"github.com/theroymind/go-tix/internal/tix/backend"
)

func TestParseStaged(t *testing.T) {
	tests := []struct {
		name          string
		output        string
		expectedPaths []string
		expectedError error
	}{
		{"Nothing staged is no paths", "", nil, nil},
		{"Added and modified files keep their order",
			"A\x00docs/decisions/new.md\x00M\x00docs/tasks/mana.md\x00",
			[]string{"docs/decisions/new.md", "docs/tasks/mana.md"}, nil},
		{"A path with spaces stays whole", "A\x00docs/a note.md\x00", []string{"docs/a note.md"}, nil},
		{"A status the filter excludes is rejected", "D\x00docs/old.md\x00", nil, ErrInvalidStaged},
		{"A status with no path is rejected", "A\x00", nil, ErrInvalidStaged},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			paths, err := parseStaged([]byte(tt.output))

			if !errors.Is(err, tt.expectedError) {
				t.Fatalf("parseStaged() error = %v, expected %v", err, tt.expectedError)
			}
			if !reflect.DeepEqual(paths, tt.expectedPaths) {
				t.Errorf("parseStaged() = %q, expected %q", paths, tt.expectedPaths)
			}
		})
	}
}

func TestStagedNotes(t *testing.T) {
	lintable := []backend.LintNote{{ID: "mana", Path: "docs/tasks/mana.md"}, {ID: "new", Path: "docs/decisions/new.md"}}
	tests := []struct {
		name        string
		staged      []string
		expectedIDs []string
	}{
		{"Staged notes keep git's order", []string{"docs/decisions/new.md", "docs/tasks/mana.md"}, []string{"new", "mana"}},
		{"Files that aren't ticket or doc notes are left out",
			[]string{"sim/turn.odin", "docs/templates/task.md", "docs/tasks/mana.md"}, []string{"mana"}},
		{"No staged notes is none", []string{"README.md"}, nil},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ids := stagedNotes(tt.staged, lintable)

			if !reflect.DeepEqual(ids, tt.expectedIDs) {
				t.Errorf("stagedNotes() = %q, expected %q", ids, tt.expectedIDs)
			}
		})
	}
}
