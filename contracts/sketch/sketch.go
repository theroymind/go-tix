// Package sketch provides the Note contract: a plain .sketch.md note beside the file it plans, read as the
// feature it belongs to, its status, its body, the links it draws to other files and the questions it asks.
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
// status its front matter gives, its body as written below the front matter, and its edges and questions in the
// order they appear. Mark says whether the target exists yet, and each edge's Mark whether its file does; a note
// read straight from its file has no marks set.
type Note struct {
	SchemaVersion int        `json:"schema_version"`
	Path          string     `json:"path"`
	Target        string     `json:"target"`
	Mark          Mark       `json:"mark,omitempty"`
	Feature       string     `json:"feature,omitempty"`
	Status        Status     `json:"status,omitempty"`
	Body          string     `json:"body"`
	Edges         []Edge     `json:"edges,omitempty"`
	Questions     []Question `json:"questions,omitempty"`
}

// Edge is one link from a note to another file, from a line such as -> src/camera.ts : reads the target: the
// file's slash-separated path from the repo root, the label after the colon, and a Mark saying whether the file
// exists (MarkChanged) or is the target of a note that plans it (MarkNew). It's unmarked when it's neither.
type Edge struct {
	To    string `json:"to"`
	Label string `json:"label,omitempty"`
	Mark  Mark   `json:"mark,omitempty"`
}

// Question is one question the note asks, from a line starting ? , with the answer from the lines starting > right
// under it, joined with newlines, and the question's 1-based line number in the note. It's open while Answer is
// empty.
type Question struct {
	Text   string `json:"text"`
	Answer string `json:"answer,omitempty"`
	Line   int    `json:"line"`
}

// Status says whether a note is still being written or is ready to build from. Empty means the note doesn't say.
type Status string

// IsValid reports whether status is empty or one of Statuses.
func (status Status) IsValid() bool {
	return status == "" || slices.Contains(Statuses, status)
}

// Mark says whether a note's target, or an edge's file, is new or already exists. Empty means not marked.
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
// the target's followed by NoteSuffix, an unknown mark or status, a feature holding a control character, and a
// bad edge or question. The body is free text and never rejected.
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
	if !value.Status.IsValid() {
		problems.Add(FieldStatus, validation.InvalidOption(Statuses))
	}
	for index, edge := range value.Edges {
		problems.AddNested(fmt.Sprintf(fieldIndexFormat, FieldEdges, index), edge.Validate())
	}
	for index, question := range value.Questions {
		problems.AddNested(fmt.Sprintf(fieldIndexFormat, FieldQuestions, index), question.Validate())
	}
	return problems.Err()
}

// Validate rejects a To that isn't a clean repo path, a label holding a control character, and an unknown mark.
func (value Edge) Validate() error {
	var problems validation.Errors
	addRepoPath(&problems, FieldTo, value.To)
	addControl(&problems, FieldLabel, value.Label)
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
