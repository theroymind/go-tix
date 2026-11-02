package vaultstore_test

import (
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/theroymind/railyard/internal/contracts/ticket"
	"github.com/theroymind/railyard/internal/tix/backend"
	"github.com/theroymind/railyard/internal/tix/backend/vaultstore"
	"github.com/theroymind/railyard/internal/tix/graph"
)

// fixture is a subset of odin-tbc's docs/, with some tasks set back to todo so that ready has something to pick.
var fixture = filepath.Join("testdata", "odin-tbc")

func openFixture(t *testing.T) backend.Backend {
	t.Helper()
	backend.Register(backend.KindVault, vaultstore.Open)
	store, err := backend.Open(fixture)
	if err != nil {
		t.Fatalf("Open() error = %v", err)
	}
	return store
}

func TestOpenReadsTheProfile(t *testing.T) {
	store, isVault := openFixture(t).(vaultstore.Store)
	if !isVault {
		t.Fatalf("Open() = %T, want vaultstore.Store", store)
	}
	absolute, err := filepath.Abs(filepath.Join(fixture, "docs"))
	if err != nil {
		t.Fatal(err)
	}
	if store.Root != absolute || store.Profile.TypeProperty != "type" || store.Profile.CriteriaHeading != "Acceptance criteria" {
		t.Errorf("Open() = %+v, want root %s with the fixture's profile", store, absolute)
	}
}

func TestList(t *testing.T) {
	tickets, err := openFixture(t).List()
	if err != nil {
		t.Fatalf("List() error = %v", err)
	}
	type row struct {
		id, title    string
		ticketType   ticket.Type
		status       ticket.Status
		parent       string
		dependencies []string
	}
	// Decisions, ideas, specs, the home note and the templates are absent.
	expected := []row{
		{"enemy-puzzle", "Enemy puzzle", ticket.TypeEpic, ticket.StatusInProgress, "", nil},
		{"first-fight", "First fight", ticket.TypeEpic, ticket.StatusInProgress, "", nil},
		{"goal-scoring", "Goal scoring", ticket.TypeTask, ticket.StatusCompleted, "enemy-puzzle", []string{"sim-queries"}},
		{"golem-guards", "Golem guards", ticket.TypeTask, ticket.StatusOpen, "enemy-puzzle", []string{"weighted-step-choice"}},
		{"harpy-hastes", "Harpy hastes", ticket.TypeTask, ticket.StatusOpen, "enemy-puzzle", []string{"weighted-step-choice"}},
		{"mana", "Mana", ticket.TypeTask, ticket.StatusCompleted, "first-fight", []string{"sim-actions"}},
		{"sim-queries", "Sim queries", ticket.TypeTask, ticket.StatusCompleted, "enemy-puzzle", nil},
		{"weighted-step-choice", "Weighted step choice", ticket.TypeTask, ticket.StatusOpen, "enemy-puzzle", []string{"goal-scoring"}},
		{"wraith-casts", "Wraith casts", ticket.TypeTask, ticket.StatusOpen, "first-fight", []string{"mana", "enemy-behaviour-profiles"}},
	}
	got := make([]row, len(tickets))
	for index, listed := range tickets {
		got[index] = row{listed.ID, listed.Title, listed.Type, listed.Status, listed.Parent, listed.Dependencies}
		if listed.Priority != nil || listed.CreatedAt != "" || listed.UpdatedAt != "" {
			t.Errorf("%s has a priority or timestamps, want none", listed.ID)
		}
	}
	if !reflect.DeepEqual(got, expected) {
		t.Errorf("List() =\n%+v\nwant\n%+v", got, expected)
	}
}

func TestReadyPicksTodoTasksWhoseDependenciesAreDone(t *testing.T) {
	tickets, err := openFixture(t).List()
	if err != nil {
		t.Fatal(err)
	}
	var ready []string
	for _, picked := range graph.Ready(tickets) {
		ready = append(ready, picked.ID)
	}
	// golem-guards and harpy-hastes wait on a todo task, and wraith-casts on a note the vault lacks.
	if expected := []string{"weighted-step-choice"}; !reflect.DeepEqual(ready, expected) {
		t.Errorf("Ready() = %v, want %v", ready, expected)
	}
}

func TestReadCriteria(t *testing.T) {
	tests := []struct {
		id       string
		expected []bool
		first    string
	}{
		{"goal-scoring", []bool{true, true, true}, "`score_goal` returns an integer"},
		{"weighted-step-choice", []bool{true, false, false, false}, "`scoring_target` returns"},
		{"enemy-puzzle", nil, ""},
	}
	store := openFixture(t)
	for _, tt := range tests {
		t.Run(tt.id, func(t *testing.T) {
			read, err := store.Read(tt.id)
			if err != nil {
				t.Fatalf("Read() error = %v", err)
			}
			var done []bool
			for _, criterion := range read.Acceptance {
				done = append(done, criterion.Done)
			}
			if !reflect.DeepEqual(done, tt.expected) {
				t.Errorf("Read() criteria done = %v, want %v", done, tt.expected)
			}
			if len(read.Acceptance) > 0 && !strings.HasPrefix(read.Acceptance[0].Text, tt.first) {
				t.Errorf("first criterion = %q, want prefix %q", read.Acceptance[0].Text, tt.first)
			}
		})
	}
}

func TestResolve(t *testing.T) {
	tests := []struct {
		name          string
		reference     string
		expected      string
		expectedError error
		mentions      []string
	}{
		{"An exact slug", "goal-scoring", "goal-scoring", nil, nil},
		{"A unique prefix", "weigh", "weighted-step-choice", nil, nil},
		{"An exact slug that prefixes another", "mana", "mana", nil, nil},
		{"An ambiguous prefix names the matches", "g", "", backend.ErrAmbiguous, []string{"goal-scoring", "golem-guards"}},
		{"No match", "nonsense", "", backend.ErrNoMatch, []string{"nonsense"}},
		{"An empty reference", "", "", backend.ErrNoMatch, nil},
		{"A doc is not a ticket", "enemy-goals", "", backend.ErrNoMatch, nil},
		{"A template is not a ticket", "task", "", backend.ErrNoMatch, nil},
	}
	store := openFixture(t)
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := store.Resolve(tt.reference)
			if got != tt.expected || !errors.Is(err, tt.expectedError) {
				t.Fatalf("Resolve(%q) = %q, %v, want %q, %v", tt.reference, got, err, tt.expected, tt.expectedError)
			}
			for _, mention := range tt.mentions {
				if !strings.Contains(err.Error(), mention) {
					t.Errorf("Resolve() error = %q, want it to mention %s", err, mention)
				}
			}
		})
	}
}

func TestReadText(t *testing.T) {
	store := openFixture(t)
	text, err := store.(backend.TextReader).ReadText("goal-scoring")
	if err != nil {
		t.Fatalf("ReadText() error = %v", err)
	}
	expected, err := os.ReadFile(filepath.Join(fixture, "docs", "tasks", "goal-scoring.md"))
	if err != nil {
		t.Fatal(err)
	}
	if string(text) != string(expected) {
		t.Errorf("ReadText() = %q, want the note's bytes", text)
	}
}

func TestUnsupportedWrites(t *testing.T) {
	store := openFixture(t)
	tests := []struct {
		name  string
		write func() error
	}{
		{"Apply", func() error {
			return store.Apply("goal-scoring", []backend.Change{backend.UncheckCriterion{Index: 1}})
		}},
		{"Create", func() error {
			_, err := store.Create(ticket.Ticket{Title: "New", Type: ticket.TypeTask, Status: ticket.StatusOpen})
			return err
		}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := tt.write(); !errors.Is(err, backend.ErrUnsupported) {
				t.Errorf("%s() error = %v, want %v", tt.name, err, backend.ErrUnsupported)
			}
		})
	}
}

const profile = `prefix: v
backend: vault
vault:
  tickets:
    task:
      ticket_type: task
      statuses: {todo: open, done: completed}
      fields: {depends_on: dependencies, phase: parent}
`

func TestMappingNotes(t *testing.T) {
	tests := []struct {
		name          string
		notes         map[string]string
		expected      ticket.Ticket
		expectedError error
		mentions      []string
	}{
		{"An unmapped status names the note and the allowed statuses",
			map[string]string{"a.md": "---\ntype: task\nstatus: wip\n---\n# A\n"},
			ticket.Ticket{}, vaultstore.ErrUnknownStatus, []string{"a.md", `"wip"`, `"done", "todo"`}},
		{"A missing status is unmapped",
			map[string]string{"a.md": "---\ntype: task\n---\n# A\n"},
			ticket.Ticket{}, vaultstore.ErrUnknownStatus, []string{"a.md", `""`}},
		{"A note with no H1 takes its slug as title, and links in any form",
			map[string]string{"a.md": "---\ntype: task\nstatus: todo\nphase: \"[[p|Phase]]\"\ndepends_on: [b, \"[[c]]\", [[d]]]\n---\n## Not a title\n"},
			ticket.Ticket{ID: "a", Title: "a", Type: ticket.TypeTask, Status: ticket.StatusOpen, Parent: "p", Dependencies: []string{"b", "c", "d"}}, nil, nil},
		{"An empty phase is no parent",
			map[string]string{"a.md": "---\ntype: task\nstatus: done\nphase:\ndepends_on: []\n---\n# A\n\n## Acceptance criteria\n\n- [x] one\n- [ ] two\n"},
			ticket.Ticket{ID: "a", Title: "A", Type: ticket.TypeTask, Status: ticket.StatusCompleted,
				Acceptance: []ticket.Criterion{{Text: "one", Done: true}, {Text: "two"}}}, nil, nil},
		{"Two parents are rejected",
			map[string]string{"a.md": "---\ntype: task\nstatus: todo\nphase: [p, q]\n---\n"},
			ticket.Ticket{}, vaultstore.ErrTooManyParents, []string{"a.md", "phase"}},
		{"A slug outside the id pattern is invalid",
			map[string]string{"Big Note.md": "---\ntype: task\nstatus: todo\n---\n"},
			ticket.Ticket{}, backend.ErrInvalidFile, []string{"Big Note.md", "id"}},
		{"Two ticket notes with one slug are rejected",
			map[string]string{"x/a.md": "---\ntype: task\nstatus: todo\n---\n", "y/a.md": "---\ntype: task\nstatus: todo\n---\n"},
			ticket.Ticket{}, vaultstore.ErrDuplicateSlug, []string{"a.md"}},
		{"Notes of other types, without a type, and in hidden directories are ignored",
			map[string]string{
				"a.md":           "---\ntype: task\nstatus: todo\n---\n# A\n",
				"spec.md":        "---\ntype: spec\nstatus: wip\n---\n",
				"plain.md":       "# Plain\n",
				"listed.md":      "---\ntype: [task]\nstatus: wip\n---\n",
				".obsidian/b.md": "---\ntype: task\nstatus: wip\n---\n",
				"notes.txt":      "---\ntype: task\nstatus: wip\n---\n",
			},
			ticket.Ticket{ID: "a", Title: "A", Type: ticket.TypeTask, Status: ticket.StatusOpen}, nil, nil},
	}
	backend.Register(backend.KindVault, vaultstore.Open)
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			root := t.TempDir()
			write(t, filepath.Join(root, ticket.DirectoryName, ticket.ConfigFileName), profile)
			for path, text := range tt.notes {
				write(t, filepath.Join(root, backend.DefaultVaultRoot, path), text)
			}
			store, err := backend.Open(root)
			if err != nil {
				t.Fatal(err)
			}

			tickets, err := store.List()

			if !errors.Is(err, tt.expectedError) {
				t.Fatalf("List() error = %v, want %v", err, tt.expectedError)
			}
			for _, mention := range tt.mentions {
				if !strings.Contains(err.Error(), mention) {
					t.Errorf("List() error = %q, want it to mention %s", err, mention)
				}
			}
			if tt.expectedError == nil && (len(tickets) != 1 || !reflect.DeepEqual(tickets[0], tt.expected)) {
				t.Errorf("List() = %+v, want [%+v]", tickets, tt.expected)
			}
		})
	}
}

func write(t *testing.T, path, text string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), backend.DirectoryPermission); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(text), backend.FilePermission); err != nil {
		t.Fatal(err)
	}
}
