// Package grammar turns one sketch command, such as `fn score_strike` or `in actor Combatant_Id`, into an edit on
// a sketch at a context, the one parser the builder prompt and the one-shot commands share.
package grammar

import (
	"errors"
	"fmt"
	"maps"
	"slices"
	"strconv"
	"strings"

	"github.com/theroymind/go-tix/contracts/sketch"
)

var (
	ErrEmptyLine       = errors.New("empty command")
	ErrUnknownVerb     = errors.New("unknown verb")
	ErrWrongContext    = errors.New("wrong context")
	ErrMissingArgument = errors.New("missing argument")
	ErrNotFound        = errors.New("not in the sketch")
	ErrTypeConflict    = errors.New("already an item of another type")
	ErrBadLineNumber   = errors.New("bad line number")
	ErrBadContext      = errors.New("bad context")
	ErrInvalidSketch   = errors.New("the edit leaves an invalid sketch")
	errNoLineNumberAt  = errors.New("a line number needs an item")
)

// Verbs returns every verb spelling Apply accepts, for completion and help: task and file, the item types, the
// line types the item types don't already name, mv, rm and set, then the one-letter aliases in alphabetical
// order.
func Verbs() []string {
	verbs := []string{VerbTask, VerbFile}
	for _, itemType := range sketch.ItemTypes {
		verbs = append(verbs, string(itemType))
	}
	for _, lineType := range sketch.LineTypes {
		if !slices.Contains(verbs, string(lineType)) {
			verbs = append(verbs, string(lineType))
		}
	}
	verbs = append(verbs, VerbRename, VerbRemove, VerbSet)
	return append(verbs, slices.Sorted(maps.Keys(aliases))...)
}

// FullVerb returns the verb an alias stands for, such as fn for f, and any other verb unchanged.
func FullVerb(verb string) string {
	if full, isAlias := aliases[verb]; isAlias {
		return full
	}
	return verb
}

// Apply runs one command line on value at the context at, and returns the edited sketch and the context the
// command leaves: task, file and item verbs open what they name, mv follows the rename, and rm steps up to the
// parent. value is not changed. A task, file or item verb creates the task and file the context names when the
// sketch doesn't hold them yet, so one call can add an item to an empty sketch.
//
// `rule` adds a rule line inside an item and a rule item at file level.
func Apply(value sketch.Sketch, at Context, line string) (sketch.Sketch, Context, error) {
	verb, argument := splitVerb(line)
	if verb == "" {
		return value, at, ErrEmptyLine
	}
	verb = FullVerb(verb)
	at = at.trimmed()
	edited := value.Trimmed()
	next, err := apply(&edited, at, verb, argument)
	if err != nil {
		return value, at, err
	}
	if err := edited.Validate(); err != nil {
		return value, at, fmt.Errorf("%w: %w", ErrInvalidSketch, err)
	}
	return edited, next, nil
}

func apply(value *sketch.Sketch, at Context, verb, argument string) (Context, error) {
	lineType, itemType := sketch.LineType(verb), sketch.ItemType(verb)
	switch {
	case verb == VerbTask:
		return addTask(value, at, argument)
	case verb == VerbFile:
		return addFile(value, at, argument)
	case lineType.IsValid() && (at.Item != "" || !itemType.IsValid()):
		return addLine(value, at, lineType, argument)
	case itemType.IsValid():
		return addItem(value, at, itemType, argument)
	case verb == VerbRename:
		return rename(value, at, argument)
	case verb == VerbRemove:
		return remove(value, at, argument)
	case verb == VerbSet:
		return setLine(value, at, argument)
	default:
		return at, fmt.Errorf("%w %q", ErrUnknownVerb, verb)
	}
}

func addTask(value *sketch.Sketch, at Context, name string) (Context, error) {
	if name == "" {
		return at, missing(VerbTask, "a task name")
	}
	ensureTask(value, name)
	return Context{Feature: at.Feature, Task: name}, nil
}

func addFile(value *sketch.Sketch, at Context, path string) (Context, error) {
	if at.Task == "" {
		return at, needs(VerbFile, levelTask, at)
	}
	if path == "" {
		return at, missing(VerbFile, "a repo path")
	}
	ensureFile(ensureTask(value, at.Task), path)
	return Context{Feature: at.Feature, Task: at.Task, File: path}, nil
}

// addItem opens the item when the file already holds one of that type and name, so typing it again moves there.
func addItem(value *sketch.Sketch, at Context, itemType sketch.ItemType, name string) (Context, error) {
	if at.File == "" {
		return at, needs(string(itemType), levelFile, at)
	}
	if name == "" {
		return at, missing(string(itemType), "an item name")
	}
	file := ensureFile(ensureTask(value, at.Task), at.File)
	next := Context{Feature: at.Feature, Task: at.Task, File: at.File, Item: name}
	if index := findItem(file, name); index >= 0 {
		if existing := file.Items[index].Type; existing != itemType {
			return at, fmt.Errorf("%w: %s is a %s in %s", ErrTypeConflict, name, existing, at.File)
		}
		return next, nil
	}
	file.Items = append(file.Items, sketch.Item{Type: itemType, Name: name})
	return next, nil
}

func addLine(value *sketch.Sketch, at Context, lineType sketch.LineType, text string) (Context, error) {
	if at.Item == "" {
		return at, needs(string(lineType), levelItem, at)
	}
	if text == "" {
		return at, missing(string(lineType), "its text")
	}
	item, err := lookupItem(value, at)
	if err != nil {
		return at, err
	}
	item.Lines = append(item.Lines, sketch.Line{Type: lineType, Text: text})
	return at, nil
}

// rename renames the deepest level the context sets: the item, else the file, else the task.
func rename(value *sketch.Sketch, at Context, name string) (Context, error) {
	if at.Task == "" {
		return at, needs(VerbRename, levelTask, at)
	}
	if name == "" {
		return at, missing(VerbRename, "the new name")
	}
	switch at.level() {
	case levelItem:
		item, err := lookupItem(value, at)
		if err != nil {
			return at, err
		}
		item.Name, at.Item = name, name
	case levelFile:
		file, err := lookupFile(value, at)
		if err != nil {
			return at, err
		}
		file.Path, at.File = name, name
	default:
		task, err := lookupTask(value, at)
		if err != nil {
			return at, err
		}
		task.Name, at.Task = name, name
	}
	return at, nil
}

// remove removes line N of the context's item when given N, and otherwise the deepest level the context sets.
func remove(value *sketch.Sketch, at Context, argument string) (Context, error) {
	if argument != "" {
		item, index, err := lookupLine(value, at, argument)
		if err != nil {
			return at, err
		}
		item.Lines = slices.Delete(item.Lines, index, index+1)
		return at, nil
	}
	switch at.level() {
	case levelItem:
		file, err := lookupFile(value, at)
		if err != nil {
			return at, err
		}
		index := findItem(file, at.Item)
		if index < 0 {
			return at, notFound(levelItem, at)
		}
		file.Items = slices.Delete(file.Items, index, index+1)
		at.Item = ""
	case levelFile:
		task, err := lookupTask(value, at)
		if err != nil {
			return at, err
		}
		index := findFile(task, at.File)
		if index < 0 {
			return at, notFound(levelFile, at)
		}
		task.Files = slices.Delete(task.Files, index, index+1)
		at.File = ""
	case levelTask:
		index := findTask(value, at.Task)
		if index < 0 {
			return at, notFound(levelTask, at)
		}
		value.Tasks = slices.Delete(value.Tasks, index, index+1)
		at.Task = ""
	default:
		return at, needs(VerbRemove, levelTask, at)
	}
	return at, nil
}

// setLine replaces the text of line N of the context's item, keeping its type. It clears a question's answer, so
// a reworded question opens again.
func setLine(value *sketch.Sketch, at Context, argument string) (Context, error) {
	number, text := splitVerb(argument)
	if text == "" {
		return at, missing(VerbSet, "a line number and its new text")
	}
	item, index, err := lookupLine(value, at, number)
	if err != nil {
		return at, err
	}
	item.Lines[index].Text, item.Lines[index].Answer = text, ""
	return at, nil
}

// lookupLine finds the line a 1-based number names in the context's item, the numbering show prints.
func lookupLine(value *sketch.Sketch, at Context, number string) (*sketch.Item, int, error) {
	if at.Item == "" {
		return nil, 0, fmt.Errorf("%w: %w, and the context is %s", ErrWrongContext, errNoLineNumberAt, describe(at))
	}
	item, err := lookupItem(value, at)
	if err != nil {
		return nil, 0, err
	}
	parsed, err := strconv.Atoi(number)
	if err != nil || parsed < firstLine || parsed > len(item.Lines) {
		return nil, 0, fmt.Errorf("%w %q: %s has %d lines", ErrBadLineNumber, number, at.Item, len(item.Lines))
	}
	return item, parsed - firstLine, nil
}

func lookupTask(value *sketch.Sketch, at Context) (*sketch.Task, error) {
	index := findTask(value, at.Task)
	if index < 0 {
		return nil, notFound(levelTask, at)
	}
	return &value.Tasks[index], nil
}

func lookupFile(value *sketch.Sketch, at Context) (*sketch.File, error) {
	task, err := lookupTask(value, at)
	if err != nil {
		return nil, err
	}
	index := findFile(task, at.File)
	if index < 0 {
		return nil, notFound(levelFile, at)
	}
	return &task.Files[index], nil
}

func lookupItem(value *sketch.Sketch, at Context) (*sketch.Item, error) {
	file, err := lookupFile(value, at)
	if err != nil {
		return nil, err
	}
	index := findItem(file, at.Item)
	if index < 0 {
		return nil, notFound(levelItem, at)
	}
	return &file.Items[index], nil
}

func ensureTask(value *sketch.Sketch, name string) *sketch.Task {
	index := findTask(value, name)
	if index < 0 {
		value.Tasks = append(value.Tasks, sketch.Task{Name: name})
		index = len(value.Tasks) - 1
	}
	return &value.Tasks[index]
}

func ensureFile(task *sketch.Task, path string) *sketch.File {
	index := findFile(task, path)
	if index < 0 {
		task.Files = append(task.Files, sketch.File{Path: path})
		index = len(task.Files) - 1
	}
	return &task.Files[index]
}

func findTask(value *sketch.Sketch, name string) int {
	return slices.IndexFunc(value.Tasks, func(task sketch.Task) bool { return task.Name == name })
}

func findFile(task *sketch.Task, path string) int {
	return slices.IndexFunc(task.Files, func(file sketch.File) bool { return file.Path == path })
}

func findItem(file *sketch.File, name string) int {
	return slices.IndexFunc(file.Items, func(item sketch.Item) bool { return item.Name == name })
}

// splitVerb splits a line at its first run of spaces into the verb and the trimmed rest, which keeps its own
// inner spaces.
func splitVerb(line string) (string, string) {
	line = strings.TrimSpace(line)
	index := strings.IndexFunc(line, isSpace)
	if index < 0 {
		return line, ""
	}
	return line[:index], strings.TrimSpace(line[index:])
}

func isSpace(character rune) bool {
	return character == ' ' || character == '\t'
}

func needs(verb, level string, at Context) error {
	return fmt.Errorf("%w: %s needs %s, and the context is %s", ErrWrongContext, verb, withArticle(level), describe(at))
}

func withArticle(level string) string {
	if level == levelItem {
		return "an " + level
	}
	return "a " + level
}

func missing(verb, what string) error {
	return fmt.Errorf("%w: %s needs %s", ErrMissingArgument, verb, what)
}

func notFound(level string, at Context) error {
	return fmt.Errorf("%w: no %s %s", ErrNotFound, level, at)
}

// describe names the context and its level, such as `the file fight-camera/score-strikes/client/camera.odin`.
func describe(at Context) string {
	return fmt.Sprintf("the %s %s", at.level(), at)
}
