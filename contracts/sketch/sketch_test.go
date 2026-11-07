package sketch

import (
	"errors"
	"slices"
	"testing"

	"github.com/theroymind/go-tix/contracts/validation"
)

func validNote() Note {
	return Note{
		SchemaVersion: SchemaVersion,
		Path:          "src/useOrbit.ts.sketch.md",
		Target:        "src/useOrbit.ts",
		Mark:          MarkChanged,
		Feature:       "sketch-explorer",
		Task:          "read-notes",
		Items: []Item{{Type: ItemTypeFn, Name: "useOrbit", Text: "target Ref<Vec3>", Mark: MarkChanged},
			{Type: ItemTypeType, Name: "Orbit", Mark: MarkNew}},
		Questions: []Question{{Text: "who owns the camera?", Line: 7}, {Text: "how fast?", Answer: "1 rad/s", Line: 9}},
	}
}

func TestNoteValidate(t *testing.T) {
	tests := []struct {
		name           string
		change         func(value *Note)
		expectedFields []string
	}{
		{"A note with every part", func(*Note) {}, nil},
		{"A bare note", func(value *Note) {
			value.Mark, value.Feature, value.Task, value.Items, value.Questions = "", "", "", nil, nil
		}, nil},
		{"An unknown note mark", func(value *Note) { value.Mark = "+" }, []string{FieldMark}},
		{"An unknown item mark", func(value *Note) { value.Items[0].Mark = "~" }, []string{"items.0.mark"}},
		{"An unmarked item", func(value *Note) { value.Items[0].Mark = "" }, nil},
		{"A feature tix doesn't know", func(value *Note) { value.UnknownFeature = true }, nil},
		{"An unknown feature with no feature", func(value *Note) { value.UnknownFeature, value.Feature = true, "" },
			[]string{FieldUnknownFeature}},
		{"An unknown schema version", func(value *Note) { value.SchemaVersion = 2 }, []string{FieldSchemaVersion}},
		{"A missing schema version", func(value *Note) { value.SchemaVersion = 0 }, []string{FieldSchemaVersion}},
		{"A missing target", func(value *Note) { value.Target = "" }, []string{FieldTarget}},
		{"A missing path", func(value *Note) { value.Path = "" }, []string{FieldPath}},
		{"A path that isn't the target's note", func(value *Note) { value.Path = "src/other.ts.sketch.md" },
			[]string{FieldPath}},
		{"An absolute target", func(value *Note) {
			value.Target, value.Path = "/src/useOrbit.ts", "/src/useOrbit.ts.sketch.md"
		}, []string{FieldPath, FieldTarget}},
		{"A target outside the repo", func(value *Note) {
			value.Target, value.Path = "../useOrbit.ts", "../useOrbit.ts.sketch.md"
		}, []string{FieldPath, FieldTarget}},
		{"A target that isn't clean", func(value *Note) {
			value.Target, value.Path = "src//useOrbit.ts", "src//useOrbit.ts.sketch.md"
		}, []string{FieldPath, FieldTarget}},
		{"A tab in the feature", func(value *Note) { value.Feature = "sketch\texplorer" }, []string{FieldFeature}},
		{"An escape in the task", func(value *Note) { value.Task = "read\x1bnotes" }, []string{FieldTask}},
		{"An unknown item type", func(value *Note) { value.Items[1].Type = "struct" }, []string{"items.1.type"}},
		{"An empty item name", func(value *Note) { value.Items[0].Name = " " }, []string{"items.0.name"}},
		{"A control character in an item name", func(value *Note) { value.Items[0].Name = "use\x00Orbit" },
			[]string{"items.0.name"}},
		{"A repeated item name", func(value *Note) { value.Items[1].Name = "useOrbit" }, nil},
		{"An empty question", func(value *Note) { value.Questions[0].Text = "" }, []string{"questions.0.text"}},
		{"A question without a line number", func(value *Note) { value.Questions[1].Line = 0 },
			[]string{"questions.1.line"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			value := validNote()
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

func TestItemTypeIsValid(t *testing.T) {
	tests := []struct {
		name     string
		itemType ItemType
		expected bool
	}{
		{"fn", ItemTypeFn, true},
		{"test", ItemTypeTest, true},
		{"An empty type", "", false},
		{"The wrong case", "Fn", false},
		{"A tag that isn't a type", "todo", false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if actual := tt.itemType.IsValid(); actual != tt.expected {
				t.Errorf("IsValid() = %v, want %v", actual, tt.expected)
			}
		})
	}
}

func TestMarkSymbol(t *testing.T) {
	tests := []struct {
		name     string
		mark     Mark
		expected string
	}{
		{"New", MarkNew, "+"},
		{"Changed", MarkChanged, "~"},
		{"Unmarked", "", ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if actual := tt.mark.Symbol(); actual != tt.expected {
				t.Errorf("Symbol() = %q, want %q", actual, tt.expected)
			}
		})
	}
}
