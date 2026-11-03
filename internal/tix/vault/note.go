// Package vault reads and edits Obsidian notes: YAML frontmatter, sections, checkboxes and
// wikilinks. It knows nothing about tickets, and rendering a note changes only the bytes an edit
// touched.
package vault

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"go.yaml.in/yaml/v3"
)

var (
	ErrUnterminatedFrontmatter = errors.New("frontmatter has no closing ---")
	ErrInvalidFrontmatter      = errors.New("invalid frontmatter")
	ErrFrontmatterNotMapping   = errors.New("frontmatter is not a mapping")
	ErrDuplicateKey            = errors.New("duplicate frontmatter key")
	ErrPropertyNotFound        = errors.New("frontmatter has no property")
	ErrSectionNotFound         = errors.New("section not found")
	ErrNotCheckbox             = errors.New("line is not a checkbox")
	ErrNotHeading              = errors.New("line is not a heading")
	ErrLineOutOfRange          = errors.New("line out of range")
	ErrMultilineText           = errors.New("text spans more than one line")
)

// Note is one Markdown file. Body holds the lines after the frontmatter, split on "\n" with any
// "\r" kept, so changing a line in place changes only that line when rendered.
//
// Frontmatter is the parsed YAML document, or nil when the note has none. Read it freely, but
// change it only through SetProperty: Render writes the original frontmatter text, not the node.
type Note struct {
	Path        string
	Slug        string
	Frontmatter *yaml.Node
	Body        []string

	// head is the frontmatter's raw lines, delimiters included; empty when the note has none.
	head []string
}

// Read parses the note at path.
func Read(path string) (*Note, error) {
	text, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	return Parse(path, text)
}

// Parse splits text into frontmatter and body. The slug is the file name without ".md".
func Parse(path string, text []byte) (*Note, error) {
	note := &Note{Path: path, Slug: SlugOf(path)}
	lines := strings.Split(string(text), "\n")
	end := closingDelimiter(lines)
	if end < 0 {
		note.Body = lines
		return note, nil
	}
	if end == len(lines) {
		return nil, fmt.Errorf("%s: %w", path, ErrUnterminatedFrontmatter)
	}
	if err := note.setHead(lines[:end+1]); err != nil {
		return nil, fmt.Errorf("%s: %w", path, err)
	}
	note.Body = lines[end+1:]
	return note, nil
}

// SlugOf returns the slug Obsidian links a file by: its base name without ".md".
func SlugOf(path string) string {
	return strings.TrimSuffix(filepath.Base(path), noteExtension)
}

// closingDelimiter returns the index of the line closing the frontmatter, -1 when the note has no
// frontmatter, or len(lines) when it opens one but never closes it.
func closingDelimiter(lines []string) int {
	if len(lines) < 2 || !isDelimiter(lines[0]) {
		return -1
	}
	for index := 1; index < len(lines); index++ {
		if isDelimiter(lines[index]) {
			return index
		}
	}
	return len(lines)
}

func isDelimiter(line string) bool {
	return strings.TrimSuffix(line, carriageReturn) == frontmatterDelimiter
}

// Render returns the note's text: byte for byte what Parse read, plus any edits.
func (note *Note) Render() []byte {
	return []byte(strings.Join(append(append([]string{}, note.head...), note.Body...), "\n"))
}

// Property returns the value node of a top-level frontmatter key, or nil when it is absent.
func (note *Note) Property(key string) *yaml.Node {
	mapping := mappingRoot(note.Frontmatter)
	if index := valueIndex(mapping, key); index >= 0 {
		return mapping.Content[index]
	}
	return nil
}

// SetProperty sets a top-level frontmatter key to value, rewriting only that key's lines, or
// appending the key when it is absent. A replaced value keeps the old one's quote style or flow
// style, and its line comment.
func (note *Note) SetProperty(key string, value any) error {
	var replacement yaml.Node
	if err := replacement.Encode(value); err != nil {
		return fmt.Errorf("%s: %w: %w", note.Path, ErrInvalidFrontmatter, err)
	}
	if len(note.head) == 0 {
		note.head = []string{frontmatterDelimiter, frontmatterDelimiter}
		note.Frontmatter = nil
	}
	mapping := mappingRoot(note.Frontmatter)
	if note.Frontmatter != nil && note.Frontmatter.Kind != 0 && mapping == nil {
		return fmt.Errorf("%s: %w", note.Path, ErrFrontmatterNotMapping)
	}
	content := note.head[1 : len(note.head)-1]
	first, last := len(content), len(content)
	keyNode := &yaml.Node{Kind: yaml.ScalarNode, Tag: "!!str", Value: key}
	if index := valueIndex(mapping, key); index >= 0 {
		keyNode, first, last = entryLines(mapping, index, content)
		keepStyle(mapping.Content[index], &replacement)
	}
	lines, err := encodeEntry(keyNode, &replacement)
	if err != nil {
		return fmt.Errorf("%s: %w: %w", note.Path, ErrInvalidFrontmatter, err)
	}
	if usesCarriageReturns(note.head) {
		for index := range lines {
			lines[index] += carriageReturn
		}
	}
	edited := make([]string, 0, len(note.head)+len(lines))
	edited = append(edited, note.head[0])
	edited = append(edited, content[:first]...)
	edited = append(edited, lines...)
	edited = append(edited, content[last:]...)
	edited = append(edited, note.head[len(note.head)-1])
	if err := note.setHead(edited); err != nil {
		return fmt.Errorf("%s: %w", note.Path, err)
	}
	return nil
}

// RemoveProperty deletes key's entry from the frontmatter, every line its value spans, and leaves the rest of the
// note as it was. A key the frontmatter doesn't hold fails with ErrPropertyNotFound.
func (note *Note) RemoveProperty(key string) error {
	mapping := mappingRoot(note.Frontmatter)
	index := valueIndex(mapping, key)
	if index < 0 {
		return fmt.Errorf("%s: %w: %s", note.Path, ErrPropertyNotFound, key)
	}
	content := note.head[1 : len(note.head)-1]
	_, first, last := entryLines(mapping, index, content)
	edited := make([]string, 0, len(note.head))
	edited = append(edited, note.head[0])
	edited = append(edited, content[:first]...)
	edited = append(edited, content[last:]...)
	edited = append(edited, note.head[len(note.head)-1])
	if err := note.setHead(edited); err != nil {
		return fmt.Errorf("%s: %w", note.Path, err)
	}
	return nil
}

// setHead stores the frontmatter's raw lines and parses them into Frontmatter.
func (note *Note) setHead(head []string) error {
	text := strings.Join(head[1:len(head)-1], "\n")
	decoder := yaml.NewDecoder(strings.NewReader(text))
	var document yaml.Node
	if err := decoder.Decode(&document); err != nil && !errors.Is(err, io.EOF) {
		return fmt.Errorf("%w: %w", ErrInvalidFrontmatter, err)
	}
	if mapping := mappingRoot(&document); mapping != nil {
		if err := checkDuplicateKeys(mapping); err != nil {
			return err
		}
	} else if document.Kind != 0 {
		return ErrFrontmatterNotMapping
	}
	note.head = head
	note.Frontmatter = &document
	return nil
}

// entryLines returns the key node at index-1 and the half-open range of content lines its entry
// spans: from the key's line to the next key's, less any trailing blank or comment lines, which
// belong to what follows.
func entryLines(mapping *yaml.Node, index int, content []string) (*yaml.Node, int, int) {
	key := mapping.Content[index-1]
	first, last := key.Line-1, len(content)
	if index+1 < len(mapping.Content) {
		last = mapping.Content[index+1].Line - 1
	}
	for last > first+1 {
		trimmed := strings.TrimSpace(content[last-1])
		if trimmed != "" && !strings.HasPrefix(trimmed, "#") {
			break
		}
		last--
	}
	copied := *key
	copied.HeadComment, copied.FootComment = "", ""
	return &copied, first, last
}

// keepStyle carries the old value's quoting, flow style and line comment onto its replacement.
func keepStyle(old, replacement *yaml.Node) {
	switch {
	case old.Kind == yaml.ScalarNode && replacement.Kind == yaml.ScalarNode && replacement.Tag == stringTag && keepsPlainTag(old, replacement):
		replacement.Tag, replacement.Style = old.ShortTag(), 0
	case old.Kind == yaml.ScalarNode && replacement.Kind == yaml.ScalarNode && replacement.Tag == stringTag:
		replacement.Style = old.Style & (yaml.DoubleQuotedStyle | yaml.SingleQuotedStyle)
	case old.Kind == replacement.Kind && old.Kind != yaml.ScalarNode:
		replacement.Style = old.Style & yaml.FlowStyle
	}
	replacement.LineComment = old.LineComment
}

// keepsPlainTag reports whether the old value is a plain scalar of a type other than string, such as an int or a
// date, and the replacement text read unquoted is a value of that same type, so it can stay plain.
func keepsPlainTag(old, replacement *yaml.Node) bool {
	if old.Style&(yaml.DoubleQuotedStyle|yaml.SingleQuotedStyle|yaml.LiteralStyle|yaml.FoldedStyle) != 0 || old.ShortTag() == stringTag {
		return false
	}
	unquoted := &yaml.Node{Kind: yaml.ScalarNode, Value: replacement.Value}
	return unquoted.ShortTag() == old.ShortTag()
}

func encodeEntry(key, value *yaml.Node) ([]string, error) {
	var buffer bytes.Buffer
	encoder := yaml.NewEncoder(&buffer)
	encoder.SetIndent(yamlIndent)
	entry := &yaml.Node{Kind: yaml.MappingNode, Tag: "!!map", Content: []*yaml.Node{key, value}}
	if err := encoder.Encode(entry); err != nil {
		return nil, err
	}
	if err := encoder.Close(); err != nil {
		return nil, err
	}
	return strings.Split(strings.TrimSuffix(buffer.String(), "\n"), "\n"), nil
}

func usesCarriageReturns(lines []string) bool {
	return len(lines) > 0 && strings.HasSuffix(lines[0], carriageReturn)
}

func mappingRoot(document *yaml.Node) *yaml.Node {
	if document == nil || document.Kind != yaml.DocumentNode || len(document.Content) != 1 || document.Content[0].Kind != yaml.MappingNode {
		return nil
	}
	return document.Content[0]
}

func valueIndex(mapping *yaml.Node, key string) int {
	if mapping == nil {
		return -1
	}
	for index := 0; index+1 < len(mapping.Content); index += 2 {
		if mapping.Content[index].Value == key {
			return index + 1
		}
	}
	return -1
}

func checkDuplicateKeys(mapping *yaml.Node) error {
	lines := map[string]int{}
	for index := 0; index+1 < len(mapping.Content); index += 2 {
		key := mapping.Content[index]
		if firstLine, exists := lines[key.Value]; exists {
			return fmt.Errorf("line %d: %w %q, first defined at line %d", key.Line, ErrDuplicateKey, key.Value, firstLine)
		}
		lines[key.Value] = key.Line
	}
	return nil
}
