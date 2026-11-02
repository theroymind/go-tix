package vault

import (
	"errors"
	"strings"
	"testing"
)

func TestSectionAndCheckboxes(t *testing.T) {
	tests := []struct {
		name      string
		note      *Note
		heading   string
		wantLines int
		wantBoxes int
		wantDone  int
		wantFirst string
	}{
		{name: "task criteria", note: parseFixture(t, "goal-scoring.md"), heading: "Acceptance criteria", wantLines: 5, wantBoxes: 3, wantDone: 3, wantFirst: "`score_goal` returns"},
		{name: "spec criteria", note: parseFixture(t, "enemy-goals.md"), heading: "Success criteria", wantLines: 11, wantBoxes: 9, wantDone: 0, wantFirst: "A goal is a generic rule"},
		{name: "section ends at a higher heading only", note: parseFixture(t, "enemy-goals.md"), heading: "Shape", wantBoxes: 0},
		{name: "no checkboxes", note: parseFixture(t, "enemy-choice-is-a-weighted-argmax.md"), heading: "Rejected", wantLines: 8, wantBoxes: 0},
		{
			name:    "fenced heading and checkbox ignored",
			note:    mustParse(t, "# A\n```\n# B\n- [ ] fenced\n```\n- [x] real\n* [X] star\n1. [ ] numbered\n  - [ ]\n# B\n- [ ] later\n"),
			heading: "A", wantLines: 8, wantBoxes: 4, wantDone: 2, wantFirst: "real",
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			section, err := test.note.Section(test.heading)
			if err != nil {
				t.Fatal(err)
			}
			if test.wantLines > 0 && section.End-section.Start != test.wantLines {
				t.Errorf("Section = %+v, %d lines, want %d", section, section.End-section.Start, test.wantLines)
			}
			boxes := test.note.Checkboxes(section)
			if len(boxes) != test.wantBoxes {
				t.Fatalf("Checkboxes = %+v, want %d", boxes, test.wantBoxes)
			}
			done := 0
			for _, box := range boxes {
				if box.Done {
					done++
				}
				if box.Line < section.Start || box.Line >= section.End {
					t.Errorf("checkbox line %d outside %+v", box.Line, section)
				}
			}
			if done != test.wantDone {
				t.Errorf("done = %d, want %d", done, test.wantDone)
			}
			if len(boxes) > 0 && !strings.HasPrefix(boxes[0].Text, test.wantFirst) {
				t.Errorf("first text = %q, want prefix %q", boxes[0].Text, test.wantFirst)
			}
		})
	}
}

func mustParse(t *testing.T, text string) *Note {
	t.Helper()
	note, err := Parse("note.md", []byte(text))
	if err != nil {
		t.Fatal(err)
	}
	return note
}

func TestHeading(t *testing.T) {
	tests := []struct {
		name      string
		text      string
		level     int
		wantText  string
		wantFound bool
	}{
		{"first H1", "intro\n## Sub\n# Title\n# Second\n", 1, "Title", true},
		{"fenced heading skipped", "```\n# Fenced\n```\n# Real\n", 1, "Real", true},
		{"closing hashes trimmed", "# Title #\n", 1, "Title", true},
		{"other level", "# Title\n## Sub\n", 2, "Sub", true},
		{"none", "## Sub\ntext\n", 1, "", false},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got, found := mustParse(t, test.text).Heading(test.level)
			if got != test.wantText || found != test.wantFound {
				t.Errorf("Heading(%d) = %q, %v, want %q, %v", test.level, got, found, test.wantText, test.wantFound)
			}
		})
	}
}

func TestSectionNotFound(t *testing.T) {
	tests := []struct {
		name    string
		text    string
		heading string
	}{
		{name: "absent", text: "# A\n", heading: "B"},
		{name: "only in a fence", text: "```\n## B\n```\n", heading: "B"},
		{name: "empty note", text: "", heading: "B"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if _, err := mustParse(t, test.text).Section(test.heading); !errors.Is(err, ErrSectionNotFound) {
				t.Errorf("Section error = %v, want %v", err, ErrSectionNotFound)
			}
		})
	}
}

func TestEditCheckbox(t *testing.T) {
	tests := []struct {
		name     string
		text     []byte
		index    int
		edit     func(note *Note, line int) error
		wantLine string
	}{
		{name: "tick", text: readFixture(t, "enemy-goals.md"), index: 2, edit: func(note *Note, line int) error { return note.SetCheckbox(line, true) }, wantLine: "- [x] Choice is deterministic"},
		{name: "untick", text: readFixture(t, "goal-scoring.md"), index: 0, edit: func(note *Note, line int) error { return note.SetCheckbox(line, false) }, wantLine: "- [ ] `score_goal` returns"},
		{name: "tick crlf", text: toCRLF(readFixture(t, "enemy-goals.md")), index: 0, edit: func(note *Note, line int) error { return note.SetCheckbox(line, true) }, wantLine: "- [x] A goal is a generic rule"},
		{name: "replace text", text: readFixture(t, "goal-scoring.md"), index: 1, edit: func(note *Note, line int) error { return note.SetCheckboxText(line, "New text.") }, wantLine: "- [x] New text."},
		{name: "replace empty text", text: []byte("## Acceptance criteria\n\n- [ ]\n"), index: 0, edit: func(note *Note, line int) error { return note.SetCheckboxText(line, "Filled.") }, wantLine: "- [ ] Filled."},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			note := mustParse(t, string(test.text))
			heading := "Acceptance criteria"
			if _, err := note.Section(heading); err != nil {
				heading = "Success criteria"
			}
			section, err := note.Section(heading)
			if err != nil {
				t.Fatal(err)
			}
			box := note.Checkboxes(section)[test.index]
			if err := test.edit(note, box.Line); err != nil {
				t.Fatal(err)
			}
			changed := lineDiff(t, string(test.text), string(note.Render()), 0)
			if len(changed) != 1 || !strings.HasPrefix(changed[0], test.wantLine) {
				t.Errorf("changed lines = %q, want one starting %q", changed, test.wantLine)
			}
			if strings.HasSuffix(string(test.text), "\r\n") && !strings.HasSuffix(changed[0], "\r") {
				t.Errorf("changed line %q lost its \\r", changed[0])
			}
		})
	}
}

func TestEditCheckboxErrors(t *testing.T) {
	note := mustParse(t, "# A\n- [ ] box\nplain\n")
	tests := []struct {
		name string
		edit func() error
		want error
	}{
		{name: "not a checkbox", edit: func() error { return note.SetCheckbox(2, true) }, want: ErrNotCheckbox},
		{name: "heading", edit: func() error { return note.SetCheckboxText(0, "x") }, want: ErrNotCheckbox},
		{name: "negative", edit: func() error { return note.SetCheckbox(-1, true) }, want: ErrLineOutOfRange},
		{name: "past end", edit: func() error { return note.SetCheckbox(10, true) }, want: ErrLineOutOfRange},
		{name: "multiline", edit: func() error { return note.SetCheckboxText(1, "a\nb") }, want: ErrMultilineText},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if err := test.edit(); !errors.Is(err, test.want) {
				t.Errorf("error = %v, want %v", err, test.want)
			}
		})
	}
}
