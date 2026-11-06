package merge

import (
	"encoding/json"
	"errors"
	"testing"

	"github.com/theroymind/go-tix/contracts/sketch"
	"github.com/theroymind/go-tix/internal/sketch/grammar"
)

const (
	atFile = "fight-camera/score-strikes/client/camera.odin"
	atItem = atFile + ":score_strike"
)

// edited applies grammar lines from a context, the way the builder makes the change Rebase replays.
func edited(t *testing.T, value sketch.Sketch, context string, lines ...string) sketch.Sketch {
	t.Helper()
	at, err := grammar.ParseContext(context)
	if err != nil {
		t.Fatal(err)
	}
	for _, line := range lines {
		if value, at, err = grammar.Apply(value, at, line); err != nil {
			t.Fatalf("Apply(%q) = %v", line, err)
		}
	}
	return value
}

func start(t *testing.T) sketch.Sketch {
	return edited(t, sketch.New(), atFile, "fn score_strike", "in actor Combatant_Id", "ask who anchors a kill",
		"t Camera_Move", "t Camera_Cut")
}

func asJSON(t *testing.T, value sketch.Sketch) string {
	t.Helper()
	text, err := json.Marshal(value.Trimmed())
	if err != nil {
		t.Fatal(err)
	}
	return string(text)
}

func TestRebase(t *testing.T) {
	tests := []struct {
		name string
		// base and changed are the change to replay; current is what's stored now; expected is the result.
		base, changed, current, expected func(t *testing.T) sketch.Sketch
	}{
		{"Undoing an added line keeps another writer's answer on another item",
			func(t *testing.T) sketch.Sketch { return edited(t, start(t), atFile+":Camera_Move", "out x") },
			start,
			func(t *testing.T) sketch.Sketch {
				return edited(t, edited(t, start(t), atFile+":Camera_Move", "out x"), atItem, "a later question")
			},
			func(t *testing.T) sketch.Sketch { return edited(t, start(t), atItem, "a later question") }},
		{"Undoing a removed item puts it back between its siblings",
			func(t *testing.T) sketch.Sketch { return edited(t, start(t), atFile+":Camera_Move", "rm") },
			start,
			func(t *testing.T) sketch.Sketch {
				return edited(t, edited(t, start(t), atFile+":Camera_Move", "rm"), atFile, "fn frame_kill")
			},
			func(t *testing.T) sketch.Sketch { return edited(t, start(t), atFile, "fn frame_kill") }},
		{"Undoing a removed file restores it and its items",
			func(t *testing.T) sketch.Sketch { return edited(t, start(t), atFile, "rm") },
			start,
			func(t *testing.T) sketch.Sketch { return edited(t, start(t), atFile, "rm") },
			start},
		{"Undoing a rename restores the old name",
			func(t *testing.T) sketch.Sketch { return edited(t, start(t), atItem, "mv score_hit") },
			start,
			func(t *testing.T) sketch.Sketch { return edited(t, start(t), atItem, "mv score_hit") },
			start},
		{"An edit lands beside another writer's new task",
			start,
			func(t *testing.T) sketch.Sketch { return edited(t, start(t), atItem, "out Camera_Move") },
			func(t *testing.T) sketch.Sketch { return edited(t, start(t), "fight-camera", "task frame-kills") },
			func(t *testing.T) sketch.Sketch {
				return edited(t, start(t), atItem, "out Camera_Move", "task frame-kills")
			}},
		{"No change leaves current as it is",
			start, start,
			func(t *testing.T) sketch.Sketch { return edited(t, start(t), atItem, "a later question") },
			func(t *testing.T) sketch.Sketch { return edited(t, start(t), atItem, "a later question") }},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			actual, err := Rebase(tt.base(t), tt.changed(t), tt.current(t))

			if err != nil {
				t.Fatalf("Rebase() = %v", err)
			}
			if asJSON(t, actual) != asJSON(t, tt.expected(t)) {
				t.Errorf("Rebase() =\n%s\nwant\n%s", asJSON(t, actual), asJSON(t, tt.expected(t)))
			}
		})
	}
}

func TestRebaseConflicts(t *testing.T) {
	tests := []struct {
		name                   string
		base, changed, current func(t *testing.T) sketch.Sketch
		expected               string
	}{
		{"Undo after another writer changed the same item",
			func(t *testing.T) sketch.Sketch { return edited(t, start(t), atItem, "out x") },
			start,
			func(t *testing.T) sketch.Sketch { return edited(t, start(t), atItem, "out x", "a later question") },
			"score-strikes/client/camera.odin:score_strike changed since"},
		{"An edit to an item another writer removed",
			start,
			func(t *testing.T) sketch.Sketch { return edited(t, start(t), atItem, "out x") },
			func(t *testing.T) sketch.Sketch { return edited(t, start(t), atItem, "rm") },
			"score-strikes/client/camera.odin:score_strike changed since"},
		{"A rename onto a name another writer took",
			start,
			func(t *testing.T) sketch.Sketch { return edited(t, start(t), atItem, "mv score_hit") },
			func(t *testing.T) sketch.Sketch { return edited(t, start(t), atFile, "fn score_hit") },
			"score-strikes/client/camera.odin:score_hit changed since"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			current := tt.current(t)

			actual, err := Rebase(tt.base(t), tt.changed(t), current)

			if !errors.Is(err, ErrConflict) {
				t.Fatalf("Rebase() = %v, want %v", err, ErrConflict)
			}
			if err.Error() != tt.expected {
				t.Errorf("Rebase() = %q, want %q", err, tt.expected)
			}
			if asJSON(t, actual) != asJSON(t, current) {
				t.Errorf("a conflicting Rebase() changed current")
			}
		})
	}
}
