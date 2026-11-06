package builder

import (
	"os"
	"path/filepath"
	"slices"
	"strings"

	"github.com/theroymind/go-tix/contracts/sketch"
	"github.com/theroymind/go-tix/internal/sketch/grammar"
)

// Completer offers completions for the word under the cursor: a verb or builder command as the first word, then
// the feature's tasks after task, repo paths after file, and the context file's item names after an item verb.
type Completer struct {
	Root    string
	Session *Session
	Load    Loader
	// Tasks are the feature's tix tasks, offered after task beside the sketch's own.
	Tasks []string
}

// Complete is a liner.WordCompleter: it returns the line before the word, the candidates for the word, and the
// line after the cursor.
func (completer Completer) Complete(line string, pos int) (string, []string, string) {
	head, tail := line[:pos], line[pos:]
	verb, argument, hasArgument := strings.Cut(strings.TrimLeft(head, " "), " ")
	if !hasArgument {
		return head[:len(head)-len(verb)], completer.verbs(verb), tail
	}
	argument = strings.TrimLeft(argument, " ")
	before := head[:len(head)-len(argument)]
	switch fullVerb := grammar.FullVerb(verb); {
	case fullVerb == grammar.VerbTask:
		return before, withPrefix(completer.taskNames(), argument), tail
	case fullVerb == grammar.VerbFile:
		return before, completer.paths(argument), tail
	case isOnlyItemVerb(fullVerb) || (sketch.ItemType(fullVerb).IsValid() && completer.Session.At.Item == ""):
		return before, withPrefix(completer.itemNames(), argument), tail
	default:
		return before, nil, tail
	}
}

func (completer Completer) verbs(prefix string) []string {
	var candidates []string
	for _, verb := range append(grammar.Verbs(), sessionCommands...) {
		if strings.HasPrefix(verb, prefix) {
			candidates = append(candidates, verb+" ")
		}
	}
	return candidates
}

// taskNames are the tix tasks, then the sketch's own tasks tix doesn't list, without repeats.
func (completer Completer) taskNames() []string {
	names := slices.Clone(completer.Tasks)
	for _, task := range completer.loaded().Tasks {
		if !slices.Contains(names, task.Name) {
			names = append(names, task.Name)
		}
	}
	return names
}

func (completer Completer) itemNames() []string {
	at := completer.Session.At
	var names []string
	for _, task := range completer.loaded().Tasks {
		for _, file := range task.Files {
			if task.Name != at.Task || file.Path != at.File {
				continue
			}
			for _, item := range file.Items {
				names = append(names, item.Name)
			}
		}
	}
	return names
}

// paths offers the entries of the directory the prefix names, from the repo root, with a slash after each
// directory so the next tab goes into it, and the sketch's own file paths for files that don't exist yet. Hidden
// entries are left out.
func (completer Completer) paths(prefix string) []string {
	dir := ""
	if index := strings.LastIndex(prefix, pathSeparator); index >= 0 {
		dir = prefix[:index+1]
	}
	var candidates []string
	entries, _ := os.ReadDir(filepath.Join(completer.Root, filepath.FromSlash(dir)))
	for _, entry := range entries {
		if strings.HasPrefix(entry.Name(), hiddenPrefix) {
			continue
		}
		candidate := dir + entry.Name()
		if entry.IsDir() {
			candidate += pathSeparator
		}
		candidates = append(candidates, candidate)
	}
	for _, task := range completer.loaded().Tasks {
		for _, file := range task.Files {
			if !slices.Contains(candidates, file.Path) {
				candidates = append(candidates, file.Path)
			}
		}
	}
	return withPrefix(candidates, prefix)
}

// loaded is the stored sketch, or an empty one when it can't be read, since completion has nowhere to report.
func (completer Completer) loaded() sketch.Sketch {
	value, err := completer.Load()
	if err != nil {
		return sketch.New()
	}
	return value
}

// isOnlyItemVerb reports whether verb always names an item, unlike rule, which names a line inside an item.
func isOnlyItemVerb(verb string) bool {
	return sketch.ItemType(verb).IsValid() && !sketch.LineType(verb).IsValid()
}

func withPrefix(candidates []string, prefix string) []string {
	var matches []string
	for _, candidate := range candidates {
		if strings.HasPrefix(candidate, prefix) {
			matches = append(matches, candidate)
		}
	}
	return matches
}
