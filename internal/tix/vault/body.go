package vault

import (
	"fmt"
	"slices"
	"strconv"
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

// HeadingLine is an ATX heading. Line is its index in Body.
type HeadingLine struct {
	Line  int
	Level int
	Text  string
}

// Headings returns the note's headings in order, skipping code fences.
func (note *Note) Headings() []HeadingLine {
	var headings []HeadingLine
	fence := fenceTracker{}
	for index, line := range note.Body {
		if fence.inCode(line) {
			continue
		}
		if level, text, ok := parseHeading(line); ok {
			headings = append(headings, HeadingLine{Line: index, Level: level, Text: text})
		}
	}
	return headings
}

// Heading returns the text of the first heading of the given level, skipping code fences, and
// whether there is one.
func (note *Note) Heading(level int) (string, bool) {
	for _, heading := range note.Headings() {
		if heading.Level == level {
			return heading.Text, true
		}
	}
	return "", false
}

// SetHeadingText replaces the text of the heading on Body line index line, keeping its level and line ending.
func (note *Note) SetHeadingText(line int, text string) error {
	if strings.ContainsAny(text, "\r\n") {
		return fmt.Errorf("%s: %w", note.Path, ErrMultilineText)
	}
	level, err := note.headingAt(line)
	if err != nil {
		return err
	}
	_, ending := splitEnding(note.Body[line])
	note.Body[line] = headingLine(level, text) + ending
	return nil
}

// InsertHeading inserts a heading of the given level before the body's first text, with a blank line between it
// and the text and between it and any frontmatter. It takes the line ending of the lines around it.
func (note *Note) InsertHeading(level int, text string) error {
	if strings.ContainsAny(text, "\r\n") {
		return fmt.Errorf("%s: %w", note.Path, ErrMultilineText)
	}
	end := len(note.Body)
	if end > 0 && note.Body[end-1] == "" {
		// The empty string a trailing newline leaves is not a line.
		end--
	}
	line := 0
	for line < end && isBlank(note.Body[line]) {
		line++
	}
	ending := note.endingNear(line)
	inserted := []string{headingLine(level, text) + ending}
	if line == 0 && len(note.head) > 0 {
		inserted = append([]string{ending}, inserted...)
	}
	if line < end {
		inserted = append(inserted, ending)
	}
	note.Body = slices.Insert(note.Body, line, inserted...)
	return nil
}

// SetTextUnder replaces the text under the heading on Body line index line, up to the next heading of any level,
// with text, keeping the blank lines around it. Where there is no text, text goes under the heading with a blank
// line on either side. text's trailing newlines are dropped, and each of its lines takes the heading's line ending.
func (note *Note) SetTextUnder(line int, text string) error {
	if _, err := note.headingAt(line); err != nil {
		return err
	}
	start, end := line+1, len(note.Body)
	for _, heading := range note.Headings() {
		if heading.Line > line {
			end = heading.Line
			break
		}
	}
	first, last := start, end
	for first < last && isBlank(note.Body[first]) {
		first++
	}
	for last > first && isBlank(note.Body[last-1]) {
		last--
	}
	_, ending := splitEnding(note.Body[line])
	var lines []string
	for textLine := range strings.SplitSeq(strings.TrimRight(text, "\r\n"), "\n") {
		lines = append(lines, strings.TrimSuffix(textLine, carriageReturn)+ending)
	}
	if first == last {
		first, last = start, start
		lines = append([]string{ending}, lines...)
		if start < len(note.Body) && !isBlank(note.Body[start]) {
			lines = append(lines, ending)
		}
	}
	note.Body = slices.Replace(note.Body, first, last, lines...)
	return nil
}

// headingAt returns the level of the heading on Body line index line.
func (note *Note) headingAt(line int) (int, error) {
	if line < 0 || line >= len(note.Body) {
		return 0, fmt.Errorf("%s: %w: %d", note.Path, ErrLineOutOfRange, line)
	}
	level, _, ok := parseHeading(note.Body[line])
	if !ok {
		return 0, fmt.Errorf("%s: line %d: %w", note.Path, line, ErrNotHeading)
	}
	return level, nil
}

// endingNear returns the line ending of the Body line at index line or the one before it, or of the frontmatter
// when neither is a line: "\r" for CRLF, else "".
func (note *Note) endingNear(line int) string {
	for _, index := range []int{line, line - 1} {
		if index >= 0 && index < len(note.Body) && note.Body[index] != "" {
			_, ending := splitEnding(note.Body[index])
			return ending
		}
	}
	if usesCarriageReturns(note.head) {
		return carriageReturn
	}
	return ""
}

func headingLine(level int, text string) string {
	return strings.Repeat(headingMark, level) + " " + text
}

func isBlank(line string) bool {
	return strings.TrimSpace(line) == ""
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

// Numbered returns a copy of the note whose boxes carry their 1-based position as the list marker, so "- [ ] text"
// reads "1. [ ] text" and a reader can name a checkbox by number. The note itself is unchanged.
func (note *Note) Numbered(boxes []Checkbox) *Note {
	numbered := *note
	numbered.Body = slices.Clone(note.Body)
	for index, box := range boxes {
		parts, ok := parseCheckbox(numbered.Body[box.Line])
		if !ok {
			continue
		}
		match := listMarkerPattern.FindStringSubmatch(parts.prefix)
		if match == nil {
			continue
		}
		parts.prefix = match[1] + strconv.Itoa(index+1) + numberedMarker + match[2]
		numbered.Body[box.Line] = parts.String()
	}
	return &numbered
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
