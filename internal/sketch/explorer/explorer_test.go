package explorer

import (
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
	"github.com/charmbracelet/x/ansi"

	"github.com/theroymind/go-tix/contracts/sketch"
	"github.com/theroymind/go-tix/internal/sketch/tree"
)

// files are the repo the tests explore; the notes below plan src/app.ts, which exists, and src/new/orbit.ts, which
// doesn't.
var files = map[string]string{
	"README.md":                  "# trial\nline two\n",
	"docs/guide.md":              "guide\n",
	"src/app.ts":                 "export const app = 1\n",
	"src/app.ts.sketch.md":       "app note\n? why\n",
	"src/new/orbit.ts.sketch.md": "orbit note\n",
	"src/util.ts":                "util\n",
	"logo.png":                   "\x89PNG\x00\x00",
}

func newModel(t *testing.T) Model {
	t.Helper()
	root := filepath.Join(t.TempDir(), "trial")
	var paths []string
	for file, text := range files {
		filePath := filepath.Join(root, filepath.FromSlash(file))
		if err := os.MkdirAll(filepath.Dir(filePath), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filePath, []byte(text), 0o644); err != nil {
			t.Fatal(err)
		}
		paths = append(paths, file)
	}
	notes := []sketch.Note{
		{SchemaVersion: 1, Path: "src/app.ts.sketch.md", Target: "src/app.ts", Mark: sketch.MarkChanged,
			Questions: []sketch.Question{{Text: "why", Line: 2}}},
		{SchemaVersion: 1, Path: "src/new/orbit.ts.sketch.md", Target: "src/new/orbit.ts", Mark: sketch.MarkNew},
	}
	model, _ := New(root, tree.Build(paths, notes)).Update(tea.WindowSizeMsg{Width: 100, Height: 20})
	return model.(Model)
}

func press(model Model, keys ...string) (Model, tea.Cmd) {
	var command tea.Cmd
	for _, name := range keys {
		var message tea.KeyPressMsg
		switch name {
		case "down":
			message = tea.KeyPressMsg{Code: tea.KeyDown}
		case "up":
			message = tea.KeyPressMsg{Code: tea.KeyUp}
		case "left":
			message = tea.KeyPressMsg{Code: tea.KeyLeft}
		case "right":
			message = tea.KeyPressMsg{Code: tea.KeyRight}
		case "enter":
			message = tea.KeyPressMsg{Code: tea.KeyEnter}
		case "ctrl+c":
			message = tea.KeyPressMsg{Code: 'c', Mod: tea.ModCtrl}
		default:
			message = tea.KeyPressMsg{Code: rune(name[0]), Text: name}
		}
		var updated tea.Model
		updated, command = model.Update(message)
		model = updated.(Model)
	}
	return model, command
}

func visiblePaths(model Model) []string {
	var paths []string
	for _, row := range model.rows {
		paths = append(paths, row.Node.Path)
	}
	return paths
}

func TestKeys(t *testing.T) {
	tests := []struct {
		name     string
		keys     []string
		selected string
		visible  []string
	}{
		{"Opens with every folder above a note open", nil, "docs",
			[]string{"docs", "src", "src/new", "src/new/orbit.ts", "src/app.ts", "src/util.ts", "logo.png", "README.md"}},
		{"j and down move down", []string{"j", "down"}, "src/new", nil},
		{"k and up move up", []string{"j", "j", "j", "k", "up"}, "src", nil},
		{"Moving stops at the ends", []string{"k", "j", "j", "j", "j", "j", "j", "j", "j", "j", "j"}, "README.md", nil},
		{"l opens a closed folder", []string{"l"}, "docs",
			[]string{"docs", "docs/guide.md", "src", "src/new", "src/new/orbit.ts", "src/app.ts", "src/util.ts",
				"logo.png", "README.md"}},
		{"enter and right step into an open folder", []string{"enter", "right"}, "docs/guide.md", nil},
		{"h closes an open folder", []string{"j", "h"}, "src",
			[]string{"docs", "src", "logo.png", "README.md"}},
		{"left on a file goes to its folder", []string{"j", "j", "j", "left"}, "src/new", nil},
		{"h on a closed folder at the top stays", []string{"h"}, "docs", nil},
		{"l on a file does nothing", []string{"j", "j", "j", "l"}, "src/new/orbit.ts", nil},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			model, _ := press(newModel(t), tt.keys...)

			if selected := model.selected().Path; selected != tt.selected {
				t.Errorf("selected = %q, want %q", selected, tt.selected)
			}
			if tt.visible != nil && !slices.Equal(visiblePaths(model), tt.visible) {
				t.Errorf("visible = %q, want %q", visiblePaths(model), tt.visible)
			}
		})
	}
}

func TestNotesOnly(t *testing.T) {
	tests := []struct {
		name     string
		keys     []string
		selected string
		visible  []string
	}{
		{"n keeps notes and the folders above them, and the cursor on a note", []string{"j", "j", "j", "n"},
			"src/new/orbit.ts", []string{"src", "src/new", "src/new/orbit.ts", "src/app.ts"}},
		{"n moves the cursor off a hidden file to its folder", []string{"j", "j", "j", "j", "j", "n"}, "src", nil},
		{"n again restores the tree and the cursor", []string{"j", "j", "j", "j", "j", "n", "n"}, "src/util.ts",
			[]string{"docs", "src", "src/new", "src/new/orbit.ts", "src/app.ts", "src/util.ts", "logo.png", "README.md"}},
		{"A hidden top-level file falls back to the first row", []string{"j", "j", "j", "j", "j", "j", "n"}, "src", nil},
		{"Moving in notes only keeps the new place on return", []string{"j", "j", "j", "j", "j", "n", "j", "j", "n"},
			"src/new/orbit.ts", nil},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			model, _ := press(newModel(t), tt.keys...)

			if selected := model.selected().Path; selected != tt.selected {
				t.Errorf("selected = %q, want %q", selected, tt.selected)
			}
			if tt.visible != nil && !slices.Equal(visiblePaths(model), tt.visible) {
				t.Errorf("visible = %q, want %q", visiblePaths(model), tt.visible)
			}
		})
	}
}

func TestQuit(t *testing.T) {
	for _, name := range []string{"q", "ctrl+c"} {
		t.Run(name, func(t *testing.T) {
			_, command := press(newModel(t), name)

			if command == nil {
				t.Fatalf("%s gave no command, want tea.Quit", name)
			}
			if _, isQuit := command().(tea.QuitMsg); !isQuit {
				t.Errorf("%s gave %T, want tea.QuitMsg", name, command())
			}
		})
	}
}

func TestView(t *testing.T) {
	tests := []struct {
		name     string
		keys     []string
		contains []string
		excludes []string
	}{
		{"The header, footer and tree", nil,
			[]string{"trial  2 notes · 1 open questions", "q quit", "n notes only", "▸   docs/", "▾   src/", "+ orbit.ts",
				"~ app.ts ?1", "util.ts", "no notes below"},
			[]string{"sketch.md"}},
		{"A note's text", []string{"j", "j", "j"}, []string{"orbit note"}, nil},
		{"A folder's summary", []string{"j"}, []string{"2 notes below, 1 open questions"}, nil},
		{"A file's first lines", []string{"j", "j", "j", "j", "j", "j", "j"}, []string{"# trial", "line two"}, nil},
		{"A binary file", []string{"j", "j", "j", "j", "j", "j"}, []string{"binary file"}, []string{"PNG"}},
		{"Notes only says so", []string{"n"}, []string{"[notes only]"}, []string{"README.md"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			model, _ := press(newModel(t), tt.keys...)

			view := ansi.Strip(model.View().Content)

			for _, text := range tt.contains {
				if !strings.Contains(view, text) {
					t.Errorf("View() lacks %q:\n%s", text, view)
				}
			}
			for _, text := range tt.excludes {
				if strings.Contains(view, text) {
					t.Errorf("View() has %q:\n%s", text, view)
				}
			}
		})
	}
}
