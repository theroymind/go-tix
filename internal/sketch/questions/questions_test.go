package questions

import (
	"errors"
	"reflect"
	"testing"

	"github.com/theroymind/go-tix/contracts/sketch"
)

const feature = "fight-camera"

func fixture() sketch.Sketch {
	value := sketch.New()
	value.Tasks = []sketch.Task{
		{Name: "score-strikes", Files: []sketch.File{{Path: "client/camera.odin", Items: []sketch.Item{
			{Type: sketch.ItemTypeFn, Name: "score_strike", Lines: []sketch.Line{
				{Type: sketch.LineTypeAsk, Text: "who anchors a kill", Answer: "the striker"},
				{Type: sketch.LineTypeIn, Text: "actor Combatant_Id"},
				{Type: sketch.LineTypeAsk, Text: "does a counter reset it"},
			}},
		}}}},
		{Name: "frame-kills", Files: []sketch.File{{Path: "client/kill.odin", Items: []sketch.Item{
			{Type: sketch.ItemTypeTest, Name: "frames a kill", Lines: []sketch.Line{
				{Type: sketch.LineTypeAsk, Text: "cut or ease"},
			}},
		}}}},
	}
	return value
}

func TestOpen(t *testing.T) {
	tests := []struct {
		name     string
		value    sketch.Sketch
		expected []Question
	}{
		{"Numbers count answered questions, across tasks", fixture(), []Question{
			{Number: 2, Context: feature + "/score-strikes/client/camera.odin:score_strike", Question: "does a counter reset it"},
			{Number: 3, Context: feature + "/frame-kills/client/kill.odin:frames a kill", Question: "cut or ease"},
		}},
		{"No questions is an empty list, not nil", sketch.New(), []Question{}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if actual := Open(feature, tt.value); !reflect.DeepEqual(actual, tt.expected) {
				t.Errorf("Open() = %+v, want %+v", actual, tt.expected)
			}
		})
	}
}

func TestAnswer(t *testing.T) {
	tests := []struct {
		name     string
		number   int
		answer   string
		change   func(*sketch.Sketch)
		expected error
	}{
		{"An open question", 3, " ease ", func(value *sketch.Sketch) {
			value.Tasks[1].Files[0].Items[0].Lines[0].Answer = "ease"
		}, nil},
		{"An answered question takes a new answer", 1, "the target", func(value *sketch.Sketch) {
			value.Tasks[0].Files[0].Items[0].Lines[0].Answer = "the target"
		}, nil},
		{"Number 0", 0, "x", nil, ErrNoQuestion},
		{"A number past the last", 4, "x", nil, ErrNoQuestion},
		{"A blank answer", 2, "  ", nil, ErrMissingAnswer},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			original := fixture()

			edited, err := Answer(original, tt.number, tt.answer)

			if !errors.Is(err, tt.expected) {
				t.Fatalf("Answer() = %v, want %v", err, tt.expected)
			}
			expected := fixture()
			if tt.change != nil {
				tt.change(&expected)
			}
			if !reflect.DeepEqual(edited, expected) {
				t.Errorf("Answer() = %+v, want %+v", edited, expected)
			}
			if !reflect.DeepEqual(original, fixture()) {
				t.Errorf("Answer() changed the sketch it was given")
			}
		})
	}
}
