package grammar

import (
	"fmt"
	"strings"
)

// Context is where an edit applies: a feature, then optionally one of its tasks, a file of that task by repo
// path, and an item of that file. Each level is set only when the one above it is.
type Context struct {
	Feature string
	Task    string
	File    string
	Item    string
}

// ParseContext reads a context written FEATURE[/TASK[/FILE[:ITEM]]]. The file is everything after the task's
// slash up to the first colon, so it keeps its own slashes, and the item is everything after that colon.
func ParseContext(ref string) (Context, error) {
	var at Context
	var rest, file string
	var hasTask, hasFile, hasItem bool
	at.Feature, rest, hasTask = strings.Cut(ref, taskSeparator)
	if hasTask {
		at.Task, file, hasFile = strings.Cut(rest, taskSeparator)
	}
	if hasFile {
		at.File, at.Item, hasItem = strings.Cut(file, itemSeparator)
	}
	switch {
	case !isFeatureName(at.Feature):
		return Context{}, fmt.Errorf("%w %q: the feature must be a name, not empty, %q, %q or holding %q", ErrBadContext, ref,
			currentSegment, parentSegment, backslash)
	case hasTask && strings.TrimSpace(at.Task) == "":
		return Context{}, fmt.Errorf("%w %q: the task is empty", ErrBadContext, ref)
	case hasFile && strings.TrimSpace(at.File) == "":
		return Context{}, fmt.Errorf("%w %q: the file is empty", ErrBadContext, ref)
	case hasItem && strings.TrimSpace(at.Item) == "":
		return Context{}, fmt.Errorf("%w %q: the item is empty", ErrBadContext, ref)
	}
	return at.trimmed(), nil
}

// String writes the context back in the form ParseContext reads.
func (at Context) String() string {
	text := at.Feature
	if at.Task != "" {
		text += taskSeparator + at.Task
	}
	if at.File != "" {
		text += taskSeparator + at.File
	}
	if at.Item != "" {
		text += itemSeparator + at.Item
	}
	return text
}

// level names the deepest level the context sets.
func (at Context) level() string {
	switch {
	case at.Item != "":
		return levelItem
	case at.File != "":
		return levelFile
	case at.Task != "":
		return levelTask
	default:
		return levelFeature
	}
}

func (at Context) trimmed() Context {
	return Context{Feature: strings.TrimSpace(at.Feature), Task: strings.TrimSpace(at.Task),
		File: strings.TrimSpace(at.File), Item: strings.TrimSpace(at.Item)}
}

// isFeatureName reports whether name can be a feature's directory: one path segment that isn't . or ..
func isFeatureName(name string) bool {
	name = strings.TrimSpace(name)
	return name != "" && name != currentSegment && name != parentSegment && !strings.Contains(name, backslash)
}
