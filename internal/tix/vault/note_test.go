package vault

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

var fixtures = []string{
	"goal-scoring.md",                      // task
	"enemy-choice-is-a-weighted-argmax.md", // decision without an H1
	"enemy-goals.md",                       // spec with tables
	"enemy-puzzle.md",                      // phase
	"decision-template.md",                 // empty value and a {{placeholder}}
}

func readFixture(t *testing.T, name string) []byte {
	t.Helper()
	text, err := os.ReadFile(filepath.Join("testdata", name))
	if err != nil {
		t.Fatal(err)
	}
	return text
}

func parseFixture(t *testing.T, name string) *Note {
	t.Helper()
	note, err := Parse(name, readFixture(t, name))
	if err != nil {
		t.Fatalf("Parse(%s): %v", name, err)
	}
	return note
}

func toCRLF(text []byte) []byte {
	return []byte(strings.ReplaceAll(string(text), "\n", "\r\n"))
}

func TestRenderRoundTrip(t *testing.T) {
	type testCase struct {
		name string
		text []byte
	}
	tests := []testCase{
		{name: "no frontmatter", text: []byte("# Title\n\nBody\n")},
		{name: "no trailing newline", text: []byte("---\ntype: idea\n---\nBody")},
		{name: "ends at closing delimiter", text: []byte("---\ntype: idea\n---")},
		{name: "empty frontmatter", text: []byte("---\n---\nBody\n")},
		{name: "empty", text: []byte("")},
		{name: "rule but no frontmatter", text: []byte("Intro\n---\nmore\n")},
	}
	for _, name := range fixtures {
		tests = append(tests,
			testCase{name: name, text: readFixture(t, name)},
			testCase{name: name + " crlf", text: toCRLF(readFixture(t, name))},
		)
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			note, err := Parse("notes/"+test.name, test.text)
			if err != nil {
				t.Fatalf("Parse: %v", err)
			}
			if got := note.Render(); string(got) != string(test.text) {
				t.Errorf("Render changed the note:\n got %q\nwant %q", got, test.text)
			}
		})
	}
}

func TestParse(t *testing.T) {
	tests := []struct {
		name      string
		path      string
		fixture   string
		wantSlug  string
		wantType  string
		wantFirst string
	}{
		{name: "task", path: "docs/tasks/goal-scoring.md", fixture: "goal-scoring.md", wantSlug: "goal-scoring", wantType: "task", wantFirst: ""},
		{name: "decision", path: "docs/decisions/enemy-choice-is-a-weighted-argmax.md", fixture: "enemy-choice-is-a-weighted-argmax.md", wantSlug: "enemy-choice-is-a-weighted-argmax", wantType: "decision", wantFirst: ""},
		{name: "phase", path: "enemy-puzzle.md", fixture: "enemy-puzzle.md", wantSlug: "enemy-puzzle", wantType: "phase", wantFirst: ""},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			note, err := Parse(test.path, readFixture(t, test.fixture))
			if err != nil {
				t.Fatal(err)
			}
			if note.Slug != test.wantSlug || note.Path != test.path {
				t.Errorf("Slug, Path = %q, %q; want %q, %q", note.Slug, note.Path, test.wantSlug, test.path)
			}
			if got := note.Property("type"); got == nil || got.Value != test.wantType {
				t.Errorf("Property(type) = %v, want %q", got, test.wantType)
			}
			if got := note.Property("missing"); got != nil {
				t.Errorf("Property(missing) = %v, want nil", got)
			}
			if note.Body[0] != test.wantFirst {
				t.Errorf("Body[0] = %q, want %q", note.Body[0], test.wantFirst)
			}
		})
	}
}

func TestParseErrors(t *testing.T) {
	tests := []struct {
		name string
		text string
		want error
	}{
		{name: "unterminated", text: "---\ntype: task\n\nBody\n", want: ErrUnterminatedFrontmatter},
		{name: "invalid yaml", text: "---\ndepends_on: [] []\n---\n", want: ErrInvalidFrontmatter},
		{name: "not a mapping", text: "---\n- a\n- b\n---\n", want: ErrFrontmatterNotMapping},
		{name: "duplicate key", text: "---\nstatus: todo\nstatus: done\n---\n", want: ErrDuplicateKey},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			_, err := Parse("note.md", []byte(test.text))
			if !errors.Is(err, test.want) {
				t.Errorf("Parse error = %v, want %v", err, test.want)
			}
		})
	}
}

func TestRead(t *testing.T) {
	tests := []struct {
		name    string
		path    string
		wantErr error
	}{
		{name: "fixture", path: filepath.Join("testdata", "goal-scoring.md")},
		{name: "missing", path: filepath.Join("testdata", "missing.md"), wantErr: os.ErrNotExist},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			note, err := Read(test.path)
			if !errors.Is(err, test.wantErr) {
				t.Fatalf("Read error = %v, want %v", err, test.wantErr)
			}
			if err == nil && note.Slug != "goal-scoring" {
				t.Errorf("Slug = %q", note.Slug)
			}
		})
	}
}

// lineDiff returns the lines of got between the prefix and suffix it shares with want, and fails
// unless got replaces as many lines of want as it has, less added.
func lineDiff(t *testing.T, want, got string, added int) []string {
	t.Helper()
	wantLines, gotLines := strings.Split(want, "\n"), strings.Split(got, "\n")
	prefix := 0
	for prefix < len(wantLines) && prefix < len(gotLines) && wantLines[prefix] == gotLines[prefix] {
		prefix++
	}
	suffix := 0
	for suffix < len(wantLines)-prefix && suffix < len(gotLines)-prefix &&
		wantLines[len(wantLines)-1-suffix] == gotLines[len(gotLines)-1-suffix] {
		suffix++
	}
	changed := gotLines[prefix : len(gotLines)-suffix]
	if replaced := len(wantLines) - prefix - suffix; replaced != len(changed)-added {
		t.Fatalf("replaced %d lines with %q, want %d added", replaced, changed, added)
	}
	return changed
}

func TestSetProperty(t *testing.T) {
	tests := []struct {
		name      string
		text      []byte
		key       string
		value     any
		wantLine  string
		wantAdded int
	}{
		{name: "task status", text: readFixture(t, "goal-scoring.md"), key: "status", value: "todo", wantLine: "status: todo"},
		{name: "task flow list", text: readFixture(t, "goal-scoring.md"), key: "depends_on", value: []string{"sim-queries", "enemy-goals"}, wantLine: "depends_on: [sim-queries, enemy-goals]"},
		{name: "quoted date keeps its quotes", text: readFixture(t, "enemy-choice-is-a-weighted-argmax.md"), key: "date", value: "2026-12-01", wantLine: `date: "2026-12-01"`},
		{name: "decision flow list", text: readFixture(t, "enemy-choice-is-a-weighted-argmax.md"), key: "tasks", value: []string{"a", "b"}, wantLine: "tasks: [a, b]"},
		{name: "empty value", text: readFixture(t, "decision-template.md"), key: "decided_by", value: "user", wantLine: "decided_by: user"},
		{name: "new key appended", text: readFixture(t, "enemy-choice-is-a-weighted-argmax.md"), key: "superseded_by", value: "new-decision", wantLine: "superseded_by: new-decision", wantAdded: 1},
		{name: "crlf", text: toCRLF(readFixture(t, "goal-scoring.md")), key: "status", value: "todo", wantLine: "status: todo\r"},
		{name: "line comment kept", text: []byte("---\nstatus: todo # set by tix\nphase: x\n---\n"), key: "status", value: "done", wantLine: "status: done # set by tix"},
		{name: "plain int stays plain", text: []byte("---\nscope: 2\nstatus: todo\n---\n"), key: "scope", value: "3", wantLine: "scope: 3"},
		{name: "plain date stays plain", text: []byte("---\nupdated_at: 2026-11-01\nstatus: todo\n---\n"), key: "updated_at", value: "2026-11-02", wantLine: "updated_at: 2026-11-02"},
		{name: "a string that no longer resolves to the old tag is plain text", text: []byte("---\nscope: 2\nstatus: todo\n---\n"), key: "scope", value: "large", wantLine: "scope: large"},
		{name: "a string that would change type is quoted", text: []byte("---\nscope: small\nstatus: todo\n---\n"), key: "scope", value: "3", wantLine: `scope: "3"`},
		{name: "block list", text: []byte("---\ntags:\n  - a\n  - b\n\n# trailing\nstatus: todo\n---\n"), key: "tags", value: []string{"c", "d"}, wantLine: "  - c"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			note, err := Parse("note.md", test.text)
			if err != nil {
				t.Fatal(err)
			}
			if err := note.SetProperty(test.key, test.value); err != nil {
				t.Fatalf("SetProperty: %v", err)
			}
			changed := lineDiff(t, string(test.text), string(note.Render()), test.wantAdded)
			if len(changed) == 0 || changed[0] != test.wantLine {
				t.Errorf("changed lines = %q, want first %q", changed, test.wantLine)
			}
			if test.name != "block list" && len(changed) != 1 {
				t.Errorf("changed lines = %q, want exactly one", changed)
			}
			if got := note.Property(test.key); got == nil {
				t.Errorf("Property(%s) = nil after SetProperty", test.key)
			}
		})
	}
}

func TestSetPropertyTwiceAndWithoutFrontmatter(t *testing.T) {
	tests := []struct {
		name  string
		text  string
		edits [][2]string
		want  string
	}{
		{name: "two edits", text: "---\ntype: task\nstatus: todo\nphase: x\n---\nBody\n", edits: [][2]string{{"status", "done"}, {"phase", "y"}}, want: "---\ntype: task\nstatus: done\nphase: y\n---\nBody\n"},
		{name: "no frontmatter", text: "Body\n", edits: [][2]string{{"type", "idea"}}, want: "---\ntype: idea\n---\nBody\n"},
		{name: "empty frontmatter", text: "---\n---\nBody\n", edits: [][2]string{{"type", "idea"}}, want: "---\ntype: idea\n---\nBody\n"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			note, err := Parse("note.md", []byte(test.text))
			if err != nil {
				t.Fatal(err)
			}
			for _, edit := range test.edits {
				if err := note.SetProperty(edit[0], edit[1]); err != nil {
					t.Fatal(err)
				}
			}
			if got := string(note.Render()); got != test.want {
				t.Errorf("Render = %q, want %q", got, test.want)
			}
		})
	}
}
