package builder

import (
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/theroymind/go-tix/contracts/sketch"
)

const (
	feature    = "fight-camera"
	atTask     = feature + "/score-strikes"
	atFile     = atTask + "/client/camera.odin"
	atItem     = atFile + ":score_strike"
	promptFile = atTask + "/…/camera.odin"
	promptItem = promptFile + ":score_strike"
)

// memory stands in for the store: one sketch, updated and loaded in place.
type memory struct {
	value sketch.Sketch
}

func (stored *memory) update(edit func(sketch.Sketch) (sketch.Sketch, error)) error {
	edited, err := edit(stored.value)
	if err != nil {
		return err
	}
	if err := edited.Trimmed().Validate(); err != nil {
		return err
	}
	stored.value = edited.Trimmed()
	return nil
}

func (stored *memory) load() (sketch.Sketch, error) { return stored.value, nil }

// editorReturning is an Editor that saves text, recording what it was opened with.
func editorReturning(text string, opened *string) Editor {
	return func(original []byte) ([]byte, error) {
		*opened = string(original)
		return []byte(text), nil
	}
}

func newSession(stored *memory, editor Editor) *Session {
	return NewSession(feature, stored.update, stored.load, editor)
}

// runAll runs each line, failing the test on the first error.
func runAll(t *testing.T, session *Session, lines ...string) {
	t.Helper()
	for _, line := range lines {
		if _, err := session.Run(line); err != nil {
			t.Fatalf("Run(%q) = %v", line, err)
		}
	}
}

var startLines = []string{"task score-strikes", "file client/camera.odin", "fn score_strike", "in actor Combatant_Id"}

func TestNavigation(t *testing.T) {
	tests := []struct {
		name     string
		lines    []string
		expected string
	}{
		{"The session starts at the feature", nil, feature + "> "},
		{"Verbs open what they name, the file's directories shortened", startLines, promptItem + "> "},
		{".. steps up from an item", append(startLines, ".."), promptFile + "> "},
		{".. steps up twice", append(startLines, "..", ".."), atTask + "> "},
		{".. at the feature stays", []string{".."}, feature + "> "},
		{"/ returns to the feature", append(startLines, "/"), feature + "> "},
		{"A blank line changes nothing", append(startLines, "   "), promptItem + "> "},
		{"An item verb inside an item opens the sibling", append(startLines, "t Camera_Move"), promptFile + ":Camera_Move> "},
		{"A file at the repo root isn't shortened", []string{"task t", "file main.go", "fn main"}, feature + "/t/main.go:main> "},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			session := newSession(&memory{value: sketch.New()}, nil)

			runAll(t, session, tt.lines...)

			if actual := session.Prompt(); actual != tt.expected {
				t.Errorf("Prompt() = %q, want %q", actual, tt.expected)
			}
		})
	}
}

func TestRunErrors(t *testing.T) {
	tests := []struct {
		name     string
		line     string
		expected string
	}{
		{"A grammar error leaves the context", "in actor", "in needs an item"},
		{"undo with nothing done", "undo", ErrNothingToUndo.Error()},
		{"edit outside an item", "edit", "edit needs an item, and the context is " + feature},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			session := newSession(&memory{value: sketch.New()}, nil)

			_, err := session.Run(tt.line)

			if err == nil || !strings.Contains(err.Error(), tt.expected) {
				t.Errorf("Run(%q) = %v, want it to say %q", tt.line, err, tt.expected)
			}
			if session.At.String() != feature {
				t.Errorf("context = %q, want it unchanged", session.At)
			}
		})
	}
}

func TestWhere(t *testing.T) {
	session := newSession(&memory{value: sketch.New()}, nil)
	runAll(t, session, startLines...)

	output, err := session.Run("where")

	if err != nil || output != atItem {
		t.Errorf("Run(where) = %q, %v, want %q", output, err, atItem)
	}
}

func TestUndoKeepsAnotherWritersChanges(t *testing.T) {
	stored := &memory{value: sketch.New()}
	session := newSession(stored, nil)
	runAll(t, session, append(startLines, "..", "t Camera_Move", "f score_strike", "out Camera_Move")...)
	other := NewSession(feature, stored.update, stored.load, nil)
	runAll(t, other, "task frame-kills", "file client/kill.odin", "test frames a kill", "/",
		"task score-strikes", "file client/camera.odin", "t Camera_Move", "a cut or ease")

	runAll(t, session, "undo")

	lines := stored.value.Tasks[0].Files[0].Items[0].Lines
	if len(lines) != 1 || len(stored.value.Tasks) != 2 || len(stored.value.Tasks[0].Files[0].Items[1].Lines) != 1 {
		t.Errorf("after undo, sketch = %+v, want score_strike's out line gone and the other writer's task and ask kept",
			stored.value)
	}
}

func TestUndoRefusesWhenTheSameItemChanged(t *testing.T) {
	stored := &memory{value: sketch.New()}
	session := newSession(stored, nil)
	runAll(t, session, startLines...)
	runAll(t, NewSession(feature, stored.update, stored.load, nil), "task score-strikes", "file client/camera.odin",
		"f score_strike", "a who anchors a kill")
	before := stored.value

	_, err := session.Run("undo")

	if !errors.Is(err, ErrUndoConflict) || !strings.Contains(err.Error(), "score_strike changed since") {
		t.Fatalf("Run(undo) = %v, want %v naming score_strike", err, ErrUndoConflict)
	}
	if !reflect.DeepEqual(stored.value, before) {
		t.Errorf("a refused undo changed the sketch")
	}
	stored.value.Tasks[0].Files[0].Items[0].Lines = stored.value.Tasks[0].Files[0].Items[0].Lines[:1]
	if _, err := session.Run("undo"); err != nil {
		t.Errorf("Run(undo) once the item is back as the edit left it = %v, want the edit still there to undo", err)
	}
}

func TestEditRefusesWhenTheItemChangedWhileOpen(t *testing.T) {
	stored := &memory{value: sketch.New()}
	saved := "type: fn\nname: score_strike\nlines:\n  - type: out\n    text: Camera_Move\n"
	var other *Session
	editor := func([]byte) ([]byte, error) {
		runAll(t, other, "task score-strikes", "file client/camera.odin", "f score_strike", "a who anchors a kill")
		return []byte(saved), nil
	}
	session := newSession(stored, editor)
	other = NewSession(feature, stored.update, stored.load, nil)
	runAll(t, session, startLines...)

	_, err := session.Run("edit")

	if !errors.Is(err, ErrEditConflict) || !strings.Contains(err.Error(), saved) {
		t.Fatalf("Run(edit) = %v, want %v holding the edited YAML", err, ErrEditConflict)
	}
	if lines := stored.value.Tasks[0].Files[0].Items[0].Lines; len(lines) != 2 || lines[1].Type != sketch.LineTypeAsk {
		t.Errorf("after a refused edit, lines = %+v, want the other writer's ask kept", lines)
	}
}

func TestEditKeepsAnotherWritersChangesElsewhere(t *testing.T) {
	stored := &memory{value: sketch.New()}
	var other *Session
	editor := func([]byte) ([]byte, error) {
		runAll(t, other, "task frame-kills")
		return []byte("type: fn\nname: score_hit\n"), nil
	}
	session := newSession(stored, editor)
	other = NewSession(feature, stored.update, stored.load, nil)
	runAll(t, session, startLines...)

	runAll(t, session, "edit")

	if len(stored.value.Tasks) != 2 || stored.value.Tasks[0].Files[0].Items[0].Name != "score_hit" {
		t.Errorf("sketch = %+v, want the edit and the other writer's task", stored.value)
	}
}

func TestQuit(t *testing.T) {
	for _, line := range []string{"exit", " quit "} {
		t.Run(line, func(t *testing.T) {
			if _, err := newSession(&memory{value: sketch.New()}, nil).Run(line); !errors.Is(err, ErrQuit) {
				t.Errorf("Run(%q) = %v, want %v", line, err, ErrQuit)
			}
		})
	}
}

func TestUndo(t *testing.T) {
	stored := &memory{value: sketch.New()}
	session := newSession(stored, nil)
	runAll(t, session, startLines...)
	afterStart := stored.value
	runAll(t, session, "..", "t Camera_Move", "a cut or ease")

	output, err := session.Run("undo")
	if err != nil || output != undoneMessage {
		t.Fatalf("Run(undo) = %q, %v", output, err)
	}
	if session.At.String() != atFile+":Camera_Move" {
		t.Errorf("after one undo, context = %q, want where the undone edit was made", session.At)
	}
	runAll(t, session, "undo")

	if !reflect.DeepEqual(stored.value, afterStart) {
		t.Errorf("after two undos, sketch = %+v, want %+v", stored.value, afterStart)
	}
	if session.At.String() != atFile {
		t.Errorf("after two undos, context = %q, want %q", session.At, atFile)
	}
	runAll(t, session, "undo", "undo", "undo", "undo")
	if _, err := session.Run("undo"); !errors.Is(err, ErrNothingToUndo) {
		t.Errorf("undo past the session's start = %v, want %v", err, ErrNothingToUndo)
	}
	if len(stored.value.Tasks) != 0 || session.At.String() != feature {
		t.Errorf("undoing every edit left %+v at %q, want an empty sketch at the feature", stored.value, session.At)
	}
}

func TestEditsLandOnTheStoredSketch(t *testing.T) {
	stored := &memory{value: sketch.New()}
	session := newSession(stored, nil)
	runAll(t, session, startLines...)
	stored.value.Tasks = append(stored.value.Tasks, sketch.Task{Name: "frame-kills"})

	runAll(t, session, "out Camera_Move")

	if len(stored.value.Tasks) != 2 || len(stored.value.Tasks[0].Files[0].Items[0].Lines) != 2 {
		t.Errorf("sketch = %+v, want the other process's task and the session's new line", stored.value)
	}
}

func TestEdit(t *testing.T) {
	const opened = "type: fn\nname: score_strike\nlines:\n  - type: in\n    text: actor Combatant_Id\n"
	tests := []struct {
		name        string
		saved       string
		expectedErr error
		expectedAt  string
		lines       int
	}{
		{"An added line is kept", opened + "  - type: ask\n    text: who anchors a kill\n", nil, atItem, 2},
		{"A rename moves the context", "type: fn\nname: score_hit\n", nil, atFile + ":score_hit", 0},
		{"An unknown key is rejected", opened + "owner: eric\n", ErrEditRejected, atItem, 1},
		{"An unknown line type is rejected", "type: fn\nname: score_strike\nlines:\n  - type: returns\n    text: x\n",
			ErrEditRejected, atItem, 1},
		{"Broken YAML is rejected", "type: [fn\n", ErrEditRejected, atItem, 1},
		{"A rename onto a sibling is rejected", "type: type\nname: Camera_Move\n", ErrEditRejected, atItem, 1},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			stored := &memory{value: sketch.New()}
			var shown string
			session := newSession(stored, editorReturning(tt.saved, &shown))
			runAll(t, session, append(startLines, "..", "t Camera_Move", "f score_strike")...)
			before := stored.value

			_, err := session.Run("edit")

			if !errors.Is(err, tt.expectedErr) {
				t.Fatalf("Run(edit) = %v, want %v", err, tt.expectedErr)
			}
			if shown != opened {
				t.Errorf("the editor opened\n%s\nwant\n%s", shown, opened)
			}
			if session.At.String() != tt.expectedAt {
				t.Errorf("context = %q, want %q", session.At, tt.expectedAt)
			}
			if lines := len(stored.value.Tasks[0].Files[0].Items[0].Lines); lines != tt.lines {
				t.Errorf("the item has %d lines, want %d", lines, tt.lines)
			}
			if tt.expectedErr != nil && !reflect.DeepEqual(stored.value, before) {
				t.Errorf("a rejected edit changed the sketch")
			}
		})
	}
}

func TestEditCanBeUndone(t *testing.T) {
	stored := &memory{value: sketch.New()}
	var shown string
	session := newSession(stored, editorReturning("type: fn\nname: score_hit\n", &shown))
	runAll(t, session, startLines...)
	before := stored.value

	runAll(t, session, "edit", "undo")

	if !reflect.DeepEqual(stored.value, before) || session.At.String() != atItem {
		t.Errorf("after undoing an edit, sketch = %+v at %q, want %+v at %q", stored.value, session.At, before, atItem)
	}
}

func TestEditOfAnItemThatWent(t *testing.T) {
	stored := &memory{value: sketch.New()}
	var shown string
	session := newSession(stored, editorReturning("type: fn\nname: score_strike\n", &shown))
	runAll(t, session, startLines...)
	stored.value.Tasks[0].Files[0].Items = nil

	if _, err := session.Run("edit"); !errors.Is(err, ErrItemGone) {
		t.Errorf("Run(edit) = %v, want %v", err, ErrItemGone)
	}
}

func TestComplete(t *testing.T) {
	root := t.TempDir()
	for _, dir := range []string{"client/fx", ".git"} {
		if err := os.MkdirAll(filepath.Join(root, filepath.FromSlash(dir)), 0o755); err != nil {
			t.Fatal(err)
		}
	}
	for _, file := range []string{"client/camera.odin", "client/cast.odin", "main.go"} {
		if err := os.WriteFile(filepath.Join(root, filepath.FromSlash(file)), nil, 0o644); err != nil {
			t.Fatal(err)
		}
	}
	tests := []struct {
		name     string
		lines    []string
		line     string
		expected []string
		head     string
	}{
		{"Verbs by prefix, in Verbs order", nil, "fi", []string{"file ", "field "}, ""},
		{"Builder commands too", nil, "un", []string{"undo "}, ""},
		{"tix tasks, then the sketch's own", startLines, "task ", []string{"score-strikes", "frame-kills"}, "task "},
		{"Tasks by prefix", nil, "task fr", []string{"frame-kills"}, "task "},
		{"Repo paths at the root, without hidden ones", nil, "file ", []string{"client/", "main.go"}, "file "},
		{"Repo paths inside a directory", nil, "file client/ca", []string{"client/camera.odin", "client/cast.odin"},
			"file "},
		{"A sketched file that doesn't exist yet", []string{"task t", "file client/hud.odin"}, "file client/h",
			[]string{"client/hud.odin"}, "file "},
		{"Item names after an item verb", append(startLines, "..", "t Camera_Move", ".."), "f ",
			[]string{"score_strike", "Camera_Move"}, "f "},
		{"No item names for a rule line inside an item", startLines, "rule ", nil, "rule "},
		{"Item names for a rule item at file level", append(startLines, ".."), "rule s", []string{"score_strike"}, "rule "},
		{"Nothing after a line verb", startLines, "in ", nil, "in "},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			stored := &memory{value: sketch.New()}
			session := newSession(stored, nil)
			runAll(t, session, tt.lines...)
			completer := Completer{Root: root, Session: session, Load: stored.load, Tasks: []string{"frame-kills"}}
			if tt.name == "tix tasks, then the sketch's own" {
				completer.Tasks = []string{"score-strikes", "frame-kills"}
			}

			head, candidates, tail := completer.Complete(tt.line+" after", len(tt.line))

			if !reflect.DeepEqual(candidates, tt.expected) {
				t.Errorf("Complete(%q) = %q, want %q", tt.line, candidates, tt.expected)
			}
			if head != tt.head || tail != " after" {
				t.Errorf("Complete(%q) head, tail = %q, %q, want %q, %q", tt.line, head, tail, tt.head, " after")
			}
		})
	}
}
