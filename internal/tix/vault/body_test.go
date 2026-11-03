package vault

import (
	"errors"
	"slices"
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

func TestHeadings(t *testing.T) {
	tests := []struct {
		name     string
		text     string
		expected []HeadingLine
	}{
		{"every level in order", "intro\n# Title\ntext\n## Sub #\n",
			[]HeadingLine{{Line: 1, Level: 1, Text: "Title"}, {Line: 3, Level: 2, Text: "Sub"}}},
		{"fenced headings skipped", "```\n# Fenced\n```\n### Real\r\n",
			[]HeadingLine{{Line: 3, Level: 3, Text: "Real"}}},
		{"none", "text\n", nil},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if got := mustParse(t, test.text).Headings(); !slices.Equal(got, test.expected) {
				t.Errorf("Headings() = %v, want %v", got, test.expected)
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

func TestNumbered(t *testing.T) {
	tests := []struct {
		name      string
		text      string
		wantLines []string
	}{
		{"Bullets become numbers", "## Acceptance criteria\n\n- [ ] One\n- [x] Two\n", []string{"1. [ ] One", "2. [x] Two"}},
		{"Indent and numbered markers are kept", "## Acceptance criteria\n\n  * [ ] One\n  3) [ ] Two\n", []string{"  1. [ ] One", "  2. [ ] Two"}},
		{"A CRLF line keeps its ending", "## Acceptance criteria\r\n\r\n- [ ] One\r\n", []string{"1. [ ] One\r"}},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			note := mustParse(t, test.text)
			section, err := note.Section("Acceptance criteria")
			if err != nil {
				t.Fatal(err)
			}
			boxes := note.Checkboxes(section)

			numbered := note.Numbered(boxes)

			var lines []string
			for _, box := range boxes {
				lines = append(lines, numbered.Body[box.Line])
			}
			if !slices.Equal(lines, test.wantLines) {
				t.Errorf("numbered lines = %q, want %q", lines, test.wantLines)
			}
			if string(note.Render()) != test.text {
				t.Errorf("the note itself changed: %q", note.Render())
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

func TestHeadingEdits(t *testing.T) {
	tests := []struct {
		name     string
		text     string
		edit     func(note *Note) error
		expected string
	}{
		{"SetHeadingText keeps the level and line ending", "# Title\r\ntext\r\n",
			func(note *Note) error { return note.SetHeadingText(0, "New") }, "# New\r\ntext\r\n"},
		{"SetHeadingText on a deeper heading", "## Sub ##\n",
			func(note *Note) error { return note.SetHeadingText(0, "New") }, "## New\n"},
		{"InsertHeading goes before the first text, set off by blank lines", "---\na: 1\n---\n\nText.\n",
			func(note *Note) error { return note.InsertHeading(1, "New") }, "---\na: 1\n---\n\n# New\n\nText.\n"},
		{"InsertHeading under frontmatter alone", "---\na: 1\n---\n",
			func(note *Note) error { return note.InsertHeading(1, "New") }, "---\na: 1\n---\n\n# New\n"},
		{"InsertHeading keeps CRLF from the frontmatter", "---\r\na: 1\r\n---\r\n",
			func(note *Note) error { return note.InsertHeading(1, "New") }, "---\r\na: 1\r\n---\r\n\r\n# New\r\n"},
		{"InsertHeading keeps CRLF from the text", "---\r\na: 1\r\n---\r\n\r\nText.\r\n",
			func(note *Note) error { return note.InsertHeading(1, "New") }, "---\r\na: 1\r\n---\r\n\r\n# New\r\n\r\nText.\r\n"},
		{"InsertHeading at the top of a note without frontmatter", "Text.\n",
			func(note *Note) error { return note.InsertHeading(1, "New") }, "# New\n\nText.\n"},
		{"SetTextUnder replaces the text up to the next heading", "# T\n\nOld\nlines.\n\n## Next\n",
			func(note *Note) error { return note.SetTextUnder(0, "New.\nText.") }, "# T\n\nNew.\nText.\n\n## Next\n"},
		{"SetTextUnder up to the end", "# T\n\nOld.\n",
			func(note *Note) error { return note.SetTextUnder(0, "New.") }, "# T\n\nNew.\n"},
		{"SetTextUnder adds text where there is none", "# T\n## Next\n",
			func(note *Note) error { return note.SetTextUnder(0, "New.") }, "# T\n\nNew.\n\n## Next\n"},
		{"SetTextUnder drops the text's trailing newlines and keeps CRLF", "# T\r\n\r\nOld.\r\n\r\n## Next\r\n",
			func(note *Note) error { return note.SetTextUnder(0, "New.\r\nText.\r\n\n") }, "# T\r\n\r\nNew.\r\nText.\r\n\r\n## Next\r\n"},
		{"SetTextUnder skips a heading in a fence", "# T\n\nOld.\n```\n# x\n```\n\n## Next\n",
			func(note *Note) error { return note.SetTextUnder(0, "New.") }, "# T\n\nNew.\n\n## Next\n"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			note := mustParse(t, test.text)
			if err := test.edit(note); err != nil {
				t.Fatal(err)
			}
			if got := string(note.Render()); got != test.expected {
				t.Errorf("Render() = %q, want %q", got, test.expected)
			}
		})
	}
}

func TestHeadingEditErrors(t *testing.T) {
	note := mustParse(t, "# A\nplain\n")
	tests := []struct {
		name string
		edit func() error
		want error
	}{
		{name: "not a heading", edit: func() error { return note.SetHeadingText(1, "x") }, want: ErrNotHeading},
		{name: "text under a non-heading", edit: func() error { return note.SetTextUnder(1, "x") }, want: ErrNotHeading},
		{name: "past end", edit: func() error { return note.SetHeadingText(10, "x") }, want: ErrLineOutOfRange},
		{name: "negative", edit: func() error { return note.SetTextUnder(-1, "x") }, want: ErrLineOutOfRange},
		{name: "multiline heading", edit: func() error { return note.SetHeadingText(0, "a\nb") }, want: ErrMultilineText},
		{name: "multiline inserted heading", edit: func() error { return note.InsertHeading(1, "a\nb") }, want: ErrMultilineText},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if err := test.edit(); !errors.Is(err, test.want) {
				t.Errorf("error = %v, want %v", err, test.want)
			}
		})
	}
}
