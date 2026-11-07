// Package sketch provides the Note contract: a plain .sketch.md note beside the file it plans, read as the
// feature and task it links to, the items it names and the questions it asks.
package sketch

import (
	"fmt"
	"path"
	"slices"
	"strings"
	"unicode"

	"github.com/theroymind/go-tix/contracts/validation"
)

// Note is one note: its slash-separated path from the repo root, the path of the file it plans, the feature and
// task its front matter links it to, and its items and questions in the order they appear. Mark says whether the
// target exists yet, and UnknownFeature that the repo is a tix repo with no ticket named Feature; a note read
// straight from its file has neither set.
type Note struct {
	SchemaVersion  int        `json:"schema_version"`
	Path           string     `json:"path"`
	Target         string     `json:"target"`
	Mark           Mark       `json:"mark,omitempty"`
	Feature        string     `json:"feature,omitempty"`
	UnknownFeature bool       `json:"unknown_feature,omitempty"`
	Task           string     `json:"task,omitempty"`
	Items          []Item     `json:"items,omitempty"`
	Questions      []Question `json:"questions,omitempty"`
}

// Item is one named piece the note plans, from a line such as #fn useOrbit target Ref<Vec3>: its type, its name,
// the rest of the line as Text, and a Mark saying whether the target already holds the name.
type Item struct {
	Type ItemType `json:"type"`
	Name string   `json:"name"`
	Text string   `json:"text,omitempty"`
	Mark Mark     `json:"mark,omitempty"`
}

// Question is one question the note asks, from a line starting ? , with the answer from a line starting > right
// under it, and the question's 1-based line number in the note. It's open while Answer is empty.
type Question struct {
	Text   string `json:"text"`
	Answer string `json:"answer,omitempty"`
	Line   int    `json:"line"`
}

// ItemType is the kind of an item.
type ItemType string

func (itemType ItemType) IsValid() bool {
	return slices.Contains(ItemTypes, itemType)
}

// Mark says whether a note's target, or an item's name in it, is new or already exists. Empty means not marked.
type Mark string

// IsValid reports whether mark is empty or one of Marks.
func (mark Mark) IsValid() bool {
	return mark == "" || slices.Contains(Marks, mark)
}

// Symbol returns + for MarkNew and ~ for MarkChanged, as tables and the tree show them, and "" when unmarked.
func (mark Mark) Symbol() string {
	switch mark {
	case MarkNew:
		return symbolNew
	case MarkChanged:
		return symbolChanged
	}
	return ""
}

// Validate rejects an unknown schema version, a path or target that isn't a clean repo path, a path that isn't
// the target's followed by NoteSuffix, an unknown mark, a feature or task holding a control character, and
// UnknownFeature set with no feature.
func (value Note) Validate() error {
	var problems validation.Errors
	if value.SchemaVersion != SchemaVersion {
		problems.Add(FieldSchemaVersion, problemUnknownVersion)
	}
	addRepoPath(&problems, FieldPath, value.Path)
	addRepoPath(&problems, FieldTarget, value.Target)
	if value.Path != "" && value.Target != "" && value.Path != value.Target+NoteSuffix {
		problems.Add(FieldPath, problemNotTarget)
	}
	if !value.Mark.IsValid() {
		problems.Add(FieldMark, validation.InvalidOption(Marks))
	}
	addControl(&problems, FieldFeature, value.Feature)
	if value.UnknownFeature && value.Feature == "" {
		problems.Add(FieldUnknownFeature, problemNoFeature)
	}
	addControl(&problems, FieldTask, value.Task)
	for index, item := range value.Items {
		problems.AddNested(fmt.Sprintf(fieldIndexFormat, FieldItems, index), item.Validate())
	}
	for index, question := range value.Questions {
		problems.AddNested(fmt.Sprintf(fieldIndexFormat, FieldQuestions, index), question.Validate())
	}
	return problems.Err()
}

// Validate rejects an unknown item type, an empty name or one holding a control character, and an unknown mark.
func (value Item) Validate() error {
	var problems validation.Errors
	if !value.Type.IsValid() {
		problems.Add(FieldType, validation.InvalidOption(ItemTypes))
	}
	addRequired(&problems, FieldName, value.Name)
	if !value.Mark.IsValid() {
		problems.Add(FieldMark, validation.InvalidOption(Marks))
	}
	return problems.Err()
}

// Validate rejects empty text and a line number below 1.
func (value Question) Validate() error {
	var problems validation.Errors
	addRequired(&problems, FieldText, value.Text)
	if value.Line < 1 {
		problems.Add(FieldLine, problemLine)
	}
	return problems.Err()
}

// addRepoPath reports a path that's empty, holds a control character, or isn't slash-separated, relative, clean
// and inside the repo, so two spellings of one file can't both appear.
func addRepoPath(problems *validation.Errors, field, repoPath string) {
	addRequired(problems, field, repoPath)
	if repoPath == "" {
		return
	}
	if path.IsAbs(repoPath) || path.Clean(repoPath) != repoPath || repoPath == parentDirectory ||
		strings.HasPrefix(repoPath, parentDirectory+"/") {
		problems.Add(field, problemRepoPath)
	}
}

// addRequired reports text that's empty once trimmed, and text holding a control character.
func addRequired(problems *validation.Errors, field, text string) {
	if strings.TrimSpace(text) == "" {
		problems.Add(field, problemRequired)
	}
	addControl(problems, field, text)
}

// addControl reports text holding a control character, such as a tab, which can't sit on one row of a table.
func addControl(problems *validation.Errors, field, text string) {
	if strings.ContainsFunc(text, unicode.IsControl) {
		problems.Add(field, problemControl)
	}
}
