package vaultstore_test

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/theroymind/railyard/internal/contracts/ticket"
	"github.com/theroymind/railyard/internal/tix/backend"
	"github.com/theroymind/railyard/internal/tix/backend/vaultstore"
)

// copyFixture opens a copy of the fixture vault, so that Apply never writes to testdata.
func copyFixture(t *testing.T) (backend.Backend, string) {
	t.Helper()
	root := t.TempDir()
	if err := os.CopyFS(root, os.DirFS(fixture)); err != nil {
		t.Fatal(err)
	}
	backend.Register(backend.KindVault, vaultstore.Open)
	store, err := backend.Open(root)
	if err != nil {
		t.Fatalf("Open() error = %v", err)
	}
	return store, root
}

func TestApply(t *testing.T) {
	const note = "docs/tasks/golem-guards.md"
	updated := backend.SetField{Field: ticket.FieldUpdatedAt, Value: "2026-11-02T10:00:00Z"}
	tests := []struct {
		name          string
		changes       []backend.Change
		replace       [2]string
		expectedError error
		mentions      []string
	}{
		{"A ticket status maps back to its note status, and a timestamp the note lacks is skipped",
			[]backend.Change{backend.SetField{Field: ticket.FieldStatus, Value: ticket.StatusCompleted}, updated},
			[2]string{"status: todo\n", "status: done\n"}, nil, nil},
		{"A note status is taken as written",
			[]backend.Change{backend.SetField{Field: ticket.FieldStatus, Value: "done"}},
			[2]string{"status: todo\n", "status: done\n"}, nil, nil},
		{"Any other field is written as a property",
			[]backend.Change{backend.SetField{Field: "scope", Value: "large"}},
			[2]string{"scope: small\n", "scope: large\n"}, nil, nil},
		{"A new property is appended to the frontmatter",
			[]backend.Change{backend.SetField{Field: "owner", Value: "eric"}},
			[2]string{"depends_on: [weighted-step-choice]\n", "depends_on: [weighted-step-choice]\nowner: eric\n"}, nil, nil},
		{"A status the profile doesn't map names the note and the allowed statuses",
			[]backend.Change{backend.SetField{Field: ticket.FieldStatus, Value: "nonsense"}},
			[2]string{}, vaultstore.ErrUnknownStatus, []string{"golem-guards.md", `"nonsense"`, `"done", "todo"`}},
		{"A failing change leaves the note as it was",
			[]backend.Change{backend.SetField{Field: "scope", Value: "large"}, backend.SetField{Field: ticket.FieldStatus, Value: "nonsense"}},
			[2]string{}, vaultstore.ErrUnknownStatus, nil},
		{"A ticket field with no property to hold it is unsupported",
			[]backend.Change{backend.SetField{Field: ticket.FieldTitle, Value: "Golems"}},
			[2]string{}, backend.ErrUnsupported, []string{ticket.FieldTitle}},
		{"A link field is unsupported",
			[]backend.Change{backend.SetField{Field: ticket.FieldDependencies, Value: []string{"mana"}}},
			[2]string{}, backend.ErrUnsupported, []string{ticket.FieldDependencies}},
		{"A property the profile maps to a link field is unsupported",
			[]backend.Change{backend.SetField{Field: "depends_on", Value: "mana"}},
			[2]string{}, backend.ErrUnsupported, []string{"depends_on"}},
		{"Criterion changes are unsupported",
			[]backend.Change{backend.CheckCriterion{Index: 1}},
			[2]string{}, backend.ErrUnsupported, nil},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			store, root := copyFixture(t)
			path := filepath.Join(root, note)
			original, err := os.ReadFile(path)
			if err != nil {
				t.Fatal(err)
			}

			err = store.Apply("golem-guards", tt.changes)

			if !errors.Is(err, tt.expectedError) {
				t.Fatalf("Apply() error = %v, want %v", err, tt.expectedError)
			}
			for _, mention := range tt.mentions {
				if !strings.Contains(err.Error(), mention) {
					t.Errorf("Apply() error = %q, want it to mention %s", err, mention)
				}
			}
			expected := string(original)
			if tt.expectedError == nil {
				if !strings.Contains(expected, tt.replace[0]) {
					t.Fatalf("fixture lacks %q", tt.replace[0])
				}
				expected = strings.Replace(expected, tt.replace[0], tt.replace[1], 1)
			}
			got, err := os.ReadFile(path)
			if err != nil {
				t.Fatal(err)
			}
			if string(got) != expected {
				t.Errorf("note after Apply() =\n%s\nwant\n%s", got, expected)
			}
		})
	}
}

func TestApplyWritesATimestampTheNoteHolds(t *testing.T) {
	store, root := copyFixture(t)
	path := filepath.Join(root, "docs/tasks/golem-guards.md")
	original, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	write(t, path, strings.Replace(string(original), "scope: small\n", "scope: small\nupdated_at: 2026-11-01\n", 1))

	if err := store.Apply("golem-guards", []backend.Change{backend.SetField{Field: ticket.FieldUpdatedAt, Value: "2026-11-02"}}); err != nil {
		t.Fatalf("Apply() error = %v", err)
	}

	got, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if expected := strings.Replace(string(original), "scope: small\n", "scope: small\nupdated_at: 2026-11-02\n", 1); string(got) != expected {
		t.Errorf("note after Apply() =\n%s\nwant\n%s", got, expected)
	}
}

func TestApplyRepairsANoteWithAnUnmappedStatus(t *testing.T) {
	store, root := copyFixture(t)
	path := filepath.Join(root, "docs/tasks/golem-guards.md")
	original, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	write(t, path, strings.Replace(string(original), "status: todo\n", "status: tdo\n", 1))

	if err := store.Apply("golem-guards", []backend.Change{backend.SetField{Field: ticket.FieldStatus, Value: "todo"}}); err != nil {
		t.Fatalf("Apply() error = %v", err)
	}

	got, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != string(original) {
		t.Errorf("note after Apply() =\n%s\nwant\n%s", got, original)
	}
}
