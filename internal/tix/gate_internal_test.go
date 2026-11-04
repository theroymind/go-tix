package tix

import (
	"errors"
	"io"
	"reflect"
	"strings"
	"testing"

	"github.com/theroymind/railyard/internal/tix/backend"
)

var errGateFailed = errors.New("gate failed")

func TestGatesRun(t *testing.T) {
	tests := []struct {
		name          string
		notes         []gateNote
		lintError     error
		judgeError    error
		expectedCalls []string
		expectedError error
	}{
		{"Lint runs on every note, then the judge on the new ones",
			[]gateNote{{ID: "mana", IsNew: false}, {ID: "wait-counter", IsNew: true}}, nil, nil,
			[]string{"lint mana,wait-counter", "judge wait-counter"}, nil},
		{"A lint failure stops before the judge",
			[]gateNote{{ID: "wait-counter", IsNew: true}}, errGateFailed, nil,
			[]string{"lint wait-counter"}, errGateFailed},
		{"With no new notes the judge isn't asked",
			[]gateNote{{ID: "mana", IsNew: false}}, nil, nil,
			[]string{"lint mana"}, nil},
		{"A judge refusal fails the gate",
			[]gateNote{{ID: "wait-counter", IsNew: true}}, nil, errGateFailed,
			[]string{"lint wait-counter", "judge wait-counter"}, errGateFailed},
		{"No staged notes run no gate", nil, nil, nil, nil, nil},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var calls []string
			checks := gates{
				lint: func(ids []string, _ io.Writer) error {
					calls = append(calls, "lint "+strings.Join(ids, ","))
					return tt.lintError
				},
				judge: func(ids []string, _ io.Writer) error {
					calls = append(calls, "judge "+strings.Join(ids, ","))
					return tt.judgeError
				},
			}

			err := checks.run(tt.notes, io.Discard)

			if !errors.Is(err, tt.expectedError) {
				t.Errorf("run() error = %v, expected %v", err, tt.expectedError)
			}
			if !reflect.DeepEqual(calls, tt.expectedCalls) {
				t.Errorf("calls = %q, expected %q", calls, tt.expectedCalls)
			}
		})
	}
}

func TestParseStaged(t *testing.T) {
	tests := []struct {
		name          string
		output        string
		expectedFiles []stagedFile
		expectedError error
	}{
		{"Nothing staged is no files", "", nil, nil},
		{"Added and modified files keep their order",
			"A\x00docs/decisions/new.md\x00M\x00docs/tasks/mana.md\x00",
			[]stagedFile{{Path: "docs/decisions/new.md", IsAdded: true}, {Path: "docs/tasks/mana.md", IsAdded: false}}, nil},
		{"A path with spaces stays whole", "A\x00docs/a note.md\x00", []stagedFile{{Path: "docs/a note.md", IsAdded: true}}, nil},
		{"A status the filter excludes is rejected", "D\x00docs/old.md\x00", nil, ErrInvalidStaged},
		{"A status with no path is rejected", "A\x00", nil, ErrInvalidStaged},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			files, err := parseStaged([]byte(tt.output))

			if !errors.Is(err, tt.expectedError) {
				t.Fatalf("parseStaged() error = %v, expected %v", err, tt.expectedError)
			}
			if !reflect.DeepEqual(files, tt.expectedFiles) {
				t.Errorf("parseStaged() = %+v, expected %+v", files, tt.expectedFiles)
			}
		})
	}
}

func TestStagedNotes(t *testing.T) {
	lintable := []backend.LintNote{{ID: "mana", Path: "docs/tasks/mana.md"}, {ID: "new", Path: "docs/decisions/new.md"}}
	tests := []struct {
		name          string
		staged        []stagedFile
		expectedNotes []gateNote
	}{
		{"Staged notes keep git's order and whether they are new",
			[]stagedFile{{Path: "docs/decisions/new.md", IsAdded: true}, {Path: "docs/tasks/mana.md"}},
			[]gateNote{{ID: "new", IsNew: true}, {ID: "mana", IsNew: false}}},
		{"Files that aren't ticket or doc notes are left out",
			[]stagedFile{{Path: "sim/turn.odin", IsAdded: true}, {Path: "docs/templates/task.md"}, {Path: "docs/tasks/mana.md"}},
			[]gateNote{{ID: "mana", IsNew: false}}},
		{"No staged notes is none", []stagedFile{{Path: "README.md"}}, nil},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			notes := stagedNotes(tt.staged, lintable)

			if !reflect.DeepEqual(notes, tt.expectedNotes) {
				t.Errorf("stagedNotes() = %+v, expected %+v", notes, tt.expectedNotes)
			}
		})
	}
}
