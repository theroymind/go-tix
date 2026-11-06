package sketch

import (
	"errors"
	"reflect"
	"slices"
	"testing"

	"github.com/theroymind/go-tix/contracts/validation"
)

func validSketch() Sketch {
	value := New()
	value.Tasks = []Task{{
		Name: "score-strikes",
		Files: []File{{
			Path: "client/camera.odin",
			Items: []Item{
				{Type: ItemTypeFn, Name: "score_strike", Lines: []Line{
					{Type: LineTypeIn, Text: "actor Combatant_Id"},
					{Type: LineTypeOut, Text: "Camera_Move"},
					{Type: LineTypeAsk, Text: "who anchors a kill"},
				}},
				{Type: ItemTypeType, Name: "Camera_Move"},
			},
		}},
	}}
	return value
}

func TestSketchValidate(t *testing.T) {
	tests := []struct {
		name           string
		change         func(value *Sketch)
		expectedFields []string
	}{
		{"A sketch with every level", func(*Sketch) {}, nil},
		{"An empty sketch", func(value *Sketch) { value.Tasks = nil }, nil},
		{"An unknown schema version", func(value *Sketch) { value.SchemaVersion = 2 }, []string{FieldSchemaVersion}},
		{"A missing schema version", func(value *Sketch) { value.SchemaVersion = 0 }, []string{FieldSchemaVersion}},
		{"An empty task name", func(value *Sketch) { value.Tasks[0].Name = "" }, []string{"tasks.0.name"}},
		{"A duplicate task name", func(value *Sketch) { value.Tasks = append(value.Tasks, Task{Name: "score-strikes"}) },
			[]string{"tasks.1.name"}},
		{"Two empty task names report only as empty", func(value *Sketch) {
			value.Tasks = []Task{{}, {}}
		}, []string{"tasks.0.name", "tasks.1.name"}},
		{"An empty file path", func(value *Sketch) { value.Tasks[0].Files[0].Path = "" },
			[]string{"tasks.0.files.0.path"}},
		{"An absolute file path", func(value *Sketch) { value.Tasks[0].Files[0].Path = "/client/camera.odin" },
			[]string{"tasks.0.files.0.path"}},
		{"A path that isn't clean", func(value *Sketch) { value.Tasks[0].Files[0].Path = "./client/camera.odin" },
			[]string{"tasks.0.files.0.path"}},
		{"A path with a doubled slash", func(value *Sketch) { value.Tasks[0].Files[0].Path = "client//camera.odin" },
			[]string{"tasks.0.files.0.path"}},
		{"A path outside the repo", func(value *Sketch) { value.Tasks[0].Files[0].Path = "../camera.odin" },
			[]string{"tasks.0.files.0.path"}},
		{"The parent directory", func(value *Sketch) { value.Tasks[0].Files[0].Path = ".." },
			[]string{"tasks.0.files.0.path"}},
		{"A name starting with two dots", func(value *Sketch) { value.Tasks[0].Files[0].Path = "..camera.odin" }, nil},
		{"A padded path is checked trimmed", func(value *Sketch) { value.Tasks[0].Files[0].Path = " client/camera.odin " },
			nil},
		{"A duplicate file path in one task", func(value *Sketch) {
			value.Tasks[0].Files = append(value.Tasks[0].Files, File{Path: "client/camera.odin"})
		}, []string{"tasks.0.files.1.path"}},
		{"The same file path in two tasks", func(value *Sketch) {
			value.Tasks = append(value.Tasks, Task{Name: "frame-kills", Files: []File{{Path: "client/camera.odin"}}})
		}, nil},
		{"An unknown item type", func(value *Sketch) { value.Tasks[0].Files[0].Items[0].Type = "struct" },
			[]string{"tasks.0.files.0.items.0.type"}},
		{"An empty item name", func(value *Sketch) { value.Tasks[0].Files[0].Items[1].Name = "" },
			[]string{"tasks.0.files.0.items.1.name"}},
		{"A duplicate item name in one file", func(value *Sketch) {
			value.Tasks[0].Files[0].Items[1].Name = "score_strike"
		}, []string{"tasks.0.files.0.items.1.name"}},
		{"A duplicate item name differing only in spaces", func(value *Sketch) {
			value.Tasks[0].Files[0].Items[1].Name = "score_strike "
		}, []string{"tasks.0.files.0.items.1.name"}},
		{"The same item name in two files", func(value *Sketch) {
			value.Tasks[0].Files = append(value.Tasks[0].Files,
				File{Path: "client/hud.odin", Items: []Item{{Type: ItemTypeFn, Name: "score_strike"}}})
		}, nil},
		{"An unknown line type", func(value *Sketch) { value.Tasks[0].Files[0].Items[0].Lines[1].Type = "returns" },
			[]string{"tasks.0.files.0.items.0.lines.1.type"}},
		{"An answered question", func(value *Sketch) {
			value.Tasks[0].Files[0].Items[0].Lines[2].Answer = "the striker"
		}, nil},
		{"An answer on a line that isn't a question", func(value *Sketch) {
			value.Tasks[0].Files[0].Items[0].Lines[0].Answer = "the striker"
		}, []string{"tasks.0.files.0.items.0.lines.0.answer"}},
		{"A tab inside a path", func(value *Sketch) { value.Tasks[0].Files[0].Path = "client/ca\tmera.odin" },
			[]string{"tasks.0.files.0.path"}},
		{"A newline inside an item name", func(value *Sketch) { value.Tasks[0].Files[0].Items[0].Name = "score\nstrike" },
			[]string{"tasks.0.files.0.items.0.name"}},
		{"An escape inside a task name", func(value *Sketch) { value.Tasks[0].Name = "score\x1b[2Jstrikes" },
			[]string{"tasks.0.name"}},
		{"A tab inside a line's text", func(value *Sketch) { value.Tasks[0].Files[0].Items[0].Lines[0].Text = "a\tb" },
			[]string{"tasks.0.files.0.items.0.lines.0.text"}},
		{"A newline inside an answer", func(value *Sketch) {
			value.Tasks[0].Files[0].Items[0].Lines[2].Answer = "the\nstriker"
		}, []string{"tasks.0.files.0.items.0.lines.2.answer"}},
		{"A trailing newline is trimmed, not refused", func(value *Sketch) { value.Tasks[0].Name = "score-strikes\n" },
			nil},
		{"An empty line text", func(value *Sketch) { value.Tasks[0].Files[0].Items[0].Lines[2].Text = "" },
			[]string{"tasks.0.files.0.items.0.lines.2.text"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			value := validSketch()
			tt.change(&value)

			err := value.Validate()

			var problems validation.Errors
			errors.As(err, &problems)
			if !slices.Equal(problems.Fields(), tt.expectedFields) {
				t.Errorf("Validate() fields = %q, want %q (error %v)", problems.Fields(), tt.expectedFields, err)
			}
			if tt.expectedFields != nil && !errors.Is(err, validation.ErrInvalid) {
				t.Errorf("Validate() = %v, want an error wrapping %v", err, validation.ErrInvalid)
			}
		})
	}
}

func TestTrimmed(t *testing.T) {
	padded := validSketch()
	padded.Tasks[0].Name = " score-strikes"
	padded.Tasks[0].Files[0].Path = "client/camera.odin "
	padded.Tasks[0].Files[0].Items[1].Name = "\tCamera_Move\n"
	padded.Tasks[0].Files[0].Items[0].Lines[0].Text = "  actor Combatant_Id "
	padded.Tasks[0].Files[0].Items[0].Lines[2].Answer = " the striker "
	answered := validSketch()
	answered.Tasks[0].Files[0].Items[0].Lines[2].Answer = "the striker"
	tests := []struct {
		name     string
		value    Sketch
		expected Sketch
	}{
		{"Spaces around every name, path, text and answer", padded, answered},
		{"An already trimmed sketch", validSketch(), validSketch()},
		{"An empty sketch", New(), New()},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if actual := tt.value.Trimmed(); !reflect.DeepEqual(actual, tt.expected) {
				t.Errorf("Trimmed() = %+v, want %+v", actual, tt.expected)
			}
		})
	}
	if padded.Tasks[0].Files[0].Items[0].Lines[0].Text != "  actor Combatant_Id " {
		t.Error("Trimmed() changed the sketch it was called on")
	}
}

func TestLineIsOpen(t *testing.T) {
	tests := []struct {
		name     string
		line     Line
		expected bool
	}{
		{"An unanswered question", Line{Type: LineTypeAsk, Text: "who anchors a kill"}, true},
		{"A question answered with spaces only", Line{Type: LineTypeAsk, Text: "who", Answer: "  "}, true},
		{"An answered question", Line{Type: LineTypeAsk, Text: "who", Answer: "the striker"}, false},
		{"A line that isn't a question", Line{Type: LineTypeIn, Text: "actor Combatant_Id"}, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if actual := tt.line.IsOpen(); actual != tt.expected {
				t.Errorf("IsOpen() = %v, want %v", actual, tt.expected)
			}
		})
	}
}

func TestTypesAreValid(t *testing.T) {
	tests := []struct {
		name     string
		valid    bool
		expected bool
	}{
		{"Every item type", allValid(ItemTypes), true},
		{"Every line type", allValid(LineTypes), true},
		{"An empty item type", ItemType("").IsValid(), false},
		{"A line type in the wrong case", LineType("In").IsValid(), false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.valid != tt.expected {
				t.Errorf("IsValid() = %v, want %v", tt.valid, tt.expected)
			}
		})
	}
}

func allValid[Kind interface{ IsValid() bool }](kinds []Kind) bool {
	for _, kind := range kinds {
		if !kind.IsValid() {
			return false
		}
	}
	return true
}
