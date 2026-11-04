package tix

import (
	"bytes"
	"errors"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/theroymind/railyard/internal/contracts/verdict"
	"github.com/theroymind/railyard/internal/tix/backend"
	"github.com/theroymind/railyard/internal/tix/judge"
)

const judgeConfig = `prefix: odin
backend: vault
vault:
  tickets:
    task:
      ticket_type: task
      statuses: {todo: open, done: completed}
  docs:
    decision:
      statuses: [proposed, accepted, superseded]
`

var judgeVault = map[string]string{
	"decisions/wait-counter.md": "---\ntype: decision\nstatus: accepted\nsummary: Turn order is a wait counter.\n---\n\n# Wait counter\n\nSpeed drains a counter.\n",
	"decisions/initiative.md":   "---\ntype: decision\nstatus: proposed\nsummary: Turn order is fixed initiative.\n---\n\n# Initiative\n\nReplaces [[wait-counter]].\n",
	"decisions/old-order.md":    "---\ntype: decision\nstatus: superseded\nsummary: Turn order is round robin.\n---\n\n# Old order\n",
	"tasks/mana.md":             "---\ntype: task\nstatus: todo\nsummary: Add mana.\n---\n\n# Mana\n\nUses [[initiative]].\n",
}

// fakeJudge answers every prompt with one verdict and keeps the prompts it was asked.
type fakeJudge struct {
	answer  verdict.Verdict
	prompts *[]string
}

func (fake fakeJudge) Judge(prompt string) (verdict.Verdict, error) {
	*fake.prompts = append(*fake.prompts, prompt)
	return fake.answer, nil
}

func TestJudgeNotes(t *testing.T) {
	refuse := func(duplicateOf, conflictsWith string) verdict.Verdict {
		return verdict.Verdict{Verdict: verdict.Refuse, DuplicateOf: duplicateOf, ConflictsWith: conflictsWith, Reason: "Same choice."}
	}
	tests := []struct {
		name               string
		draft              string
		answer             verdict.Verdict
		search             searchFunction
		expectedError      error
		expectedLines      []string
		expectedNeighbours []string
	}{
		{"An acceptance passes", "initiative", verdict.Verdict{Verdict: verdict.Accept, Reason: "New."}, nil, nil,
			[]string{"initiative: accept: New."}, []string{"mana", "wait-counter"}},
		{"A duplicate is refused naming the note", "initiative", refuse("wait-counter", ""), nil, ErrJudgeRefused,
			[]string{"initiative: refuse: duplicates wait-counter (decision, accepted): Same choice."}, nil},
		{"A conflict with an accepted decision points at tix supersede", "initiative", refuse("", "wait-counter"), nil, ErrJudgeRefused,
			[]string{"initiative: refuse: conflicts with wait-counter (decision, accepted): Same choice.",
				"  if initiative replaces wait-counter, run tix supersede wait-counter initiative"}, nil},
		{"A conflict with a note of another type doesn't point at tix supersede", "initiative", refuse("", "mana"), nil, ErrJudgeRefused,
			[]string{"initiative: refuse: conflicts with mana (task, open): Same choice."}, nil},
		{"Search hits join the context neighbours once each", "initiative", verdict.Verdict{Verdict: verdict.Accept, Reason: "New."},
			func(string) ([]string, error) { return []string{"initiative", "mana", "old-order"}, nil }, nil,
			[]string{"initiative: accept"}, []string{"mana", "wait-counter", "old-order"}},
		{"A conflict with a superseded doc doesn't point at tix supersede", "initiative", refuse("", "old-order"),
			func(string) ([]string, error) { return []string{"old-order"}, nil }, ErrJudgeRefused,
			[]string{"conflicts with old-order (decision, superseded)"}, nil},
		{"Search failing falls back to the context neighbours and says so", "initiative", verdict.Verdict{Verdict: verdict.Accept, Reason: "New."},
			func(string) ([]string, error) { return nil, errors.New("endpoint down") }, nil,
			[]string{"search unavailable (endpoint down); judging against context neighbours only", "initiative: accept"},
			[]string{"mana", "wait-counter"}},
		{"A verdict naming a note the judge wasn't shown fails", "initiative", refuse("old-order", ""), nil, judge.ErrUnknownNote, nil, nil},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			root := writeJudgeVault(t)
			store, err := backend.Open(root)
			if err != nil {
				t.Fatal(err)
			}
			notes, err := contextNotes(store)
			if err != nil {
				t.Fatal(err)
			}
			var prompts []string
			session := judging{store: store, notes: notes, search: tt.search, judge: fakeJudge{answer: tt.answer, prompts: &prompts}}
			var stdout bytes.Buffer

			err = session.judgeNotes([]string{tt.draft}, &stdout)

			if !errors.Is(err, tt.expectedError) {
				t.Fatalf("judgeNotes() error = %v, expected %v; output %q", err, tt.expectedError, stdout.String())
			}
			for _, line := range tt.expectedLines {
				if !strings.Contains(stdout.String(), line) {
					t.Errorf("output = %q, expected it to contain %q", stdout.String(), line)
				}
			}
			if tt.expectedNeighbours != nil {
				shown := neighboursShown(prompts[0])
				if !slices.Equal(shown, tt.expectedNeighbours) {
					t.Errorf("neighbours shown = %v, expected %v", shown, tt.expectedNeighbours)
				}
			}
		})
	}
}

// neighboursShown returns the ids of the notes the prompt sets out after the draft, in order.
func neighboursShown(prompt string) []string {
	_, after, _ := strings.Cut(prompt, neighboursMarker)
	var ids []string
	for line := range strings.SplitSeq(after, "\n") {
		if heading, isHeading := strings.CutPrefix(line, "## "); isHeading {
			id, _, _ := strings.Cut(heading, " ")
			ids = append(ids, id)
		}
	}
	return ids
}

const neighboursMarker = "# Its neighbours"

func writeJudgeVault(t *testing.T) string {
	t.Helper()
	root := t.TempDir()
	files := map[string]string{".tix/config.yml": judgeConfig}
	for path, text := range judgeVault {
		files[filepath.Join("docs", path)] = text
	}
	for path, text := range files {
		full := filepath.Join(root, path)
		if err := os.MkdirAll(filepath.Dir(full), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(full, []byte(text), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	return root
}
