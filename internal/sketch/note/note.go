// Package note reads .sketch.md notes: it parses one into a sketch.Note, maps a note's path to the file it
// plans, and finds every note under a repo root.
package note

import (
	"errors"
	"fmt"
	"os"
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

// frontMatter holds the keys a note's front matter links it with. Any other key is ignored, since notes are
// written by hand.
type frontMatter struct {
	Feature string `yaml:"feature"`
	Task    string `yaml:"task"`
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
// the feature and task. Below it, a line #TYPE NAME REST, where TYPE is a sketch item type, names an item; a line
// starting ? asks a question, and a line starting > right under it answers it. Every other line is free text and
// is skipped. Only front matter with no closing --- line, or that isn't YAML, fails, with ErrFrontMatter.
func Parse(notePath, text string) (sketch.Note, error) {
	target, err := Target(notePath)
	if err != nil {
		return sketch.Note{}, err
	}
	value := sketch.Note{SchemaVersion: sketch.SchemaVersion, Path: notePath, Target: target}
	lines := strings.Split(text, "\n")
	for index := range lines {
		lines[index] = strings.TrimRightFunc(lines[index], unicode.IsSpace)
	}
	body, err := readFrontMatter(lines, &value)
	if err != nil {
		return sketch.Note{}, fmt.Errorf("%s: %w", notePath, err)
	}
	for index := body; index < len(lines); index++ {
		line := lines[index]
		if item, isItem := parseItem(line); isItem {
			value.Items = append(value.Items, item)
			continue
		}
		questionText, isQuestion := strings.CutPrefix(line, questionMarker)
		questionText = strings.TrimSpace(questionText)
		if !isQuestion || questionText == "" {
			continue
		}
		question := sketch.Question{Text: questionText, Line: index + 1}
		if index+1 < len(lines) {
			if answer, isAnswer := parseAnswer(lines[index+1]); isAnswer {
				question.Answer = answer
				index++
			}
		}
		value.Questions = append(value.Questions, question)
	}
	return value, nil
}

// readFrontMatter fills value's feature and task from the front matter at the top of lines, when there is any,
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
		value.Feature, value.Task = strings.TrimSpace(fields.Feature), strings.TrimSpace(fields.Task)
		return index + 1, nil
	}
	return 0, fmt.Errorf("%w: no closing %s line", ErrFrontMatter, frontMatterFence)
}

// SetFeature returns text with its front matter linking it to feature: the feature line replaced when there is
// one, added to the front matter when there isn't, or new front matter put on top when the note has none. Front
// matter with no closing --- line fails with ErrFrontMatter.
func SetFeature(text, feature string) (string, error) {
	featureLine := featureKey + " " + feature
	lines := strings.Split(text, "\n")
	if strings.TrimRightFunc(lines[0], unicode.IsSpace) != frontMatterFence {
		return frontMatterFence + "\n" + featureLine + "\n" + frontMatterFence + "\n" + text, nil
	}
	for index := 1; index < len(lines); index++ {
		line := strings.TrimRightFunc(lines[index], unicode.IsSpace)
		if line == frontMatterFence {
			lines = slices.Insert(lines, 1, featureLine)
			return strings.Join(lines, "\n"), nil
		}
		if strings.HasPrefix(line, featureKey) {
			lines[index] = featureLine
			return strings.Join(lines, "\n"), nil
		}
	}
	return "", fmt.Errorf("%w: no closing %s line", ErrFrontMatter, frontMatterFence)
}

// parseItem reads a line #TYPE NAME REST as an item. A #word that isn't an item type, such as an Obsidian tag,
// a # heading, and an item type with no name are free text.
func parseItem(line string) (sketch.Item, bool) {
	marked, isMarked := strings.CutPrefix(line, itemMarker)
	if !isMarked {
		return sketch.Item{}, false
	}
	itemType, afterType := cutWord(marked)
	name, rest := cutWord(afterType)
	if !sketch.ItemType(itemType).IsValid() || name == "" {
		return sketch.Item{}, false
	}
	return sketch.Item{Type: sketch.ItemType(itemType), Name: name, Text: rest}, true
}

// parseAnswer reads a line starting > as an answer, the text after it trimmed.
func parseAnswer(line string) (string, bool) {
	answer, isAnswer := strings.CutPrefix(line, answerMarker)
	return strings.TrimSpace(answer), isAnswer
}

// cutWord splits text at its first run of spaces, returning the word before it and the text after, trimmed.
// text that starts with a space has an empty first word.
func cutWord(text string) (string, string) {
	end := strings.IndexFunc(text, unicode.IsSpace)
	if end < 0 {
		return text, ""
	}
	return text[:end], strings.TrimSpace(text[end:])
}
