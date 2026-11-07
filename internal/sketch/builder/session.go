package builder

import (
	"bytes"
	"errors"
	"fmt"
	"path"
	"slices"
	"strings"

	"go.yaml.in/yaml/v3"

	"github.com/theroymind/go-tix/contracts/sketch"
	"github.com/theroymind/go-tix/internal/sketch/grammar"
	"github.com/theroymind/go-tix/internal/sketch/merge"
)

var (
	ErrQuit          = errors.New("quit")
	ErrNothingToUndo = errors.New("nothing to undo in this session")
	ErrEditNeedsItem = errors.New("edit needs an item")
	ErrItemGone      = errors.New("the item is no longer in the sketch")
	ErrEditRejected  = errors.New("the edited item doesn't fit the sketch; nothing was saved")
	ErrUndoConflict  = errors.New("can't undo: another writer changed what the edit touched; nothing was undone")
	ErrEditConflict  = errors.New("can't save the edit: another writer changed the item while it was open")
)

// Updater runs edit on the stored sketch under its lock and saves what edit returns, as store.Update does.
type Updater func(edit func(sketch.Sketch) (sketch.Sketch, error)) error

// Loader reads the stored sketch, an empty one when there's none yet.
type Loader func() (sketch.Sketch, error)

// Editor lets a person change text, such as by opening it in $EDITOR, and returns what they saved.
type Editor func(text []byte) ([]byte, error)

// Session is one builder session on a feature's sketch: the context the prompt shows and each of its edits, newest
// last, for undo. Every edit applies to the stored sketch as it is at that moment, and undo and edit replay only
// what they change through merge.Rebase, so what another process saves in between still lands.
type Session struct {
	At     grammar.Context
	update Updater
	load   Loader
	editor Editor
	undo   []step
}

// step is one edit undo can revert: the sketch before and after it, and the context it was made from.
type step struct {
	before sketch.Sketch
	after  sketch.Sketch
	at     grammar.Context
}

// NewSession starts a session at the feature, saving through update, reading through load, and opening items in
// editor.
func NewSession(feature string, update Updater, load Loader, editor Editor) *Session {
	return &Session{At: grammar.Context{Feature: feature}, update: update, load: load, editor: editor}
}

// Prompt is the session's prompt: its context with the file's directories shortened to …, so that a deep path
// leaves the terminal room to edit the line, then the prompt marker. where prints the context in full.
func (session *Session) Prompt() string {
	short := session.At
	if directory := path.Dir(short.File); directory != "." {
		short.File = ellipsis + pathSeparator + path.Base(short.File)
	}
	return short.String() + promptMarker
}

// Run runs one line typed at the prompt and returns what to print, if anything. Besides the grammar's commands,
// whose .. and / move without saving, it takes undo, edit, where, and exit or quit, which return ErrQuit.
func (session *Session) Run(line string) (string, error) {
	if next, moved := grammar.Move(session.At, line); moved {
		session.At = next
		return "", nil
	}
	switch strings.TrimSpace(line) {
	case "":
		return "", nil
	case commandUndo:
		return session.undoLast()
	case commandEdit:
		return session.editItem()
	case commandWhere:
		return session.At.String(), nil
	case commandExit, commandQuit:
		return "", ErrQuit
	}
	at := session.At
	var made step
	err := session.update(func(value sketch.Sketch) (sketch.Sketch, error) {
		edited, next, err := grammar.Apply(value, at, line)
		made, at = step{before: value, after: edited, at: session.At}, next
		return edited, err
	})
	if err != nil {
		return "", err
	}
	session.undo = append(session.undo, made)
	session.At = at
	return "", nil
}

// undoLast reverts the session's last edit and returns to the context it was made from. It reverts only the tasks,
// files and items that edit touched, so another writer's changes elsewhere stay, and refuses, keeping the edit to
// undo, when another writer has changed one of those since.
func (session *Session) undoLast() (string, error) {
	if len(session.undo) == 0 {
		return "", ErrNothingToUndo
	}
	last := session.undo[len(session.undo)-1]
	err := session.update(func(current sketch.Sketch) (sketch.Sketch, error) {
		reverted, err := merge.Rebase(last.after.Trimmed(), last.before.Trimmed(), current)
		if errors.Is(err, merge.ErrConflict) {
			return current, fmt.Errorf("%w: %w", ErrUndoConflict, err)
		}
		return reverted, err
	})
	if err != nil {
		return "", err
	}
	session.undo = session.undo[:len(session.undo)-1]
	session.At = last.at
	return undoneMessage, nil
}

// editItem opens the context's item as YAML in the editor, without holding the lock while it's open, and saves the
// edited item in its place only when it decodes strictly and the sketch still validates with it. A rename in the
// editor moves the context to the new name. When another writer changed the item while it was open, nothing is
// saved, and the error holds the edited YAML so it can be redone.
func (session *Session) editItem() (string, error) {
	at := session.At
	if at.Item == "" {
		return "", fmt.Errorf("%w, and the context is %s", ErrEditNeedsItem, at)
	}
	current, err := session.load()
	if err != nil {
		return "", err
	}
	item, found := findItem(current, at)
	if !found {
		return "", fmt.Errorf("%w: %s", ErrItemGone, at)
	}
	text, err := encodeItem(item)
	if err != nil {
		return "", err
	}
	editedText, err := session.editor(text)
	if err != nil {
		return "", err
	}
	edited, err := decodeItem(editedText)
	if err != nil {
		return "", fmt.Errorf("%w: %w", ErrEditRejected, err)
	}
	withEdit, err := replaceItem(current, at, edited)
	if err != nil {
		return "", err
	}
	if err := withEdit.Trimmed().Validate(); err != nil {
		return "", fmt.Errorf("%w: %w", ErrEditRejected, err)
	}
	var made step
	err = session.update(func(value sketch.Sketch) (sketch.Sketch, error) {
		saved, err := merge.Rebase(current.Trimmed(), withEdit.Trimmed(), value)
		if errors.Is(err, merge.ErrConflict) {
			return value, fmt.Errorf("%w: %w; your edit was:\n%s", ErrEditConflict, err, editedText)
		}
		if err != nil {
			return value, err
		}
		if err := saved.Trimmed().Validate(); err != nil {
			return value, fmt.Errorf("%w: %w", ErrEditRejected, err)
		}
		made = step{before: value, after: saved, at: at}
		return saved, nil
	})
	if err != nil {
		return "", err
	}
	session.undo = append(session.undo, made)
	session.At.Item = strings.TrimSpace(edited.Name)
	return "", nil
}

func findItem(value sketch.Sketch, at grammar.Context) (sketch.Item, bool) {
	for _, task := range value.Tasks {
		for _, file := range task.Files {
			for _, item := range file.Items {
				if task.Name == at.Task && file.Path == at.File && item.Name == at.Item {
					return item, true
				}
			}
		}
	}
	return sketch.Item{}, false
}

// replaceItem returns a copy of value with the context's item replaced by item.
func replaceItem(value sketch.Sketch, at grammar.Context, item sketch.Item) (sketch.Sketch, error) {
	replaced := value.Trimmed()
	for taskIndex, task := range replaced.Tasks {
		for fileIndex, file := range task.Files {
			if task.Name != at.Task || file.Path != at.File {
				continue
			}
			index := slices.IndexFunc(file.Items, func(candidate sketch.Item) bool { return candidate.Name == at.Item })
			if index >= 0 {
				replaced.Tasks[taskIndex].Files[fileIndex].Items[index] = item
				return replaced, nil
			}
		}
	}
	return value, fmt.Errorf("%w: %s", ErrItemGone, at)
}

func encodeItem(item sketch.Item) ([]byte, error) {
	var buffer bytes.Buffer
	encoder := yaml.NewEncoder(&buffer)
	encoder.SetIndent(yamlIndent)
	if err := encoder.Encode(item); err != nil {
		return nil, err
	}
	if err := encoder.Close(); err != nil {
		return nil, err
	}
	return buffer.Bytes(), nil
}

// decodeItem reads one item, rejecting a key the item has no field for, as the store does.
func decodeItem(text []byte) (sketch.Item, error) {
	decoder := yaml.NewDecoder(bytes.NewReader(text))
	decoder.KnownFields(true)
	var item sketch.Item
	if err := decoder.Decode(&item); err != nil {
		return sketch.Item{}, err
	}
	if err := item.Validate(); err != nil {
		return sketch.Item{}, err
	}
	return item, nil
}
