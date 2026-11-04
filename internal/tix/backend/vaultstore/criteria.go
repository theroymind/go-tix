package vaultstore

import (
	"slices"
	"strings"

	"github.com/theroymind/go-tix/contracts/ticket"
	"github.com/theroymind/go-tix/internal/tix/backend"
	"github.com/theroymind/go-tix/internal/tix/vault"
)

// newCheckbox is the line an added criterion starts from when the section has no checkbox to copy.
const newCheckbox = "- [ ] "

// applyCriterionChange makes one criterion change to the note's checkboxes under the profile's criteria heading.
func (store Store) applyCriterionChange(note *vault.Note, change backend.Change) error {
	section, boxes, err := criterionBoxes(store.Profile, note)
	if err != nil {
		return err
	}
	switch change := change.(type) {
	case backend.CheckCriterion:
		return withCriterion(note, boxes, change.Index, func(line int) error { return note.SetCheckbox(line, true) })
	case backend.UncheckCriterion:
		return withCriterion(note, boxes, change.Index, func(line int) error { return note.SetCheckbox(line, false) })
	case backend.SetCriterion:
		return withCriterion(note, boxes, change.Index, func(line int) error { return note.SetCheckboxText(line, change.Text) })
	case backend.AddCriterion:
		return addCriterion(note, section, boxes, change.Text)
	default:
		return backend.Unsupported(change, backend.KindVault)
	}
}

// setCriteria replaces the checkboxes under the profile's criteria heading with one unticked checkbox per criterion.
// The first checkbox's line is kept, so the criteria keep its marker and indent.
func setCriteria(profile backend.Profile, note *vault.Note, criteria []ticket.Criterion) error {
	section, boxes, err := criterionBoxes(profile, note)
	if err != nil {
		return err
	}
	if len(boxes) > 0 {
		// Every item's lines are found before any is deleted, then deleted last to first so that the earlier ones
		// stay where they were. The first item keeps its own line and loses only its wrapped lines.
		items := make([]vault.Range, len(boxes))
		for index, box := range boxes {
			items[index] = vault.Range{Start: box.Line + 1, End: itemEnd(note.Body, box.Line, section.End)}
			if index > 0 {
				items[index].Start = box.Line
			}
		}
		for index := len(items) - 1; index >= 0; index-- {
			note.Body = slices.Delete(note.Body, items[index].Start, items[index].End)
		}
		if err := note.SetCheckbox(boxes[0].Line, false); err != nil {
			return err
		}
		if err := note.SetCheckboxText(boxes[0].Line, criteria[0].Text); err != nil {
			return err
		}
		criteria = criteria[1:]
	}
	for _, criterion := range criteria {
		section, boxes, err := criterionBoxes(profile, note)
		if err != nil {
			return err
		}
		if err := addCriterion(note, section, boxes, criterion.Text); err != nil {
			return err
		}
	}
	return nil
}

// withCriterion runs edit on the Body line of the 1-based index-th criterion.
func withCriterion(note *vault.Note, boxes []vault.Checkbox, index int, edit func(line int) error) error {
	if index < 1 || index > len(boxes) {
		return criterionOutOfRange(note.Path, index, len(boxes))
	}
	return edit(boxes[index-1].Line)
}

// addCriterion inserts an unticked checkbox reading text as the section's last checkbox. It goes directly after the
// last checkbox's item, its wrapped lines included, copying the checkbox's line so the marker, indent and line ending
// match. In a section with none, it goes after the heading and any blank line below it, before the empty line a
// trailing newline leaves, with a blank line kept between it and any text that follows.
func addCriterion(note *vault.Note, section vault.Range, boxes []vault.Checkbox, text string) error {
	if len(boxes) > 0 {
		last := boxes[len(boxes)-1].Line
		line := itemEnd(note.Body, last, section.End)
		note.Body = slices.Insert(note.Body, line, note.Body[last])
		if err := note.SetCheckbox(line, false); err != nil {
			return err
		}
		return note.SetCheckboxText(line, text)
	}
	line := section.Start
	if line < section.End && line+1 < len(note.Body) && isBlank(note.Body[line]) {
		line++
	}
	ending := lineEnding(note.Body[section.Start-1])
	inserted := []string{newCheckbox + ending}
	if line < len(note.Body) && !isBlank(note.Body[line]) {
		inserted = append(inserted, ending)
	}
	note.Body = slices.Insert(note.Body, line, inserted...)
	return note.SetCheckboxText(line, text)
}

// itemEnd returns the Body index just past the list item that starts on line start: past the lines that wrap onto
// it, which are indented deeper than its own line, up to a blank line or end.
func itemEnd(body []string, start, end int) int {
	depth := indent(body[start])
	line := start + 1
	for line < end && !isBlank(body[line]) && indent(body[line]) > depth {
		line++
	}
	return line
}

func indent(line string) int {
	return len(line) - len(strings.TrimLeft(line, " \t"))
}

func isBlank(line string) bool {
	return strings.TrimSpace(line) == ""
}

// lineEnding returns the "\r" a CRLF line keeps in Body, or "".
func lineEnding(line string) string {
	if strings.HasSuffix(line, "\r") {
		return "\r"
	}
	return ""
}
