package marks

import (
	"os"
	"path/filepath"
	"testing"
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
		expected Mark
	}{
		{"A name the file declares", "client/camera.odin", "Camera_Move", Changed},
		{"A name the file lacks", "client/camera.odin", "frame_kill", New},
		{"A prefix of a longer name", "client/camera.odin", "score_strike", New},
		{"A suffix of a longer name", "client/camera.odin", "Move", New},
		{"A name at the end of the file", "client/camera.odin", "1", Changed},
		{"A name with spaces, as a test's", "client/camera.odin", "frames a kill", Changed},
		{"A name next to a non-ASCII letter", "client/camera.odin", "caf", New},
		{"A later whole-word match after a partial one", "client/camera.odin", "score_strikes", Changed},
		{"A file that doesn't exist yet", "client/hud.odin", "Camera_Move", New},
		{"An empty name", "client/camera.odin", "", New},
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
	if mark, _ := reader.Mark("camera.odin", "Camera_Move"); mark != Changed {
		t.Fatalf("Mark() = %q, want %q", mark, Changed)
	}
	if err := os.Remove(path); err != nil {
		t.Fatal(err)
	}

	mark, err := reader.Mark("camera.odin", "Camera_Move")

	if err != nil || mark != Changed {
		t.Errorf("Mark() after the file went = %q, %v, want the cached %q", mark, err, Changed)
	}
}
