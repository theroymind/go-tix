// Package sketch provides the Sketch contract: a feature's rough shape of code, as tasks, then files by repo
// path, then typed items made of typed lines, which sketch stores in a feature's sketch.yml.
package sketch

import (
	"fmt"
	"path"
	"slices"
	"strings"

	"github.com/theroymind/go-tix/contracts/validation"
)

// Sketch is one feature's sketch: the schema version it was written with and its tasks in the order they were
// added. The feature it belongs to is the directory that holds it, not a field.
type Sketch struct {
	SchemaVersion int    `yaml:"schema_version" json:"schema_version"`
	Tasks         []Task `yaml:"tasks,omitempty" json:"tasks,omitempty"`
}

// Task is one task of a feature and the files it touches, in the order they were added.
type Task struct {
	Name  string `yaml:"name" json:"name"`
	Files []File `yaml:"files,omitempty" json:"files,omitempty"`
}

// File is one file a task touches, by its slash-separated path from the repo root, and the items it holds.
type File struct {
	Path  string `yaml:"path" json:"path"`
	Items []Item `yaml:"items,omitempty" json:"items,omitempty"`
}

// Item is one named piece of a file, such as a type or a function, and the lines that describe it. An item
// keeps this one shape in every language.
type Item struct {
	Type  ItemType `yaml:"type" json:"type"`
	Name  string   `yaml:"name" json:"name"`
	Lines []Line   `yaml:"lines,omitempty" json:"lines,omitempty"`
}

// Line is one typed line of an item, such as an input, an output or a question. Only an ask line holds an
// Answer, and it's open while its Answer is empty.
type Line struct {
	Type   LineType `yaml:"type" json:"type"`
	Text   string   `yaml:"text" json:"text"`
	Answer string   `yaml:"answer,omitempty" json:"answer,omitempty"`
}

// IsOpen reports whether the line is a question still waiting for its answer.
func (value Line) IsOpen() bool {
	return value.Type == LineTypeAsk && strings.TrimSpace(value.Answer) == ""
}

// ItemType is the kind of an item.
type ItemType string

// LineType is the kind of a line.
type LineType string

// New returns an empty sketch at the current schema version.
func New() Sketch {
	return Sketch{SchemaVersion: SchemaVersion}
}

// Trimmed returns a copy of value with the spaces around every name, path, line text and answer removed, the form
// Validate compares and the store saves.
func (value Sketch) Trimmed() Sketch {
	trimmed := value
	trimmed.Tasks = make([]Task, len(value.Tasks))
	for taskIndex, task := range value.Tasks {
		task.Name = strings.TrimSpace(task.Name)
		task.Files = slices.Clone(task.Files)
		for fileIndex, file := range task.Files {
			file.Path = strings.TrimSpace(file.Path)
			file.Items = slices.Clone(file.Items)
			for itemIndex, item := range file.Items {
				item.Name = strings.TrimSpace(item.Name)
				item.Lines = slices.Clone(item.Lines)
				for lineIndex := range item.Lines {
					line := &item.Lines[lineIndex]
					line.Text, line.Answer = strings.TrimSpace(line.Text), strings.TrimSpace(line.Answer)
				}
				file.Items[itemIndex] = item
			}
			task.Files[fileIndex] = file
		}
		trimmed.Tasks[taskIndex] = task
	}
	if value.Tasks == nil {
		trimmed.Tasks = nil
	}
	return trimmed
}

func (itemType ItemType) IsValid() bool {
	return slices.Contains(ItemTypes, itemType)
}

func (lineType LineType) IsValid() bool {
	return slices.Contains(LineTypes, lineType)
}

// Validate rejects an unknown schema version and a task name used twice. Names, paths and line texts compare
// with the spaces around them trimmed.
func (value Sketch) Validate() error {
	var problems validation.Errors
	if value.SchemaVersion != SchemaVersion {
		problems.Add(FieldSchemaVersion, problemUnknownVersion)
	}
	seen := map[string]bool{}
	for index, task := range value.Tasks {
		field := fmt.Sprintf(fieldIndexFormat, FieldTasks, index)
		problems.AddNested(field, task.Validate())
		addDuplicate(&problems, field+"."+FieldName, task.Name, seen, problemDuplicateTask)
	}
	return problems.Err()
}

// Validate rejects an empty name and a file path used twice in the task.
func (value Task) Validate() error {
	var problems validation.Errors
	addBlank(&problems, FieldName, value.Name)
	seen := map[string]bool{}
	for index, file := range value.Files {
		field := fmt.Sprintf(fieldIndexFormat, FieldFiles, index)
		problems.AddNested(field, file.Validate())
		addDuplicate(&problems, field+"."+FieldPath, file.Path, seen, problemDuplicateFile)
	}
	return problems.Err()
}

// Validate rejects an empty path, one that isn't clean and relative inside the repo, and an item name used
// twice in the file.
func (value File) Validate() error {
	var problems validation.Errors
	addBlank(&problems, FieldPath, value.Path)
	if repoPath := strings.TrimSpace(value.Path); repoPath != "" && !isRepoPath(repoPath) {
		problems.Add(FieldPath, problemRepoPath)
	}
	seen := map[string]bool{}
	for index, item := range value.Items {
		field := fmt.Sprintf(fieldIndexFormat, FieldItems, index)
		problems.AddNested(field, item.Validate())
		addDuplicate(&problems, field+"."+FieldName, item.Name, seen, problemDuplicateItem)
	}
	return problems.Err()
}

// Validate rejects an unknown item type and an empty name.
func (value Item) Validate() error {
	var problems validation.Errors
	if !value.Type.IsValid() {
		problems.Add(FieldType, validation.InvalidOption(ItemTypes))
	}
	addBlank(&problems, FieldName, value.Name)
	for index, line := range value.Lines {
		problems.AddNested(fmt.Sprintf(fieldIndexFormat, FieldLines, index), line.Validate())
	}
	return problems.Err()
}

// Validate rejects an unknown line type, empty text, and an answer on a line that isn't a question.
func (value Line) Validate() error {
	var problems validation.Errors
	if !value.Type.IsValid() {
		problems.Add(FieldType, validation.InvalidOption(LineTypes))
	}
	addBlank(&problems, FieldText, value.Text)
	if value.Type != LineTypeAsk && strings.TrimSpace(value.Answer) != "" {
		problems.Add(FieldAnswer, problemAnswerNotAsk)
	}
	return problems.Err()
}

func addBlank(problems *validation.Errors, field, text string) {
	if strings.TrimSpace(text) == "" {
		problems.Add(field, problemRequired)
	}
}

// isRepoPath reports whether repoPath is slash-separated, relative, already clean, and stays inside the repo, so
// two spellings of one file can't both appear.
func isRepoPath(repoPath string) bool {
	return !path.IsAbs(repoPath) && path.Clean(repoPath) == repoPath && repoPath != parentDirectory &&
		!strings.HasPrefix(repoPath, parentDirectory+"/")
}

// addDuplicate reports name when seen already holds it, comparing trimmed, and ignores a blank name, which
// addBlank reports.
func addDuplicate(problems *validation.Errors, field, name string, seen map[string]bool, problem string) {
	name = strings.TrimSpace(name)
	if name == "" {
		return
	}
	if seen[name] {
		problems.Add(field, problem)
	}
	seen[name] = true
}
