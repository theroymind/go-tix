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
)

func openDocFixture(t *testing.T) backend.DocStore {
	t.Helper()
	store, hasDocs := openFixture(t).(backend.DocStore)
	if !hasDocs {
		t.Fatal("the vault backend does not implement DocStore")
	}
	return store
}

func TestListDocs(t *testing.T) {
	tests := []struct {
		name          string
		kind          string
		expected      []string
		expectedError error
		mentions      []string
	}{
		{"Every doc kind, in slug order", "", []string{"enemy-ai", "enemy-choice-is-a-weighted-argmax", "enemy-goals", "turn-order-is-a-wait-counter"}, nil, nil},
		{"One kind", "decision", []string{"enemy-choice-is-a-weighted-argmax", "turn-order-is-a-wait-counter"}, nil, nil},
		{"A kind with one note at the vault root", "spec", []string{"enemy-goals"}, nil, nil},
		{"A kind the profile doesn't define names the kinds it does", "task", nil, vaultstore.ErrUnknownKind,
			[]string{`"task"`, `"decision", "idea", "spec"`}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			store := openDocFixture(t)

			docs, err := store.ListDocs(tt.kind)

			if !errors.Is(err, tt.expectedError) {
				t.Fatalf("ListDocs() error = %v, want %v", err, tt.expectedError)
			}
			for _, mention := range tt.mentions {
				if !strings.Contains(err.Error(), mention) {
					t.Errorf("ListDocs() error = %q, want it to mention %s", err, mention)
				}
			}
			var ids []string
			for _, doc := range docs {
				ids = append(ids, doc.ID)
			}
			if !reflect.DeepEqual(ids, tt.expected) {
				t.Errorf("ListDocs() ids = %v, want %v", ids, tt.expected)
			}
		})
	}
}

func TestReadDoc(t *testing.T) {
	tests := []struct {
		name          string
		id            string
		expected      backend.Doc
		expectedError error
	}{
		{"A decision without an H1 is titled by its slug, and its frontmatter less type and status fills the properties",
			"enemy-choice-is-a-weighted-argmax",
			backend.Doc{
				ID: "enemy-choice-is-a-weighted-argmax", Kind: "decision", Title: "enemy-choice-is-a-weighted-argmax", Status: "accepted",
				Properties: map[string]any{"decided_by": "user", "date": "2026-11-01",
					"tasks": []any{"sim-queries", "goal-scoring", "weighted-step-choice", "golem-guards", "harpy-hastes"}},
				Links: []string{"mana-pool"},
				Path:  filepath.Join("docs", "decisions", "enemy-choice-is-a-weighted-argmax.md"),
			}, nil},
		{"An idea is titled by its H1 and links each slug once",
			"enemy-ai",
			backend.Doc{
				ID: "enemy-ai", Kind: "idea", Title: "Enemy AI: targeting, behaviour and planning", Status: "open",
				Links: []string{"mana-pool", "enemy-goals", "turn-order-is-a-wait-counter", "intent", "pillars"},
				Path:  filepath.Join("docs", "ideas", "enemy-ai.md"),
			}, nil},
		{"A ticket note is not a doc", "golem-guards", backend.Doc{}, backend.ErrNoMatch},
		{"An unknown slug has no match", "nothing", backend.Doc{}, backend.ErrNoMatch},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			store := openDocFixture(t)

			doc, err := store.ReadDoc(tt.id)

			if !errors.Is(err, tt.expectedError) {
				t.Fatalf("ReadDoc() error = %v, want %v", err, tt.expectedError)
			}
			if !reflect.DeepEqual(doc, tt.expected) {
				t.Errorf("ReadDoc() = %#v, want %#v", doc, tt.expected)
			}
		})
	}
}

func TestResolveDoc(t *testing.T) {
	tests := []struct {
		name          string
		reference     string
		expected      string
		expectedError error
	}{
		{"An exact slug", "enemy-goals", "enemy-goals", nil},
		{"A unique prefix", "turn", "turn-order-is-a-wait-counter", nil},
		{"An ambiguous prefix", "enemy", "", backend.ErrAmbiguous},
		{"A ticket slug is no doc", "golem-guards", "", backend.ErrNoMatch},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			store := openDocFixture(t)

			id, err := store.ResolveDoc(tt.reference)

			if !errors.Is(err, tt.expectedError) {
				t.Fatalf("ResolveDoc() error = %v, want %v", err, tt.expectedError)
			}
			if id != tt.expected {
				t.Errorf("ResolveDoc() = %q, want %q", id, tt.expected)
			}
		})
	}
}

func TestReadDocTextIsTheNote(t *testing.T) {
	store := openDocFixture(t)
	path := filepath.Join(fixture, "docs", "ideas", "enemy-ai.md")
	expected, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}

	text, err := store.ReadDocText("enemy-ai")

	if err != nil {
		t.Fatalf("ReadDocText() error = %v", err)
	}
	if string(text) != string(expected) {
		t.Errorf("ReadDocText() =\n%s\nwant the note at %s", text, path)
	}
}

func TestReadDocRejectsAStatusOutsideTheKind(t *testing.T) {
	store, root := copyFixture(t)
	path := filepath.Join(root, "docs", "decisions", "turn-order-is-a-wait-counter.md")
	original, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	broken := strings.Replace(string(original), "status: proposed\n", "status: done\n", 1)
	if err := os.WriteFile(path, []byte(broken), 0o644); err != nil {
		t.Fatal(err)
	}

	_, err = store.(backend.DocStore).ReadDoc("turn-order-is-a-wait-counter")

	if !errors.Is(err, vaultstore.ErrUnknownStatus) {
		t.Fatalf("ReadDoc() error = %v, want %v", err, vaultstore.ErrUnknownStatus)
	}
	for _, mention := range []string{`"done"`, `"proposed", "accepted", "deferred", "superseded"`} {
		if !strings.Contains(err.Error(), mention) {
			t.Errorf("ReadDoc() error = %q, want it to mention %s", err, mention)
		}
	}
}

func TestApplyDoc(t *testing.T) {
	const note = "docs/decisions/turn-order-is-a-wait-counter.md"
	tests := []struct {
		name          string
		changes       []backend.Change
		replace       [2]string
		expectedError error
		mentions      []string
	}{
		{"A status in the kind's list changes one line, and a timestamp the note lacks is skipped",
			[]backend.Change{backend.SetField{Field: ticket.FieldStatus, Value: "accepted"},
				backend.SetField{Field: ticket.FieldUpdatedAt, Value: "2026-11-02T10:00:00Z"}},
			[2]string{"status: proposed\n", "status: accepted\n"}, nil, nil},
		{"A status outside the kind's list names the note and the kind's statuses",
			[]backend.Change{backend.SetField{Field: ticket.FieldStatus, Value: "done"}},
			[2]string{}, vaultstore.ErrUnknownStatus, []string{"turn-order-is-a-wait-counter.md", `"done"`, `"proposed", "accepted", "deferred", "superseded"`}},
		{"Any other field is written as a property",
			[]backend.Change{backend.SetField{Field: "decided_by", Value: "user"}},
			[2]string{"decided_by:\n", "decided_by: user\n"}, nil, nil},
		{"A failing change leaves the note as it was",
			[]backend.Change{backend.SetField{Field: "decided_by", Value: "user"}, backend.SetField{Field: ticket.FieldStatus, Value: "done"}},
			[2]string{}, vaultstore.ErrUnknownStatus, nil},
		{"The type property is unsupported",
			[]backend.Change{backend.SetField{Field: "type", Value: "idea"}},
			[2]string{}, backend.ErrUnsupported, []string{"type"}},
		{"The title is the H1, not a property",
			[]backend.Change{backend.SetField{Field: ticket.FieldTitle, Value: "Turns"}},
			[2]string{}, backend.ErrUnsupported, []string{ticket.FieldTitle}},
		{"A criterion change is unsupported on a doc",
			[]backend.Change{backend.CheckCriterion{Index: 1}},
			[2]string{}, backend.ErrUnsupported, []string{"CheckCriterion"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			store, root := copyFixture(t)
			path := filepath.Join(root, note)
			original, err := os.ReadFile(path)
			if err != nil {
				t.Fatal(err)
			}

			err = store.(backend.DocStore).ApplyDoc("turn-order-is-a-wait-counter", tt.changes)

			if !errors.Is(err, tt.expectedError) {
				t.Fatalf("ApplyDoc() error = %v, want %v", err, tt.expectedError)
			}
			for _, mention := range tt.mentions {
				if !strings.Contains(err.Error(), mention) {
					t.Errorf("ApplyDoc() error = %q, want it to mention %s", err, mention)
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
				t.Errorf("note after ApplyDoc() =\n%s\nwant\n%s", got, expected)
			}
		})
	}
}
