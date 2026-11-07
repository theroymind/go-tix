// Package note reads .sketch.md notes: it parses one into a sketch.Note, maps a note's path to the file it
// plans, and finds every note under a repo root.
package note

import (
	"errors"
	"fmt"
	"os"
	"path"
	"path/filepath"
	"slices"
	"strings"
	"unicode"

	"go.yaml.in/yaml/v3"

	"github.com/theroymind/go-tix/contracts/sketch"
)

var (
	ErrNotANote    = errors.New("isn't a note: its name doesn't end in " + sketch.NoteSuffix)
	ErrFrontMatter = errors.New("bad front matter")
)

// frontMatter holds the keys sketch reads from a note's front matter. Any other key is ignored, since notes are
// written by hand.
type frontMatter struct {
	Feature string `yaml:"feature"`
	Status  string `yaml:"status"`
}

// Target returns the path of the file the note at notePath plans, in the same folder: a/b/useOrbit.ts.sketch.md
// plans a/b/useOrbit.ts. A path that doesn't end in sketch.NoteSuffix fails with ErrNotANote.
func Target(notePath string) (string, error) {
	target, isNote := strings.CutSuffix(notePath, sketch.NoteSuffix)
	if !isNote {
		return "", fmt.Errorf("%q %w", notePath, ErrNotANote)
	}
	return target, nil
}

// Load reads and parses the note at notePath, a slash-separated path from root.
func Load(root, notePath string) (sketch.Note, error) {
	text, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(notePath)))
	if err != nil {
		return sketch.Note{}, err
	}
	return Parse(notePath, string(text))
}

// Parse reads text as the note at notePath. Optional YAML front matter between --- lines at the very top gives
// the feature and status, and everything below it is the body, kept as written. In the body, a line -> PATH or
// -> PATH : LABEL links to another file, and a line starting ? asks a question, answered by the lines starting >
// right under it, or under one blank line below it. Every other line is free text. Only front matter with no
// closing --- line, or that isn't YAML, fails, with ErrFrontMatter.
func Parse(notePath, text string) (sketch.Note, error) {
	target, err := Target(notePath)
	if err != nil {
		return sketch.Note{}, err
	}
	value := sketch.Note{SchemaVersion: sketch.SchemaVersion, Path: notePath, Target: target}
	rawLines := strings.Split(text, "\n")
	lines := make([]string, len(rawLines))
	for index, line := range rawLines {
		lines[index] = strings.TrimRightFunc(line, unicode.IsSpace)
	}
	bodyStart, err := readFrontMatter(lines, &value)
	if err != nil {
		return sketch.Note{}, fmt.Errorf("%s: %w", notePath, err)
	}
	value.Body = strings.Join(rawLines[bodyStart:], "\n")
	noteFolder := path.Dir(notePath)
	for index := bodyStart; index < len(lines); index++ {
		line := lines[index]
		if edge, isEdge := parseEdge(line, noteFolder); isEdge {
			value.Edges = append(value.Edges, edge)
			continue
		}
		questionText, isQuestion := strings.CutPrefix(line, questionMarker)
		questionText = strings.TrimSpace(questionText)
		if !isQuestion || questionText == "" {
			continue
		}
		question := sketch.Question{Text: questionText, Line: index + 1}
		question.Answer, index = readAnswer(lines, index+1)
		value.Questions = append(value.Questions, question)
	}
	return value, nil
}

// readFrontMatter fills value's feature and status from the front matter at the top of lines, when there is any,
// and returns the index of the first line below it.
func readFrontMatter(lines []string, value *sketch.Note) (int, error) {
	if lines[0] != frontMatterFence {
		return 0, nil
	}
	for index := 1; index < len(lines); index++ {
		if lines[index] != frontMatterFence {
			continue
		}
		var fields frontMatter
		if err := yaml.Unmarshal([]byte(strings.Join(lines[1:index], "\n")), &fields); err != nil {
			return 0, fmt.Errorf("%w: %w", ErrFrontMatter, err)
		}
		value.Feature = strings.TrimSpace(fields.Feature)
		value.Status = sketch.Status(strings.TrimSpace(fields.Status))
		return index + 1, nil
	}
	return 0, fmt.Errorf("%w: no closing %s line", ErrFrontMatter, frontMatterFence)
}

// parseEdge reads a line -> PATH or -> PATH : LABEL as an edge. PATH is from the repo root, or from noteFolder when
// it starts with ./ or ../, and is stored clean and from the repo root. A line with no path, or whose path leaves
// the repo, is free text.
func parseEdge(line, noteFolder string) (sketch.Edge, bool) {
	link, isLink := strings.CutPrefix(line, edgeMarker)
	if !isLink {
		return sketch.Edge{}, false
	}
	linkPath, label, _ := strings.Cut(link, labelSeparator)
	linkPath = strings.TrimSpace(linkPath)
	if strings.HasPrefix(linkPath, currentFolderPrefix) || strings.HasPrefix(linkPath, parentFolderPrefix) {
		linkPath = path.Join(noteFolder, linkPath)
	}
	linkPath = path.Clean(linkPath)
	if linkPath == "." || linkPath == parentFolder || strings.HasPrefix(linkPath, parentFolderPrefix) ||
		path.IsAbs(linkPath) {
		return sketch.Edge{}, false
	}
	return sketch.Edge{To: linkPath, Label: strings.TrimSpace(label)}, true
}

// readAnswer reads the answer starting at lines[start]: the consecutive lines starting >, which may follow one
// blank line, each with the > and surrounding spaces cut, joined with newlines. It returns the answer and the
// index of its last line, or start-1 when there's no answer, so the blank line stays free text.
func readAnswer(lines []string, start int) (string, int) {
	first := start
	if first < len(lines) && lines[first] == "" {
		first++
	}
	var answer []string
	index := first
	for ; index < len(lines); index++ {
		text, isAnswer := strings.CutPrefix(lines[index], answerMarker)
		if !isAnswer {
			break
		}
		answer = append(answer, strings.TrimSpace(text))
	}
	if len(answer) == 0 {
		return "", start - 1
	}
	return strings.TrimSpace(strings.Join(answer, "\n")), index - 1
}

// SetField returns text with its front matter giving the key name, such as FeatureKey or StatusKey, the value:
// the key's line replaced when there is one, added to the front matter when there isn't, or new front matter put
// on top when the note has none. Front matter with no closing --- line fails with ErrFrontMatter.
func SetField(text, name, value string) (string, error) {
	key := name + ":"
	fieldLine := key + " " + value
	lines := strings.Split(text, "\n")
	if strings.TrimRightFunc(lines[0], unicode.IsSpace) != frontMatterFence {
		return frontMatterFence + "\n" + fieldLine + "\n" + frontMatterFence + "\n" + text, nil
	}
	for index := 1; index < len(lines); index++ {
		line := strings.TrimRightFunc(lines[index], unicode.IsSpace)
		if line == frontMatterFence {
			lines = slices.Insert(lines, 1, fieldLine)
			return strings.Join(lines, "\n"), nil
		}
		if strings.HasPrefix(line, key) {
			lines[index] = fieldLine
			return strings.Join(lines, "\n"), nil
		}
	}
	return "", fmt.Errorf("%w: no closing %s line", ErrFrontMatter, frontMatterFence)
}
