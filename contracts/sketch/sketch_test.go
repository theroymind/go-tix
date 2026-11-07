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
		Status:        StatusReady,
		Body:          "Orbit the camera.\n\tfn useOrbit(target)\n",
		Edges: []Edge{{To: "src/camera.ts", Label: "reads the target", Mark: MarkChanged},
			{To: "src/speed.ts", Mark: MarkNew}, {To: "src/gone.ts"}},
		Questions: []Question{{Text: "who owns the camera?", Line: 7}, {Text: "how fast?", Answer: "1 rad/s\nmaybe 2", Line: 9}},
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
			value.Mark, value.Feature, value.Status, value.Body, value.Edges, value.Questions = "", "", "", "", nil, nil
		}, nil},
		{"An unknown note mark", func(value *Note) { value.Mark = "+" }, []string{FieldMark}},
		{"A draft note", func(value *Note) { value.Status = StatusDraft }, nil},
		{"An unknown status", func(value *Note) { value.Status = "done" }, []string{FieldStatus}},
		{"A status in the wrong case", func(value *Note) { value.Status = "Ready" }, []string{FieldStatus}},
		{"A body with control characters", func(value *Note) { value.Body = "a\tb\r\n\x1b" }, nil},
		{"An unknown schema version", func(value *Note) { value.SchemaVersion = 1 }, []string{FieldSchemaVersion}},
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
		{"An edge with no path", func(value *Note) { value.Edges[0].To = "" }, []string{"edges.0.to"}},
		{"An edge that isn't clean", func(value *Note) { value.Edges[1].To = "src/./speed.ts" }, []string{"edges.1.to"}},
		{"An edge escaping the repo", func(value *Note) { value.Edges[0].To = "../camera.ts" }, []string{"edges.0.to"}},
		{"An absolute edge", func(value *Note) { value.Edges[0].To = "/src/camera.ts" }, []string{"edges.0.to"}},
		{"A control character in an edge label", func(value *Note) { value.Edges[0].Label = "a\tb" },
			[]string{"edges.0.label"}},
		{"An unknown edge mark", func(value *Note) { value.Edges[2].Mark = "!" }, []string{"edges.2.mark"}},
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

func TestStatusIsValid(t *testing.T) {
	tests := []struct {
		name     string
		status   Status
		expected bool
	}{
		{"Draft", StatusDraft, true},
		{"Ready", StatusReady, true},
		{"Unset", "", true},
		{"Unknown", "done", false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if actual := tt.status.IsValid(); actual != tt.expected {
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
