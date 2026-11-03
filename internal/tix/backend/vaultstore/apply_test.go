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
	"github.com/theroymind/railyard/internal/tix/vault"
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

func TestApplyCriteria(t *testing.T) {
	const (
		golemSecond  = "- [x] `GOLEM_ATTACK_WEIGHTS`"
		weightedArg  = "- [ ] The argmax"
		criteriaHead = "## Acceptance criteria\n"
	)
	golemCriteria := criteriaHead + "\n" +
		"- [x] On a constructed world with a party member in killing range, the Golem attacks that member; with none in range and no Guard, it defends; with none in range and Guard held, it attacks the lowest-HP member.\n" +
		"- [x] `GOLEM_ATTACK_WEIGHTS` and `GOLEM_DEFEND_WEIGHTS` live in `balance.odin`, and an assertion there states that a killing-blow Attack outscores Defend under the Golem's disposition.\n" +
		"- [x] The determinism test's scripted fight includes a Golem guard and still matches on both worlds.\n"
	lastGolem := "- [x] The determinism test's scripted fight includes a Golem guard and still matches on both worlds.\n"
	tests := []struct {
		name          string
		slug          string
		setup         [2]string
		changes       []backend.Change
		replace       [2]string
		expectedError error
		mentions      []string
	}{
		{"Ticking a criterion changes only its mark", "weighted-step-choice", [2]string{},
			[]backend.Change{backend.CheckCriterion{Index: 2}},
			[2]string{weightedArg, "- [x] The argmax"}, nil, nil},
		{"Unticking a criterion changes only its mark", "goal-scoring", [2]string{},
			[]backend.Change{backend.UncheckCriterion{Index: 1}},
			[2]string{"- [x] `score_goal`", "- [ ] `score_goal`"}, nil, nil},
		{"An index past the last criterion names the index and the count", "golem-guards", [2]string{},
			[]backend.Change{backend.CheckCriterion{Index: 9}},
			[2]string{}, vaultstore.ErrCriterionOutOfRange, []string{"golem-guards.md", "criterion 9", "has 3"}},
		{"Index 0 is out of range", "golem-guards", [2]string{},
			[]backend.Change{backend.UncheckCriterion{Index: 0}},
			[2]string{}, vaultstore.ErrCriterionOutOfRange, []string{"criterion 0", "has 3"}},
		{"A negative index is out of range", "golem-guards", [2]string{},
			[]backend.Change{backend.SetCriterion{Index: -1, Text: "x"}},
			[2]string{}, vaultstore.ErrCriterionOutOfRange, []string{"criterion -1", "has 3"}},
		{"Setting a criterion's text keeps its mark", "golem-guards", [2]string{},
			[]backend.Change{backend.SetCriterion{Index: 2, Text: "Weights live in `balance.odin`."}},
			[2]string{golemSecond + " and `GOLEM_DEFEND_WEIGHTS` live in `balance.odin`, and an assertion there states that a killing-blow Attack outscores Defend under the Golem's disposition.\n",
				"- [x] Weights live in `balance.odin`.\n"}, nil, nil},
		{"Setting an unticked criterion's text keeps it unticked", "weighted-step-choice", [2]string{},
			[]backend.Change{backend.SetCriterion{Index: 2, Text: "[x] kept verbatim"}},
			[2]string{"- [ ] The argmax takes the higher score, the earlier step on a tie, and falls through to Attack when every other step is unaffordable or fails `check_command`; each case has a test on a constructed world.\n",
				"- [ ] [x] kept verbatim\n"}, nil, nil},
		{"Text spanning lines is refused", "golem-guards", [2]string{},
			[]backend.Change{backend.SetCriterion{Index: 1, Text: "a\nb"}},
			[2]string{}, vault.ErrMultilineText, nil},
		{"Adding a criterion appends it after the last checkbox", "golem-guards", [2]string{},
			[]backend.Change{backend.AddCriterion{Text: "Golems guard twice."}},
			[2]string{lastGolem, lastGolem + "- [ ] Golems guard twice.\n"}, nil, nil},
		{"An added criterion copies the last checkbox's marker and indent", "golem-guards", [2]string{lastGolem, "  * [x] The determinism test's scripted fight includes a Golem guard and still matches on both worlds.\n"},
			[]backend.Change{backend.AddCriterion{Text: "Golems guard twice."}},
			[2]string{"worlds.\n\n## Verification", "worlds.\n  * [ ] Golems guard twice.\n\n## Verification"}, nil, nil},
		{"An added criterion goes after the last checkbox, before prose that follows it", "golem-guards", [2]string{lastGolem, lastGolem + "\nSome prose.\n"},
			[]backend.Change{backend.AddCriterion{Text: "Golems guard twice."}},
			[2]string{lastGolem, lastGolem + "- [ ] Golems guard twice.\n"}, nil, nil},
		{"Adding to an empty section puts the criterion after the heading's blank line", "golem-guards", [2]string{golemCriteria, criteriaHead},
			[]backend.Change{backend.AddCriterion{Text: "Golems guard twice."}},
			[2]string{criteriaHead + "\n## Verification", criteriaHead + "\n- [ ] Golems guard twice.\n\n## Verification"}, nil, nil},
		{"Adding to a section of prose keeps the prose its own paragraph", "golem-guards", [2]string{golemCriteria, criteriaHead + "\nTo be decided.\n"},
			[]backend.Change{backend.AddCriterion{Text: "Golems guard twice."}},
			[2]string{criteriaHead + "\nTo be decided.", criteriaHead + "\n- [ ] Golems guard twice.\n\nTo be decided."}, nil, nil},
		{"An added criterion goes after the last checkbox's wrapped lines", "golem-guards", [2]string{lastGolem, lastGolem + "  on both worlds.\n    and deeper.\n"},
			[]backend.Change{backend.AddCriterion{Text: "Golems guard twice."}},
			[2]string{"    and deeper.\n", "    and deeper.\n- [ ] Golems guard twice.\n"}, nil, nil},
		{"Added criteria can then be ticked by their new index", "golem-guards", [2]string{},
			[]backend.Change{backend.AddCriterion{Text: "Golems guard twice."}, backend.CheckCriterion{Index: 4}},
			[2]string{lastGolem, lastGolem + "- [x] Golems guard twice.\n"}, nil, nil},
		{"A batch with one bad change leaves the note as it was", "weighted-step-choice", [2]string{},
			[]backend.Change{backend.CheckCriterion{Index: 2}, backend.AddCriterion{Text: "more"}, backend.CheckCriterion{Index: 9}},
			[2]string{}, vaultstore.ErrCriterionOutOfRange, []string{"criterion 9", "has 5"}},
		{"A note without the criteria heading can't be ticked", "golem-guards", [2]string{criteriaHead, "## Criteria\n"},
			[]backend.Change{backend.CheckCriterion{Index: 1}},
			[2]string{}, vault.ErrSectionNotFound, []string{"Acceptance criteria"}},
		{"A note without the criteria heading can't take a new criterion", "golem-guards", [2]string{criteriaHead, "## Criteria\n"},
			[]backend.Change{backend.AddCriterion{Text: "x"}},
			[2]string{}, vault.ErrSectionNotFound, nil},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			store, root := copyFixture(t)
			path := filepath.Join(root, "docs/tasks", tt.slug+".md")
			fixtureText, err := os.ReadFile(path)
			if err != nil {
				t.Fatal(err)
			}
			original := string(fixtureText)
			if tt.setup[0] != "" {
				if !strings.Contains(original, tt.setup[0]) {
					t.Fatalf("fixture lacks %q", tt.setup[0])
				}
				original = strings.Replace(original, tt.setup[0], tt.setup[1], 1)
				write(t, path, original)
			}

			err = store.Apply(tt.slug, tt.changes)

			if !errors.Is(err, tt.expectedError) {
				t.Fatalf("Apply() error = %v, want %v", err, tt.expectedError)
			}
			for _, mention := range tt.mentions {
				if !strings.Contains(err.Error(), mention) {
					t.Errorf("Apply() error = %q, want it to mention %s", err, mention)
				}
			}
			expected := original
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

func TestApplyAddsACriterionToASectionThatEndsTheNote(t *testing.T) {
	const heading = "## Acceptance criteria"
	add := []backend.Change{backend.AddCriterion{Text: "new"}}
	tests := []struct {
		name     string
		section  string
		expected string
	}{
		{"A bare heading with a trailing newline keeps it", heading + "\n", heading + "\n- [ ] new\n"},
		{"A bare heading without a trailing newline gains none", heading, heading + "\n- [ ] new"},
		{"A heading and a blank line keep the blank line", heading + "\n\n", heading + "\n\n- [ ] new\n"},
		{"A last checkbox is followed by the new one", heading + "\n\n- [x] a\n", heading + "\n\n- [x] a\n- [ ] new\n"},
		{"A wrapped last checkbox keeps its wrapped line", heading + "\n\n- [x] a\n  wrapped", heading + "\n\n- [x] a\n  wrapped\n- [ ] new"},
		{"A CRLF note gets a CRLF line", heading + "\r\n", heading + "\r\n- [ ] new\r\n"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			store, root := copyFixture(t)
			path := filepath.Join(root, "docs/tasks/golem-guards.md")
			fixtureText, err := os.ReadFile(path)
			if err != nil {
				t.Fatal(err)
			}
			head, _, found := strings.Cut(string(fixtureText), heading)
			if !found {
				t.Fatalf("fixture lacks %q", heading)
			}
			write(t, path, head+tt.section)

			if err := store.Apply("golem-guards", add); err != nil {
				t.Fatalf("Apply() error = %v", err)
			}

			got, err := os.ReadFile(path)
			if err != nil {
				t.Fatal(err)
			}
			if expected := head + tt.expected; string(got) != expected {
				t.Errorf("note after Apply() =\n%q\nwant\n%q", got, expected)
			}
		})
	}
}
