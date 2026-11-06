// Package merge applies the change between two sketches onto a third, task by task, file by file and item by item,
// so an undo or an edit made from an older copy keeps what other writers saved since, and refuses when they
// touched the same item.
package merge

import (
	"errors"
	"fmt"
	"reflect"
	"slices"
	"strings"

	"github.com/theroymind/go-tix/contracts/sketch"
)

var ErrConflict = errors.New("changed since")

// key names one unit of a sketch: a task, a file of a task, or an item of a file. A task's key has no file, and a
// file's has no item.
type key struct {
	task string
	file string
	item string
}

// unit is a task, file or item at its key. A task or file is only there or not; an item carries its lines too.
type unit struct {
	key  key
	item sketch.Item
}

// Rebase applies the change from base to changed onto current. Each unit the change adds, removes or alters must
// be in current as it was in base, or Rebase fails with ErrConflict naming every one that isn't, and returns
// current. A unit the change adds back lands after the unit before it in changed that current still holds.
func Rebase(base, changed, current sketch.Sketch) (sketch.Sketch, error) {
	baseUnits, changedUnits, currentUnits := flatten(base), flatten(changed), flatten(current)
	baseIndex, changedIndex, currentIndex := index(baseUnits), index(changedUnits), index(currentUnits)
	var touched []key
	for _, candidate := range keysOf(changedUnits, baseUnits) {
		if !same(baseIndex, changedIndex, candidate) {
			touched = append(touched, candidate)
		}
	}
	var conflicts []string
	for _, candidate := range touched {
		if !same(baseIndex, currentIndex, candidate) {
			conflicts = append(conflicts, candidate.String())
		}
	}
	if len(conflicts) > 0 {
		return current, fmt.Errorf("%s %w", strings.Join(conflicts, ", "), ErrConflict)
	}
	result := currentUnits
	for _, candidate := range touched {
		position := slices.IndexFunc(result, func(found unit) bool { return found.key == candidate })
		replacement, inChanged := changedIndex[candidate]
		switch {
		case !inChanged:
			result = slices.Delete(result, position, position+1)
		case position >= 0:
			result[position] = replacement
		default:
			result = slices.Insert(result, insertionPoint(result, changedUnits, candidate), replacement)
		}
	}
	return rebuild(current.SchemaVersion, result), nil
}

// insertionPoint is where a unit added back goes in result: just after the nearest unit before it in changed that
// result holds, or at the start when there's none.
func insertionPoint(result, changedUnits []unit, added key) int {
	at := slices.IndexFunc(changedUnits, func(found unit) bool { return found.key == added })
	for before := at - 1; before >= 0; before-- {
		position := slices.IndexFunc(result, func(found unit) bool { return found.key == changedUnits[before].key })
		if position >= 0 {
			return position + 1
		}
	}
	return 0
}

// flatten lists a sketch's units in order: each task, then each of its files followed by the file's items.
func flatten(value sketch.Sketch) []unit {
	var units []unit
	for _, task := range value.Tasks {
		units = append(units, unit{key: key{task: task.Name}})
		for _, file := range task.Files {
			units = append(units, unit{key: key{task: task.Name, file: file.Path}})
			for _, item := range file.Items {
				units = append(units, unit{key: key{task: task.Name, file: file.Path, item: item.Name}, item: item})
			}
		}
	}
	return units
}

// rebuild nests units back into a sketch, grouping each under its task and file in the order they first appear.
func rebuild(schemaVersion int, units []unit) sketch.Sketch {
	value := sketch.Sketch{SchemaVersion: schemaVersion}
	for _, found := range units {
		task := ensureTask(&value, found.key.task)
		if found.key.file == "" {
			continue
		}
		file := ensureFile(task, found.key.file)
		if found.key.item != "" {
			file.Items = append(file.Items, found.item)
		}
	}
	return value
}

func ensureTask(value *sketch.Sketch, name string) *sketch.Task {
	index := slices.IndexFunc(value.Tasks, func(task sketch.Task) bool { return task.Name == name })
	if index < 0 {
		value.Tasks = append(value.Tasks, sketch.Task{Name: name})
		index = len(value.Tasks) - 1
	}
	return &value.Tasks[index]
}

func ensureFile(task *sketch.Task, path string) *sketch.File {
	index := slices.IndexFunc(task.Files, func(file sketch.File) bool { return file.Path == path })
	if index < 0 {
		task.Files = append(task.Files, sketch.File{Path: path})
		index = len(task.Files) - 1
	}
	return &task.Files[index]
}

func index(units []unit) map[key]unit {
	indexed := make(map[key]unit, len(units))
	for _, found := range units {
		indexed[found.key] = found
	}
	return indexed
}

// keysOf lists every key in first, then the keys of second that first lacks, each once.
func keysOf(first, second []unit) []key {
	var keys []key
	seen := map[key]bool{}
	for _, found := range slices.Concat(first, second) {
		if !seen[found.key] {
			seen[found.key] = true
			keys = append(keys, found.key)
		}
	}
	return keys
}

// same reports whether two sketches hold the unit at candidate alike: both lack it, or both hold it with the same
// item. A line list that's empty and one that's absent count as the same.
func same(left, right map[key]unit, candidate key) bool {
	leftUnit, inLeft := left[candidate]
	rightUnit, inRight := right[candidate]
	return inLeft == inRight && reflect.DeepEqual(withoutEmptyLines(leftUnit.item), withoutEmptyLines(rightUnit.item))
}

func withoutEmptyLines(item sketch.Item) sketch.Item {
	if len(item.Lines) == 0 {
		item.Lines = nil
	}
	return item
}

// String names the unit the way a context does, without its feature: TASK[/FILE[:ITEM]].
func (unitKey key) String() string {
	text := unitKey.task
	if unitKey.file != "" {
		text += "/" + unitKey.file
	}
	if unitKey.item != "" {
		text += ":" + unitKey.item
	}
	return text
}
