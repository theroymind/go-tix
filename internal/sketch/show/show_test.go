package show

import (
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
					{Type: sketch.LineTypeAsk, Text: "who anchors a kill"},
				}},
				{Type: sketch.ItemTypeType, Name: "Camera_Move"},
			}},
			{Path: "client/arena.odin"},
		}},
		{Name: "frame-kills"},
	}
	return value
}

func TestTable(t *testing.T) {
	tests := []struct {
		name     string
		value    sketch.Sketch
		expected string
	}{
		{"Every level, with gaps", fixture(), `TASK           FILE                TYPE  ITEM          LINES
score-strikes  client/camera.odin  fn    score_strike  2
score-strikes  client/camera.odin  type  Camera_Move   0
score-strikes  client/arena.odin   -     -             -
frame-kills    -                   -     -             -
`},
		{"An empty sketch", sketch.New(), "TASK  FILE  TYPE  ITEM  LINES\n"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var output strings.Builder

			if err := Table(&output, tt.value); err != nil {
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
			"- **fn** `score_strike`\n  1. in: actor Combatant_Id\n  2. ask: who anchors a kill\n- **type** `Camera_Move`\n" +
			"\n### `client/arena.odin`\n\n## frame-kills\n"},
		{"An empty sketch", sketch.New(), "# fight-camera\n"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var output strings.Builder

			if err := Markdown(&output, "fight-camera", tt.value); err != nil {
				t.Fatal(err)
			}

			if output.String() != tt.expected {
				t.Errorf("Markdown() =\n%s\nwant\n%s", output.String(), tt.expected)
			}
		})
	}
}
