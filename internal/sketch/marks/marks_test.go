package marks

import (
	"os"
	"path/filepath"
	"slices"
	"testing"

	"github.com/theroymind/go-tix/contracts/sketch"
)

const camera = "package client\n\nCamera_Move :: struct { cut: bool }\nscore_strikes :: proc() {}\n// frames a kill\ncafé := 1\n"

func TestMark(t *testing.T) {
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, "client"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "client", "camera.odin"), []byte(camera), 0o644); err != nil {
		t.Fatal(err)
	}
	tests := []struct {
		name     string
		path     string
		item     string
		expected sketch.Mark
	}{
		{"A name the file declares", "client/camera.odin", "Camera_Move", sketch.MarkChanged},
		{"A name the file lacks", "client/camera.odin", "frame_kill", sketch.MarkNew},
		{"A prefix of a longer name", "client/camera.odin", "score_strike", sketch.MarkNew},
		{"A suffix of a longer name", "client/camera.odin", "Move", sketch.MarkNew},
		{"A name at the end of the file", "client/camera.odin", "1", sketch.MarkChanged},
		{"A name with spaces, as a test's", "client/camera.odin", "frames a kill", sketch.MarkChanged},
		{"A name next to a non-ASCII letter", "client/camera.odin", "caf", sketch.MarkNew},
		{"A later whole-word match after a partial one", "client/camera.odin", "score_strikes", sketch.MarkChanged},
		{"A file that doesn't exist yet", "client/hud.odin", "Camera_Move", sketch.MarkNew},
		{"An empty name", "client/camera.odin", "", sketch.MarkNew},
	}
	reader := NewReader(root)
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			actual, err := reader.Mark(tt.path, tt.item)

			if err != nil || actual != tt.expected {
				t.Errorf("Mark() = %q, %v, want %q", actual, err, tt.expected)
			}
		})
	}
}

func TestContainsWord(t *testing.T) {
	tests := []struct {
		name     string
		text     string
		word     string
		expected bool
	}{
		{"The whole text", "score", "score", true},
		{"Bounded by punctuation", "x(score)", "score", true},
		{"A partial match, then a whole one", "scores score", "score", true},
		{"Only partial matches", "scores _score score1", "score", false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if actual := containsWord(tt.text, tt.word); actual != tt.expected {
				t.Errorf("containsWord(%q, %q) = %v, want %v", tt.text, tt.word, actual, tt.expected)
			}
		})
	}
}

func TestMarkReadsAFileOnce(t *testing.T) {
	root := t.TempDir()
	path := filepath.Join(root, "camera.odin")
	if err := os.WriteFile(path, []byte("Camera_Move"), 0o644); err != nil {
		t.Fatal(err)
	}
	reader := NewReader(root)
	if mark, _ := reader.Mark("camera.odin", "Camera_Move"); mark != sketch.MarkChanged {
		t.Fatalf("Mark() = %q, want %q", mark, sketch.MarkChanged)
	}
	if err := os.Remove(path); err != nil {
		t.Fatal(err)
	}

	mark, err := reader.Mark("camera.odin", "Camera_Move")

	if err != nil || mark != sketch.MarkChanged {
		t.Errorf("Mark() after the file went = %q, %v, want the cached %q", mark, err, sketch.MarkChanged)
	}
}

func TestMarkNote(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "camera.odin"), []byte(camera), 0o644); err != nil {
		t.Fatal(err)
	}
	items := []sketch.Item{
		{Type: sketch.ItemTypeType, Name: "Camera_Move"},
		{Type: sketch.ItemTypeFn, Name: "score_strike"},
		{Type: sketch.ItemTypeFn, Name: "frame_kill"},
	}
	tests := []struct {
		name          string
		target        string
		expectedNote  sketch.Mark
		expectedItems []sketch.Mark
	}{
		{"A target that exists", "camera.odin", sketch.MarkChanged,
			[]sketch.Mark{sketch.MarkChanged, sketch.MarkNew, sketch.MarkNew}},
		{"A target that doesn't exist yet", "hud.odin", sketch.MarkNew,
			[]sketch.Mark{sketch.MarkNew, sketch.MarkNew, sketch.MarkNew}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			value := sketch.Note{Target: tt.target, Items: slices.Clone(items)}

			err := NewReader(root).MarkNote(&value)

			if err != nil {
				t.Fatalf("MarkNote() error = %v", err)
			}
			if value.Mark != tt.expectedNote {
				t.Errorf("note mark = %q, want %q", value.Mark, tt.expectedNote)
			}
			var actualItems []sketch.Mark
			for _, item := range value.Items {
				actualItems = append(actualItems, item.Mark)
			}
			if !slices.Equal(actualItems, tt.expectedItems) {
				t.Errorf("item marks = %q, want %q", actualItems, tt.expectedItems)
			}
		})
	}
}
