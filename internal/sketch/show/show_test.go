package show

import (
	"errors"
	"strings"
	"testing"

	"github.com/theroymind/go-tix/contracts/sketch"
)

func fixture() sketch.Sketch {
	value := sketch.New()
	value.Tasks = []sketch.Task{
		{Name: "score-strikes", Files: []sketch.File{
			{Path: "client/camera.odin", Items: []sketch.Item{
				{Type: sketch.ItemTypeFn, Name: "score_strike", Lines: []sketch.Line{
					{Type: sketch.LineTypeIn, Text: "actor Combatant_Id"},
					{Type: sketch.LineTypeAsk, Text: "who anchors a kill", Answer: "the striker"},
				}},
				{Type: sketch.ItemTypeType, Name: "Camera_Move"},
			}},
			{Path: "client/arena.odin"},
		}},
		{Name: "frame-kills"},
	}
	return value
}

// changedInCamera marks Camera_Move changed, as if client/camera.odin already declared it, and the rest new.
func changedInCamera(path, name string) (string, error) {
	if path == "client/camera.odin" && name == "Camera_Move" {
		return "~", nil
	}
	return "+", nil
}

var errUnreadable = errors.New("unreadable")

func failingMarker(string, string) (string, error) { return "", errUnreadable }

func TestTable(t *testing.T) {
	tests := []struct {
		name     string
		value    sketch.Sketch
		expected string
	}{
		{"Every level, with gaps", fixture(), `TASK           FILE                MARK  TYPE  ITEM          LINES
score-strikes  client/camera.odin  +     fn    score_strike  2
score-strikes  client/camera.odin  ~     type  Camera_Move   0
score-strikes  client/arena.odin   -     -     -             -
frame-kills    -                   -     -     -             -
`},
		{"An empty sketch", sketch.New(), "TASK  FILE  MARK  TYPE  ITEM  LINES\n"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var output strings.Builder

			if err := Table(&output, tt.value, changedInCamera); err != nil {
				t.Fatal(err)
			}

			if output.String() != tt.expected {
				t.Errorf("Table() =\n%s\nwant\n%s", output.String(), tt.expected)
			}
		})
	}
}

func TestMarkdown(t *testing.T) {
	tests := []struct {
		name     string
		value    sketch.Sketch
		expected string
	}{
		{"Every level, with gaps", fixture(), "# fight-camera\n\n## score-strikes\n\n### `client/camera.odin`\n\n" +
			"- + **fn** `score_strike`\n  1. in: actor Combatant_Id\n  2. ask: who anchors a kill\n" +
			"     answer: the striker\n- ~ **type** `Camera_Move`\n" +
			"\n### `client/arena.odin`\n\n## frame-kills\n"},
		{"An empty sketch", sketch.New(), "# fight-camera\n"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var output strings.Builder

			if err := Markdown(&output, "fight-camera", tt.value, changedInCamera); err != nil {
				t.Fatal(err)
			}

			if output.String() != tt.expected {
				t.Errorf("Markdown() =\n%s\nwant\n%s", output.String(), tt.expected)
			}
		})
	}
}

func TestAMarkerErrorStopsTheOutput(t *testing.T) {
	tests := []struct {
		name  string
		write func(*strings.Builder) error
	}{
		{"Table", func(output *strings.Builder) error { return Table(output, fixture(), failingMarker) }},
		{"Markdown", func(output *strings.Builder) error {
			return Markdown(output, "fight-camera", fixture(), failingMarker)
		}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var output strings.Builder

			err := tt.write(&output)

			if !errors.Is(err, errUnreadable) || output.Len() != 0 {
				t.Errorf("= %v with %q written, want %v and nothing written", err, output.String(), errUnreadable)
			}
		})
	}
}
