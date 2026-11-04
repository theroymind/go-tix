package vaultstore_test

import (
	"errors"
	"maps"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/theroymind/go-tix/contracts/ticket"
	"github.com/theroymind/go-tix/internal/tix/backend"
	"github.com/theroymind/go-tix/internal/tix/backend/vaultstore"
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
		docType       string
		expected      []string
		expectedError error
		mentions      []string
	}{
		{"Every doc type, in slug order", "", []string{"enemy-ai", "enemy-choice-is-a-weighted-argmax", "enemy-goals", "turn-order-is-a-wait-counter"}, nil, nil},
		{"One type", "decision", []string{"enemy-choice-is-a-weighted-argmax", "turn-order-is-a-wait-counter"}, nil, nil},
		{"A type with one note at the vault root", "spec", []string{"enemy-goals"}, nil, nil},
		{"A type the profile doesn't define names the types it does", "task", nil, vaultstore.ErrUnknownType,
			[]string{`"task"`, `"decision", "idea", "spec"`}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			store := openDocFixture(t)

			docs, err := store.ListDocs(tt.docType)

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
				ID: "enemy-choice-is-a-weighted-argmax", Type: "decision", Title: "enemy-choice-is-a-weighted-argmax", Status: "accepted",
				Properties: map[string]any{"decided_by": "user", "date": "2026-11-01",
					"tasks": []any{"sim-queries", "goal-scoring", "weighted-step-choice", "golem-guards", "harpy-hastes"}},
				Links: []string{"mana-pool"},
				Path:  filepath.Join("docs", "decisions", "enemy-choice-is-a-weighted-argmax.md"),
			}, nil},
		{"An idea is titled by its H1 and links each slug once",
			"enemy-ai",
			backend.Doc{
				ID: "enemy-ai", Type: "idea", Title: "Enemy AI: targeting, behaviour and planning", Status: "open",
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

func TestReadDocRejectsAStatusOutsideTheType(t *testing.T) {
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
		{"A status in the type's list changes one line, and a timestamp the note lacks is skipped",
			[]backend.Change{backend.SetField{Field: ticket.FieldStatus, Value: "accepted"},
				backend.SetField{Field: ticket.FieldUpdatedAt, Value: "2026-11-02T10:00:00Z"}},
			[2]string{"status: proposed\n", "status: accepted\n"}, nil, nil},
		{"A status outside the type's list names the note and the type's statuses",
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

func TestBacklinks(t *testing.T) {
	tests := []struct {
		name     string
		id       string
		expected []string
	}{
		{"Tickets, docs and unmapped notes alike, in slug order, templates skipped", "enemy-goals",
			[]string{"README", "enemy-ai", "enemy-puzzle", "goal-scoring", "golem-guards", "harpy-hastes", "sim-queries", "weighted-step-choice"}},
		{"A slug no note links to", "nobody-links-here", nil},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			store := openDocFixture(t)

			slugs, err := store.Backlinks(tt.id)

			if err != nil {
				t.Fatalf("Backlinks() error = %v", err)
			}
			if !reflect.DeepEqual(slugs, tt.expected) {
				t.Errorf("Backlinks() = %v, want %v", slugs, tt.expected)
			}
		})
	}
}

func TestCreateDoc(t *testing.T) {
	today := time.Now().Format(backend.DateLayout)
	tests := []struct {
		name          string
		docType       string
		value         backend.Doc
		expectedID    string
		expectedPath  string
		expectedText  string
		expectedError error
		mentions      []string
	}{
		{"A decision starts from its template with the first status and today's date, beside the other decisions",
			"decision", backend.Doc{Title: "Bosses plan with GOAP"}, "bosses-plan-with-goap", "docs/decisions/bosses-plan-with-goap.md",
			"---\ntype: decision\nstatus: proposed\ndecided_by:\ndate: \"" + today + "\"\ntasks: []\n---\n\n" +
				"The decision, stated so it reads on its own.\n\n## Rejected\n\n- an alternative and why it lost\n", nil, nil},
		{"A given status and properties are written",
			"decision", backend.Doc{Title: "Bosses plan with GOAP", Status: "accepted", Properties: map[string]any{"decided_by": "user"}},
			"bosses-plan-with-goap", "docs/decisions/bosses-plan-with-goap.md",
			"---\ntype: decision\nstatus: accepted\ndecided_by: user\ndate: \"" + today + "\"\ntasks: []\n---\n\n" +
				"The decision, stated so it reads on its own.\n\n## Rejected\n\n- an alternative and why it lost\n", nil, nil},
		{"A status outside the type's list writes nothing",
			"decision", backend.Doc{Title: "Bosses plan with GOAP", Status: "done"}, "", "", "", vaultstore.ErrUnknownStatus, []string{`"done"`}},
		{"A type without a template writes nothing",
			"idea", backend.Doc{Title: "Bosses"}, "", "", "", vaultstore.ErrNoDocTemplate, []string{`"idea"`, `the types with one are "decision"`}},
		{"A type outside the profile names the types in it",
			"task", backend.Doc{Title: "Bosses"}, "", "", "", vaultstore.ErrUnknownType, []string{`"task"`, `"decision", "idea", "spec"`}},
		{"A slug a doc has is taken",
			"decision", backend.Doc{Title: "Turn order is a wait counter"}, "", "", "", vaultstore.ErrSlugTaken,
			[]string{`"turn-order-is-a-wait-counter"`, filepath.Join("docs", "decisions", "turn-order-is-a-wait-counter.md")}},
		{"A slug a ticket has is taken",
			"decision", backend.Doc{Title: "Golem guards"}, "", "", "", vaultstore.ErrSlugTaken, []string{`"golem-guards"`, filepath.Join("docs", "tasks", "golem-guards.md")}},
		{"A title with nothing to slug is refused",
			"decision", backend.Doc{Title: "!!!"}, "", "", "", vaultstore.ErrEmptySlug, []string{`"!!!"`}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			store, root := copyFixture(t)
			before := vaultFiles(t, root)

			id, err := store.(backend.DocStore).CreateDoc(tt.docType, tt.value)

			if !errors.Is(err, tt.expectedError) {
				t.Fatalf("CreateDoc() error = %v, want %v", err, tt.expectedError)
			}
			for _, mention := range tt.mentions {
				if !strings.Contains(err.Error(), mention) {
					t.Errorf("CreateDoc() error = %q, want it to mention %s", err, mention)
				}
			}
			if id != tt.expectedID {
				t.Errorf("CreateDoc() = %q, want %q", id, tt.expectedID)
			}
			if tt.expectedPath != "" {
				before[tt.expectedPath] = tt.expectedText
			}
			if after := vaultFiles(t, root); !maps.Equal(before, after) {
				t.Errorf("files after CreateDoc() =\n%v\nwant\n%v", after, before)
			}
		})
	}
}

func TestCreateDocFillsTheTitlePlaceholder(t *testing.T) {
	store, root := copyFixture(t)
	template := filepath.Join(root, "docs", "templates", "decision.md")
	text, err := os.ReadFile(template)
	if err != nil {
		t.Fatal(err)
	}
	withTitle := strings.Replace(string(text), "---\n\nThe decision", "---\n\n# {{title}}\n\nThe decision", 1)
	if err := os.WriteFile(template, []byte(withTitle), 0o644); err != nil {
		t.Fatal(err)
	}

	id, err := store.(backend.DocStore).CreateDoc("decision", backend.Doc{Title: "Bosses plan with GOAP"})

	if err != nil {
		t.Fatalf("CreateDoc() error = %v", err)
	}
	created, err := store.(backend.DocStore).ReadDoc(id)
	if err != nil {
		t.Fatalf("ReadDoc() error = %v", err)
	}
	if created.Title != "Bosses plan with GOAP" {
		t.Errorf("created doc's title = %q, want the title given", created.Title)
	}
}
