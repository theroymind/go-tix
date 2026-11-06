package grammar

import (
	"encoding/json"
	"errors"
	"reflect"
	"strings"
	"testing"

	"github.com/theroymind/go-tix/contracts/sketch"
)

const (
	feature  = "fight-camera"
	atTask   = feature + "/score-strikes"
	atFile   = atTask + "/client/camera.odin"
	atItem   = atFile + ":score_strike"
	camera   = "client/camera.odin"
	strike   = "score_strike"
	taskName = "score-strikes"
)

// base is the sketch most cases start from: one task, one file, an fn with two lines, and a type with an answered
// question.
func base() sketch.Sketch {
	value := sketch.New()
	value.Tasks = []sketch.Task{{Name: taskName, Files: []sketch.File{{Path: camera, Items: []sketch.Item{
		{Type: sketch.ItemTypeFn, Name: strike, Lines: []sketch.Line{
			{Type: sketch.LineTypeIn, Text: "actor Combatant_Id"},
			{Type: sketch.LineTypeOut, Text: "Camera_Move"},
		}},
		{Type: sketch.ItemTypeType, Name: "Camera_Move", Lines: []sketch.Line{
			{Type: sketch.LineTypeAsk, Text: "who anchors a kill", Answer: "the striker"},
		}},
	}}}}}
	return value
}

func item(value *sketch.Sketch) *sketch.Item { return &value.Tasks[0].Files[0].Items[0] }

func TestApply(t *testing.T) {
	tests := []struct {
		name        string
		context     string
		line        string
		change      func(value *sketch.Sketch)
		nextContext string
	}{
		{"task adds a task and opens it", feature, "task frame-kills", func(value *sketch.Sketch) {
			value.Tasks = append(value.Tasks, sketch.Task{Name: "frame-kills"})
		}, feature + "/frame-kills"},
		{"task opens an existing task from deeper", atItem, "task " + taskName, func(*sketch.Sketch) {}, atTask},
		{"file adds a file to the task", atTask, "file client/hud.odin", func(value *sketch.Sketch) {
			value.Tasks[0].Files = append(value.Tasks[0].Files, sketch.File{Path: "client/hud.odin"})
		}, atTask + "/client/hud.odin"},
		{"fn adds an item and opens it", atFile, "fn frame_kill", func(value *sketch.Sketch) {
			value.Tasks[0].Files[0].Items = append(value.Tasks[0].Files[0].Items,
				sketch.Item{Type: sketch.ItemTypeFn, Name: "frame_kill"})
		}, atFile + ":frame_kill"},
		{"An item verb inside an item adds a sibling", atItem, "t Camera_Cut", func(value *sketch.Sketch) {
			value.Tasks[0].Files[0].Items = append(value.Tasks[0].Files[0].Items,
				sketch.Item{Type: sketch.ItemTypeType, Name: "Camera_Cut"})
		}, atFile + ":Camera_Cut"},
		{"An existing item of the same type opens it", atFile, "f " + strike, func(*sketch.Sketch) {}, atItem},
		{"An item verb creates the task and file it needs", feature + "/frame-kills/client/kill.odin", "test frames a kill",
			func(value *sketch.Sketch) {
				value.Tasks = append(value.Tasks, sketch.Task{Name: "frame-kills", Files: []sketch.File{{
					Path: "client/kill.odin", Items: []sketch.Item{{Type: sketch.ItemTypeTest, Name: "frames a kill"}}}}})
			}, feature + "/frame-kills/client/kill.odin:frames a kill"},
		{"in adds a line, keeping its inner spaces", atItem, "  i   actor  Combatant_Id  ", func(value *sketch.Sketch) {
			item(value).Lines = append(item(value).Lines, sketch.Line{Type: sketch.LineTypeIn, Text: "actor  Combatant_Id"})
		}, atItem},
		{"ask adds a question", atItem, "a who anchors a kill", func(value *sketch.Sketch) {
			item(value).Lines = append(item(value).Lines, sketch.Line{Type: sketch.LineTypeAsk, Text: "who anchors a kill"})
		}, atItem},
		{"rule inside an item adds a line", atItem, "r a kill always frames", func(value *sketch.Sketch) {
			item(value).Lines = append(item(value).Lines, sketch.Line{Type: sketch.LineTypeRule, Text: "a kill always frames"})
		}, atItem},
		{"rule at file level adds an item", atFile, "rule kills_frame", func(value *sketch.Sketch) {
			value.Tasks[0].Files[0].Items = append(value.Tasks[0].Files[0].Items,
				sketch.Item{Type: sketch.ItemTypeRule, Name: "kills_frame"})
		}, atFile + ":kills_frame"},
		{"mv renames the item", atItem, "mv score_hit", func(value *sketch.Sketch) { item(value).Name = "score_hit" },
			atFile + ":score_hit"},
		{"mv renames the file", atFile, "mv client/cam.odin", func(value *sketch.Sketch) {
			value.Tasks[0].Files[0].Path = "client/cam.odin"
		}, atTask + "/client/cam.odin"},
		{"mv renames the task", atTask, "mv strikes", func(value *sketch.Sketch) { value.Tasks[0].Name = "strikes" },
			feature + "/strikes"},
		{"rm N removes that line", atItem, "rm 1", func(value *sketch.Sketch) {
			item(value).Lines = item(value).Lines[1:]
		}, atItem},
		{"rm removes the item and steps up", atItem, "rm", func(value *sketch.Sketch) {
			value.Tasks[0].Files[0].Items = value.Tasks[0].Files[0].Items[1:]
		}, atFile},
		{"rm removes the file", atFile, "rm", func(value *sketch.Sketch) { value.Tasks[0].Files = nil }, atTask},
		{"rm removes the task", atTask, "rm", func(value *sketch.Sketch) { value.Tasks = nil }, feature},
		{"set rewords an answered question and opens it again", atFile + ":Camera_Move", "set 1 who frames a kill",
			func(value *sketch.Sketch) {
				value.Tasks[0].Files[0].Items[1].Lines[0] = sketch.Line{Type: sketch.LineTypeAsk, Text: "who frames a kill"}
			}, atFile + ":Camera_Move"},
		{"set replaces a line's text and keeps its type", atItem, "set 2 Camera_Cut", func(value *sketch.Sketch) {
			item(value).Lines[1].Text = "Camera_Cut"
		}, atItem},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			at := mustContext(t, tt.context)
			original := base()
			expected := base()
			tt.change(&expected)

			edited, next, err := Apply(original, at, tt.line)

			if err != nil {
				t.Fatalf("Apply() = %v", err)
			}
			if !sameSketch(t, edited, expected) {
				t.Errorf("Apply() sketch = %+v, want %+v", edited.Tasks, expected.Tasks)
			}
			if next.String() != tt.nextContext {
				t.Errorf("Apply() context = %q, want %q", next, tt.nextContext)
			}
			if !reflect.DeepEqual(original, base()) {
				t.Errorf("Apply() changed the sketch it was given")
			}
		})
	}
}

func TestApplyRejects(t *testing.T) {
	tests := []struct {
		name        string
		context     string
		line        string
		expected    error
		messageHas  string
		emptySketch bool
	}{
		{"An empty line", atItem, "   ", ErrEmptyLine, "", false},
		{"An unknown verb", atItem, "struct Foo", ErrUnknownVerb, `"struct"`, false},
		{"A line verb at file level", atFile, "in actor Combatant_Id", ErrWrongContext,
			"in needs an item, and the context is the file " + atFile, false},
		{"A line verb at task level", atTask, "a who anchors", ErrWrongContext, "ask needs an item", false},
		{"An item verb at task level", atTask, "fn score_strike", ErrWrongContext,
			"fn needs a file, and the context is the task " + atTask, false},
		{"An item verb at feature level", feature, "type Camera_Move", ErrWrongContext, "the feature " + feature, false},
		{"file at feature level", feature, "file client/camera.odin", ErrWrongContext, "file needs a task", false},
		{"A verb with no name", atFile, "fn", ErrMissingArgument, "fn needs an item name", false},
		{"A line with no text", atItem, "out ", ErrMissingArgument, "out needs its text", false},
		{"A line into an item the sketch lacks", atFile + ":frame_kill", "in actor", ErrNotFound, "no item", false},
		{"A line into an empty sketch", atItem, "in actor", ErrNotFound, "no task", true},
		{"An item name of another type", atFile, "type " + strike, ErrTypeConflict, "score_strike is a fn", false},
		{"rm of a line number out of range", atItem, "rm 3", ErrBadLineNumber, "has 2 lines", false},
		{"rm of line 0", atItem, "rm 0", ErrBadLineNumber, "", false},
		{"rm of a line number that isn't one", atItem, "rm two", ErrBadLineNumber, "", false},
		{"rm N outside an item", atFile, "rm 1", ErrWrongContext, "needs an item", false},
		{"rm at feature level", feature, "rm", ErrWrongContext, "rm needs a task", false},
		{"rm of an item the sketch lacks", atFile + ":frame_kill", "rm", ErrNotFound, "", false},
		{"set without text", atItem, "set 2", ErrMissingArgument, "", false},
		{"mv at feature level", feature, "mv other", ErrWrongContext, "", false},
		{"A path that leaves the repo", atTask, "file ../camera.odin", ErrInvalidSketch, "tasks.0.files.1.path", false},
		{"A tab inside a path", atTask, "file client/ca\tmera.odin", ErrInvalidSketch, "control character", false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			at := mustContext(t, tt.context)
			start := base()
			if tt.emptySketch {
				start = sketch.New()
			}

			edited, next, err := Apply(start, at, tt.line)

			if !errors.Is(err, tt.expected) {
				t.Fatalf("Apply() = %v, want an error wrapping %v", err, tt.expected)
			}
			if !strings.Contains(err.Error(), tt.messageHas) {
				t.Errorf("Apply() = %q, want it to say %q", err, tt.messageHas)
			}
			if !reflect.DeepEqual(edited, start) || next != at {
				t.Errorf("Apply() = %+v at %q, want the sketch and context unchanged", edited, next)
			}
		})
	}
}

func TestApplyRejectsADuplicateRename(t *testing.T) {
	start, next, err := Apply(base(), mustContext(t, atFile), "fn frame_kill")
	if err != nil {
		t.Fatal(err)
	}

	_, _, err = Apply(start, next, "mv "+strike)

	if !errors.Is(err, ErrInvalidSketch) || !strings.Contains(err.Error(), "duplicates an earlier item name") {
		t.Errorf("Apply() = %v, want an error wrapping %v about the duplicate", err, ErrInvalidSketch)
	}
}

func mustContext(t *testing.T, ref string) Context {
	t.Helper()
	at, err := ParseContext(ref)
	if err != nil {
		t.Fatal(err)
	}
	return at
}

// sameSketch compares two sketches as their JSON, where an empty list and no list are the same.
func sameSketch(t *testing.T, actual, expected sketch.Sketch) bool {
	t.Helper()
	actualJSON, err := json.Marshal(actual)
	if err != nil {
		t.Fatal(err)
	}
	expectedJSON, err := json.Marshal(expected)
	if err != nil {
		t.Fatal(err)
	}
	return string(actualJSON) == string(expectedJSON)
}

func TestParseContext(t *testing.T) {
	tests := []struct {
		name     string
		ref      string
		expected Context
		err      error
	}{
		{"A feature", feature, Context{Feature: feature}, nil},
		{"A task", atTask, Context{Feature: feature, Task: taskName}, nil},
		{"A file keeps its slashes", atFile, Context{Feature: feature, Task: taskName, File: camera}, nil},
		{"An item", atItem, Context{Feature: feature, Task: taskName, File: camera, Item: strike}, nil},
		{"An item name holding colons", atFile + ":Camera::cut",
			Context{Feature: feature, Task: taskName, File: camera, Item: "Camera::cut"}, nil},
		{"A file at the repo root", atTask + "/main.go", Context{Feature: feature, Task: taskName, File: "main.go"}, nil},
		{"An empty ref", "", Context{}, ErrBadContext},
		{"A parent-directory feature", "../task", Context{}, ErrBadContext},
		{"An empty task", feature + "/", Context{}, ErrBadContext},
		{"An empty file", atTask + "/", Context{}, ErrBadContext},
		{"An empty item", atFile + ":", Context{}, ErrBadContext},
		{"An item with no file", atTask + "/:score_strike", Context{}, ErrBadContext},
		{"A tab in the feature", "fight\tcamera", Context{}, ErrBadContext},
		{"A tab in the file", atTask + "/client/ca\tmera.odin", Context{}, ErrBadContext},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			actual, err := ParseContext(tt.ref)

			if !errors.Is(err, tt.err) {
				t.Fatalf("ParseContext() = %v, want %v", err, tt.err)
			}
			if actual != tt.expected {
				t.Errorf("ParseContext() = %+v, want %+v", actual, tt.expected)
			}
			if tt.err == nil && actual.String() != tt.ref {
				t.Errorf("String() = %q, want %q", actual.String(), tt.ref)
			}
		})
	}
}

func TestEveryVerbAndAlias(t *testing.T) {
	type spelling struct {
		verb     string
		itemType sketch.ItemType
		lineType sketch.LineType
	}
	var spellings []spelling
	for _, itemType := range sketch.ItemTypes {
		spellings = append(spellings, spelling{verb: string(itemType), itemType: itemType})
	}
	for _, lineType := range sketch.LineTypes {
		spellings = append(spellings, spelling{verb: string(lineType), lineType: lineType})
	}
	for alias, full := range aliases {
		if lineType := sketch.LineType(full); lineType.IsValid() {
			spellings = append(spellings, spelling{verb: alias, lineType: lineType})
		} else {
			spellings = append(spellings, spelling{verb: alias, itemType: sketch.ItemType(full)})
		}
	}
	for _, tt := range spellings {
		t.Run(tt.verb, func(t *testing.T) {
			context := atFile
			if tt.lineType != "" {
				context = atItem
			}

			edited, _, err := Apply(base(), mustContext(t, context), tt.verb+" added")

			if err != nil {
				t.Fatalf("Apply() = %v", err)
			}
			file := edited.Tasks[0].Files[0]
			if tt.lineType != "" {
				lines := file.Items[0].Lines
				if last := lines[len(lines)-1]; last != (sketch.Line{Type: tt.lineType, Text: "added"}) {
					t.Errorf("last line = %+v, want a %s line", last, tt.lineType)
				}
				return
			}
			if last := file.Items[len(file.Items)-1]; last.Type != tt.itemType || last.Name != "added" {
				t.Errorf("last item = %+v, want a %s item", last, tt.itemType)
			}
		})
	}
}

func TestVerbs(t *testing.T) {
	verbs := Verbs()
	seen := map[string]bool{}
	for _, verb := range verbs {
		t.Run(verb, func(t *testing.T) {
			if seen[verb] {
				t.Errorf("Verbs() lists %q twice", verb)
			}
			seen[verb] = true

			_, _, err := Apply(base(), mustContext(t, atItem), verb+" 1 added")

			if errors.Is(err, ErrUnknownVerb) {
				t.Errorf("Apply() doesn't accept %q, which Verbs() lists", verb)
			}
		})
	}
	expectedCount := 2 + len(sketch.ItemTypes) + len(sketch.LineTypes) - 1 + 3 + len(aliases)
	if len(verbs) != expectedCount {
		t.Errorf("Verbs() lists %d spellings, want %d", len(verbs), expectedCount)
	}
}

func TestParent(t *testing.T) {
	tests := []struct {
		name     string
		ref      string
		expected string
	}{
		{"An item's file", atItem, atFile},
		{"A file's task", atFile, atTask},
		{"A task's feature", atTask, feature},
		{"A feature is its own parent", feature, feature},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if actual := mustContext(t, tt.ref).Parent().String(); actual != tt.expected {
				t.Errorf("Parent() = %q, want %q", actual, tt.expected)
			}
		})
	}
}
