package vault

import (
	"fmt"
	"strings"
)

// Range is a half-open span of Body line indexes.
type Range struct {
	Start, End int
}

// Checkbox is a task-list item. Line is its index in Body.
type Checkbox struct {
	Line int
	Text string
	Done bool
}

// Section returns the lines under the first heading whose text is heading, up to the next heading
// of the same or a higher level. Headings inside code fences don't count.
func (note *Note) Section(heading string) (Range, error) {
	level := 0
	section := Range{}
	fence := fenceTracker{}
	for index, line := range note.Body {
		if fence.inCode(line) {
			continue
		}
		lineLevel, text, ok := parseHeading(line)
		if !ok {
			continue
		}
		if level > 0 && lineLevel <= level {
			section.End = index
			return section, nil
		}
		if level == 0 && text == heading {
			level = lineLevel
			section.Start = index + 1
		}
	}
	if level == 0 {
		return Range{}, fmt.Errorf("%s: %w: %q", note.Path, ErrSectionNotFound, heading)
	}
	section.End = len(note.Body)
	return section, nil
}

// Checkboxes returns the task-list items within lines, in order, skipping code fences.
func (note *Note) Checkboxes(lines Range) []Checkbox {
	var boxes []Checkbox
	fence := fenceTracker{}
	end := min(lines.End, len(note.Body))
	for index := max(lines.Start, 0); index < end; index++ {
		if fence.inCode(note.Body[index]) {
			continue
		}
		if parts, ok := parseCheckbox(note.Body[index]); ok {
			boxes = append(boxes, Checkbox{Line: index, Text: parts.text, Done: parts.mark != undoneMark})
		}
	}
	return boxes
}

// SetCheckbox ticks or unticks the checkbox on Body line index line, changing only its mark.
func (note *Note) SetCheckbox(line int, done bool) error {
	parts, err := note.checkboxAt(line)
	if err != nil {
		return err
	}
	parts.mark = undoneMark
	if done {
		parts.mark = doneMark
	}
	note.Body[line] = parts.String()
	return nil
}

// SetCheckboxText replaces the text of the checkbox on Body line index line, keeping its marker
// and mark.
func (note *Note) SetCheckboxText(line int, text string) error {
	if strings.ContainsAny(text, "\r\n") {
		return fmt.Errorf("%s: %w", note.Path, ErrMultilineText)
	}
	parts, err := note.checkboxAt(line)
	if err != nil {
		return err
	}
	if parts.separator == "]" {
		parts.separator = "] "
	}
	parts.text = text
	note.Body[line] = parts.String()
	return nil
}

func (note *Note) checkboxAt(line int) (checkboxParts, error) {
	if line < 0 || line >= len(note.Body) {
		return checkboxParts{}, fmt.Errorf("%s: %w: %d", note.Path, ErrLineOutOfRange, line)
	}
	parts, ok := parseCheckbox(note.Body[line])
	if !ok {
		return checkboxParts{}, fmt.Errorf("%s: line %d: %w", note.Path, line, ErrNotCheckbox)
	}
	return parts, nil
}

// checkboxParts splits "  - [x] text\r" into the pieces that rebuild it exactly.
type checkboxParts struct {
	prefix, mark, separator, text, ending string
}

func (parts checkboxParts) String() string {
	return parts.prefix + parts.mark + parts.separator + parts.text + parts.ending
}

func parseCheckbox(line string) (checkboxParts, bool) {
	body, ending := splitEnding(line)
	match := checkboxPattern.FindStringSubmatch(body)
	if match == nil {
		return checkboxParts{}, false
	}
	return checkboxParts{prefix: match[1], mark: match[2], separator: match[3], text: match[4], ending: ending}, true
}

func parseHeading(line string) (int, string, bool) {
	body, _ := splitEnding(line)
	match := headingPattern.FindStringSubmatch(body)
	if match == nil {
		return 0, "", false
	}
	return len(match[1]), strings.TrimSpace(match[2]), true
}

func splitEnding(line string) (string, string) {
	if body, found := strings.CutSuffix(line, carriageReturn); found {
		return body, carriageReturn
	}
	return line, ""
}

// fenceTracker follows fenced code blocks through a sequence of lines.
type fenceTracker struct {
	open string
}

// inCode reports whether line is a fence delimiter or inside a fenced block.
func (fence *fenceTracker) inCode(line string) bool {
	body, _ := splitEnding(line)
	if fence.open == "" {
		match := fencePattern.FindStringSubmatch(body)
		if match == nil || (match[1][0] == '`' && strings.Contains(body[len(match[0]):], "`")) {
			return false
		}
		fence.open = match[1]
		return true
	}
	indent := len(body) - len(strings.TrimLeft(body, " "))
	trimmed := strings.TrimSpace(body)
	if indent <= maxFenceIndent && strings.HasPrefix(trimmed, fence.open) && strings.Trim(trimmed, fence.open[:1]) == "" {
		fence.open = ""
	}
	return true
}
