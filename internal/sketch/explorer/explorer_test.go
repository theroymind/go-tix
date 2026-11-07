package explorer

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
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
	for file, text := range files {
		writeFile(t, root, file, text)
	}
	model, err := New(root)
	if err != nil {
		t.Fatal(err)
	}
	model.open = func(string, int, func(error) tea.Msg) tea.Cmd {
		t.Error("a note opened outside the test's editor")
		return nil
	}
	// Run inside nvim, New pairs with it; paired_test.go covers that.
	model.paired = false
	// The tests start from the whole tree; TestOpensNotesOnly covers New opening on notes only.
	model.notesOnly = false
	model.refreshRows()
	updated, _ := model.Update(tea.WindowSizeMsg{Width: 100, Height: 20})
	return updated.(Model)
}

func writeFile(t *testing.T, root, file, text string) {
	t.Helper()
	filePath := filepath.Join(root, filepath.FromSlash(file))
	if err := os.MkdirAll(filepath.Dir(filePath), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filePath, []byte(text), 0o644); err != nil {
		t.Fatal(err)
	}
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
		case "esc":
			message = tea.KeyPressMsg{Code: tea.KeyEscape}
		case "backspace":
			message = tea.KeyPressMsg{Code: tea.KeyBackspace}
		case "tab":
			message = tea.KeyPressMsg{Code: tea.KeyTab}
		case "ctrl+c":
			message = tea.KeyPressMsg{Code: 'c', Mod: tea.ModCtrl}
		case "ctrl+s":
			message = tea.KeyPressMsg{Code: 's', Mod: tea.ModCtrl}
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
		{"l and right step into an open folder", []string{"l", "right"}, "docs/guide.md", nil},
		{"enter opens a closed folder", []string{"enter"}, "docs",
			[]string{"docs", "docs/guide.md", "src", "src/new", "src/new/orbit.ts", "src/app.ts", "src/util.ts",
				"logo.png", "README.md"}},
		{"enter closes an open folder", []string{"enter", "enter"}, "docs",
			[]string{"docs", "src", "src/new", "src/new/orbit.ts", "src/app.ts", "src/util.ts", "logo.png", "README.md"}},
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
			[]string{"trial  2 notes · 2 draft · 0 ready · 1 open question · 0 dangling", "n notes · a add · e edit",
				"y ready", "> answer", "] question", "? help", "▸   docs/", "▾   src/", "+ orbit.ts",
				"~ app.ts ?1", "util.ts", "no notes below"},
			[]string{"sketch.md"}},
		{"A note's text", []string{"j", "j", "j"}, []string{"orbit note"}, nil},
		{"A folder's summary", []string{"j"}, []string{"2 notes below, 1 open question"}, nil},
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

// recordOpens makes model's editor record the files it opens and return at once.
func recordOpens(model *Model) *[]string {
	opened := []string{}
	model.open = func(filePath string, _ int, done func(error) tea.Msg) tea.Cmd {
		opened = append(opened, filePath)
		return func() tea.Msg { return done(nil) }
	}
	return &opened
}

func readNote(t *testing.T, model Model, notePath string) string {
	t.Helper()
	text, err := os.ReadFile(model.filePath(notePath))
	if err != nil {
		t.Fatal(err)
	}
	return string(text)
}

func TestAdd(t *testing.T) {
	tests := []struct {
		name     string
		feature  string
		keys     []string
		note     string
		text     string
		selected string
		mark     string
	}{
		{"a on a folder plans a file in it", "", []string{"j", "a", "useOrbit.ts", "enter"},
			"src/useOrbit.ts.sketch.md", "---\nstatus: draft\n---\n", "src/useOrbit.ts", "+"},
		{"A name with folders makes them and opens them in the tree", "",
			[]string{"j", "a", "composables/useOrbit.ts", "enter"},
			"src/composables/useOrbit.ts.sketch.md", "---\nstatus: draft\n---\n", "src/composables/useOrbit.ts", "+"},
		{"a on a file plans one in its folder", "", []string{"j", "j", "j", "j", "j", "a", "x.ts", "enter"},
			"src/x.ts.sketch.md", "---\nstatus: draft\n---\n", "src/x.ts", "+"},
		{"a on a top-level file plans one at the root", "", []string{"j", "j", "j", "j", "j", "j", "j", "a", "x.ts", "enter"},
			"x.ts.sketch.md", "---\nstatus: draft\n---\n", "x.ts", "+"},
		{"backspace takes back a letter", "", []string{"j", "a", "useOrbix", "backspace", "t.ts", "enter"},
			"src/useOrbit.ts.sketch.md", "---\nstatus: draft\n---\n", "src/useOrbit.ts", "+"},
		{"A feature filter goes in front matter", "orbit", []string{"j", "a", "useOrbit.ts", "enter"},
			"src/useOrbit.ts.sketch.md", "---\nfeature: orbit\nstatus: draft\n---\n", "src/useOrbit.ts", "+"},
		{"e on a file without a note adds one", "", []string{"j", "j", "j", "j", "j", "e"},
			"src/util.ts.sketch.md", "---\nstatus: draft\n---\n", "src/util.ts", "~"},
		{"enter on a file without a note adds one", "", []string{"j", "j", "j", "j", "j", "enter"},
			"src/util.ts.sketch.md", "---\nstatus: draft\n---\n", "src/util.ts", "~"},
		{"A typed .sketch.md comes off", "", []string{"j", "a", "useOrbit.ts.sketch.md", "enter"},
			"src/useOrbit.ts.sketch.md", "---\nstatus: draft\n---\n", "src/useOrbit.ts", "+"},
		{"A typed .sketch before the extension comes out", "", []string{"j", "a", "useOrbit.sketch.ts", "enter"},
			"src/useOrbit.ts.sketch.md", "---\nstatus: draft\n---\n", "src/useOrbit.ts", "+"},
		{"A name with no extension", "", []string{"j", "a", "useOrbit", "enter"},
			"src/useOrbit.sketch.md", "---\nstatus: draft\n---\n", "src/useOrbit", "+"},
		{"a on a node with a note opens it", "", []string{"j", "j", "j", "a"},
			"src/new/orbit.ts.sketch.md", "orbit note\n", "src/new/orbit.ts", "+"},
		{"e on a node with a note opens it", "", []string{"j", "j", "j", "j", "e"},
			"src/app.ts.sketch.md", "app note\n? why\n", "src/app.ts", "~"},
		{"enter on a node with a note opens it", "", []string{"j", "j", "j", "j", "enter"},
			"src/app.ts.sketch.md", "app note\n? why\n", "src/app.ts", "~"},
		{"Naming a file that has a note opens it", "", []string{"j", "a", "app.ts", "enter"},
			"src/app.ts.sketch.md", "app note\n? why\n", "src/app.ts", "~"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			model := newModel(t)
			model.feature = tt.feature
			opened := recordOpens(&model)

			model, _ = press(model, tt.keys...)

			if text := readNote(t, model, tt.note); text != tt.text {
				t.Errorf("note = %q, want %q", text, tt.text)
			}
			if want := []string{model.filePath(tt.note)}; !slices.Equal(*opened, want) {
				t.Errorf("opened %q, want %q", *opened, want)
			}
			if model.prompt.kind != promptNone {
				t.Errorf("prompt still shows: %q", model.prompt.line())
			}
			selected := model.selected()
			if selected.Path != tt.selected {
				t.Fatalf("selected = %q, want %q", selected.Path, tt.selected)
			}
			if selected.Note == nil || selected.Note.Mark.Symbol() != tt.mark {
				t.Errorf("selected note = %+v, want mark %s", selected.Note, tt.mark)
			}
		})
	}
}

func TestAddRejects(t *testing.T) {
	tests := []struct {
		name  string
		typed string
		err   error
	}{
		{"An empty name", "  ", ErrEmptyName},
		{"An absolute path", "/etc/x.ts", ErrAbsoluteName},
		{"A .. folder", "../x.ts", ErrNameSegment},
		{"A . folder", "./x.ts", ErrNameSegment},
		{"An empty folder", "a//x.ts", ErrNameSegment},
		{"A trailing slash", "hooks/", ErrNameSegment},
		{"Only .sketch.md", ".sketch.md", ErrEmptyName},
		{"Only .sketch", ".sketch", ErrEmptyName},
		{"A .. folder after normalising", "../x.sketch.ts", ErrNameSegment},
		{"A ghost folder", "new", ErrFolderName},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			model := newModel(t)
			opened := recordOpens(&model)
			model, _ = press(model, "j", "a")
			if tt.typed != "" {
				model, _ = press(model, tt.typed)
			}
			before := visiblePaths(model)

			model, _ = press(model, "enter")

			if model.status != tt.err.Error() {
				t.Errorf("status = %q, want %q", model.status, tt.err)
			}
			if model.prompt.kind != promptName || model.prompt.typed != tt.typed {
				t.Errorf("prompt = %+v, want the name prompt still holding %q", model.prompt, tt.typed)
			}
			if len(*opened) > 0 || !slices.Equal(visiblePaths(model), before) {
				t.Errorf("opened %q and rows %q, want nothing made", *opened, visiblePaths(model))
			}
		})
	}
}

func TestAddRejectsFolderOnDisk(t *testing.T) {
	model := newModel(t)
	if err := os.MkdirAll(model.filePath("src/empty"), 0o755); err != nil {
		t.Fatal(err)
	}

	model, _ = press(model, "j", "a", "empty", "enter")

	if model.status != ErrFolderName.Error() || model.prompt.kind != promptName {
		t.Errorf("status = %q, prompt = %+v, want %q with the prompt open", model.status, model.prompt, ErrFolderName)
	}
}

func TestEscCancelsName(t *testing.T) {
	model := newModel(t)
	opened := recordOpens(&model)

	model, command := press(model, "j", "a", "useOrbit.ts", "esc")

	if command != nil || len(*opened) > 0 || model.prompt.kind != promptNone {
		t.Errorf("command %v, opened %q, prompt %+v, want nothing", command, *opened, model.prompt)
	}
	if _, err := os.Stat(model.filePath("src/useOrbit.ts.sketch.md")); !os.IsNotExist(err) {
		t.Errorf("the note was made: %v", err)
	}
	if view := ansi.Strip(model.View().Content); !strings.Contains(view, "y ready") {
		t.Errorf("the footer lacks the keys again:\n%s", view)
	}
}

func TestPromptTakesKeys(t *testing.T) {
	model := newModel(t)

	model, command := press(model, "j", "a", "q", "d", "j")

	if command != nil || model.prompt.typed != "qdj" || model.selected().Path != "src" {
		t.Errorf("command %v, typed %q, selected %q, want q, d and j typed", command, model.prompt.typed,
			model.selected().Path)
	}
	if view := ansi.Strip(model.View().Content); !strings.Contains(view, "plan file: src/qdj") {
		t.Errorf("the footer lacks the prompt:\n%s", view)
	}
}

func TestEdit(t *testing.T) {
	model := newModel(t)
	opened := recordOpens(&model)
	model, command := press(model, "j", "j", "j", "e")
	if want := []string{model.filePath("src/new/orbit.ts.sketch.md")}; !slices.Equal(*opened, want) {
		t.Fatalf("opened %q, want %q", *opened, want)
	}
	writeFile(t, model.root, "src/new/orbit.ts.sketch.md", "orbit edited\n? how\n")
	writeFile(t, model.root, "src/a/b.ts", "b\n")

	updated, _ := model.Update(command())
	model = updated.(Model)

	if model.selected().Path != "src/new/orbit.ts" {
		t.Errorf("selected = %q, want the edited note's node", model.selected().Path)
	}
	view := ansi.Strip(model.View().Content)
	for _, text := range []string{"orbit edited", "+ orbit.ts ?1", "a/", "2 open questions"} {
		if !strings.Contains(view, text) {
			t.Errorf("View() lacks %q after the editor:\n%s", text, view)
		}
	}
}

func TestEditOnFolderWithoutNote(t *testing.T) {
	model := newModel(t)
	opened := recordOpens(&model)

	model, command := press(model, "e")

	if command != nil || len(*opened) > 0 || model.status != noNoteToEdit {
		t.Errorf("command %v, opened %q, status %q, want only %q", command, *opened, model.status, noNoteToEdit)
	}
}

func TestEditorFails(t *testing.T) {
	model := newModel(t)
	model.open = func(_ string, _ int, done func(error) tea.Msg) tea.Cmd {
		return func() tea.Msg { return done(os.ErrPermission) }
	}
	model, command := press(model, "j", "j", "j", "e")

	updated, _ := model.Update(command())

	if status := updated.(Model).status; status != "editor: "+os.ErrPermission.Error() {
		t.Errorf("status = %q, want the editor's error", status)
	}
}

func TestDelete(t *testing.T) {
	tests := []struct {
		name     string
		keys     []string
		note     string
		kept     bool
		selected string
		visible  []string
	}{
		{"y deletes a ghost note, and its node and ghost folder go", []string{"j", "j", "j", "d", "y"},
			"src/new/orbit.ts.sketch.md", false, "src",
			[]string{"docs", "src", "src/app.ts", "src/util.ts", "logo.png", "README.md"}},
		{"y deletes an attached note, and its file stays plain", []string{"j", "j", "j", "j", "d", "y"},
			"src/app.ts.sketch.md", false, "src/app.ts", nil},
		{"Any other key keeps the note", []string{"j", "j", "j", "d", "n"},
			"src/new/orbit.ts.sketch.md", true, "src/new/orbit.ts", nil},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			model, _ := press(newModel(t), tt.keys...)

			_, err := os.Stat(model.filePath(tt.note))
			if kept := err == nil; kept != tt.kept {
				t.Errorf("note kept = %v, want %v", kept, tt.kept)
			}
			if _, err := os.Stat(model.filePath("src/app.ts")); err != nil {
				t.Errorf("the target went too: %v", err)
			}
			selected := model.selected()
			if selected.Path != tt.selected {
				t.Errorf("selected = %q, want %q", selected.Path, tt.selected)
			}
			if !tt.kept && selected.Note != nil {
				t.Errorf("the selected node still has a note: %+v", selected.Note)
			}
			if tt.visible != nil && !slices.Equal(visiblePaths(model), tt.visible) {
				t.Errorf("visible = %q, want %q", visiblePaths(model), tt.visible)
			}
		})
	}
}

func TestDeleteAsks(t *testing.T) {
	model, _ := press(newModel(t), "j", "j", "j", "j", "d")

	if view := ansi.Strip(model.View().Content); !strings.Contains(view, "delete src/app.ts.sketch.md? y/n") {
		t.Errorf("View() lacks the question:\n%s", view)
	}
}

func TestDeleteWithoutNote(t *testing.T) {
	model, _ := press(newModel(t), "j", "j", "j", "j", "j", "d")

	if model.status != noNoteToDelete || model.prompt.kind != promptNone {
		t.Errorf("status %q, prompt %+v, want only %q", model.status, model.prompt, noNoteToDelete)
	}
}

// selectPath puts model's cursor on the row showing nodePath.
func selectPath(t *testing.T, model Model, nodePath string) Model {
	t.Helper()
	row := model.rowOf(nodePath)
	if row < 0 {
		t.Fatalf("no row shows %q in %q", nodePath, visiblePaths(model))
	}
	model.moveTo(row)
	return model
}

func paste(model Model, text string) Model {
	updated, _ := model.Update(tea.PasteMsg{Content: text})
	return updated.(Model)
}

func TestPicker(t *testing.T) {
	choices := []string{"makeCamera", "make_camera", "useCamera", "Zoom"}
	tests := []struct {
		name     string
		keys     []string
		typed    string
		matches  []string
		cursor   int
		picked   string
		isPicked bool
	}{
		{"Opens on every choice", nil, "", choices, 0, "", false},
		{"Typing filters, ignoring case", []string{"c", "A", "m"}, "cAm",
			[]string{"makeCamera", "make_camera", "useCamera"}, 0, "", false},
		{"j and k type", []string{"k"}, "k", []string{"makeCamera", "make_camera"}, 0, "", false},
		{"down and ctrl+n move, stopping at the last", []string{"down", "ctrl+n", "down", "down", "down"}, "", choices,
			3, "", false},
		{"up and ctrl+p move, stopping at the first", []string{"down", "down", "up", "ctrl+p", "up"}, "", choices, 0,
			"", false},
		{"Typing puts the cursor back on the first match", []string{"down", "down", "o"}, "o",
			[]string{"Zoom"}, 0, "", false},
		{"backspace takes back a letter", []string{"Z", "x", "backspace"}, "Z", []string{"Zoom"}, 0, "", false},
		{"enter picks the choice under the cursor", []string{"_", "c", "enter"}, "_c", []string{"make_camera"}, 0,
			"make_camera", true},
		{"enter with no match picks nothing", []string{"q", "enter"}, "q", nil, 0, "", false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			list := picker{kind: pickFile, choices: choices}
			var picked string
			var isPicked bool
			for _, name := range tt.keys {
				var message tea.KeyPressMsg
				switch name {
				case "down":
					message = tea.KeyPressMsg{Code: tea.KeyDown}
				case "up":
					message = tea.KeyPressMsg{Code: tea.KeyUp}
				case "ctrl+n":
					message = tea.KeyPressMsg{Code: 'n', Mod: tea.ModCtrl}
				case "ctrl+p":
					message = tea.KeyPressMsg{Code: 'p', Mod: tea.ModCtrl}
				case "enter":
					message = tea.KeyPressMsg{Code: tea.KeyEnter}
				case "backspace":
					message = tea.KeyPressMsg{Code: tea.KeyBackspace}
				default:
					message = tea.KeyPressMsg{Code: rune(name[0]), Text: name}
				}
				picked, isPicked = list.press(message)
			}

			if list.typed != tt.typed || !slices.Equal(list.matches(), tt.matches) || list.cursor != tt.cursor {
				t.Errorf("typed %q, matches %q, cursor %d, want %q, %q, %d", list.typed, list.matches(), list.cursor,
					tt.typed, tt.matches, tt.cursor)
			}
			if picked != tt.picked || isPicked != tt.isPicked {
				t.Errorf("picked %q %v, want %q %v", picked, isPicked, tt.picked, tt.isPicked)
			}
		})
	}
}

func TestPickerLines(t *testing.T) {
	list := picker{title: "names in a.ts", choices: []string{"a1", "a2", "a3", "a4", "b"}, typed: "a", cursor: 3}

	lines := list.lines(4)

	if want := []string{"names in a.ts", "filter: a█", "  a3", "> a4"}; !slices.Equal(lines, want) {
		t.Errorf("lines(4) = %q, want %q", lines, want)
	}
}

func TestRefChoices(t *testing.T) {
	model, _ := press(selectPath(t, newModel(t), "src/app.ts"), "r")

	want := []string{"docs/guide.md", "src/new/orbit.ts", "src/util.ts", "logo.png", "README.md"}
	if !slices.Equal(model.picker.choices, want) {
		t.Errorf("choices = %q, want %q", model.picker.choices, want)
	}
}

func TestRef(t *testing.T) {
	tests := []struct {
		name     string
		selected string
		note     string // the selected note's text, or "" to leave it as files has it
		keys     []string
		text     string
		status   string
	}{
		{"enter adds the pick on a line of its own", "src/app.ts", "app note\n", []string{"u", "t", "i", "l", "enter"},
			"app note\n-> src/util.ts\n", ""},
		{"A note without a last newline gets one first", "src/app.ts", "app note", []string{"u", "t", "i", "l", "enter"},
			"app note\n-> src/util.ts\n", ""},
		{"A ghost note links to a planned file", "src/new/orbit.ts", "", []string{"a", "p", "p", "enter"},
			"orbit note\n-> src/app.ts\n", ""},
		{"A link already there isn't added again", "src/app.ts", "-> src/util.ts\n",
			[]string{"u", "t", "i", "l", "enter"}, "-> src/util.ts\n", "src/app.ts already links to src/util.ts"},
		{"esc cancels", "src/app.ts", "app note\n", []string{"u", "esc"}, "app note\n", ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			model := newModel(t)
			notePath := tt.selected + ".sketch.md"
			if tt.note != "" {
				writeFile(t, model.root, notePath, tt.note)
				model.reload(tt.selected)
			}
			model, _ = press(selectPath(t, model, tt.selected), "r")
			model, _ = press(model, tt.keys...)

			if text := readNote(t, model, notePath); text != tt.text {
				t.Errorf("note = %q, want %q", text, tt.text)
			}
			if model.picker.kind != pickNone || model.selected().Path != tt.selected || model.status != tt.status {
				t.Errorf("picker %+v, selected %q, status %q, want the picker closed on the note, status %q",
					model.picker, model.selected().Path, model.status, tt.status)
			}
		})
	}
}

func TestPickAfterReload(t *testing.T) {
	tests := []struct {
		name    string
		keys    []string // opening the picker on src/app.ts's note
		pick    []string
		deleted bool   // the note is deleted while the picker is open, rather than the cursor moving
		text    string // src/app.ts's note afterwards, when it's still there
		status  string
	}{
		{"r links the note it opened for, though the cursor moved", []string{"r"}, []string{"u", "t", "i", "l", "enter"},
			false, "---\nfeature: camera\n---\napp note\n? why\n-> src/util.ts\n", ""},
		{"L links the note it opened for, though the cursor moved", []string{"L"}, []string{"o", "r", "b", "enter"},
			false, "---\nfeature: orbit\n---\napp note\n? why\n", "linked src/app.ts to orbit"},
		{"r on a note gone since says so", []string{"r"}, []string{"u", "t", "i", "l", "enter"}, true, "",
			"src/app.ts has no note any more"},
		{"L on a note gone since says so", []string{"L"}, []string{"o", "r", "b", "enter"}, true, "",
			"src/app.ts has no note any more"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			model, _ := press(selectPath(t, featureModel(t), "src/app.ts"), tt.keys...)
			if tt.deleted {
				if err := os.Remove(model.filePath("src/app.ts.sketch.md")); err != nil {
					t.Fatal(err)
				}
			} else {
				model.moveTo(model.rowOf("src/new/orbit.ts"))
			}
			updated, _ := model.Update(tea.FocusMsg{})
			model = updated.(Model)

			model, _ = press(model, tt.pick...)

			if model.status != tt.status {
				t.Errorf("status = %q, want %q", model.status, tt.status)
			}
			if !tt.deleted {
				if text := readNote(t, model, "src/app.ts.sketch.md"); text != tt.text {
					t.Errorf("note = %q, want %q", text, tt.text)
				}
			}
			if text := readNote(t, model, "src/new/orbit.ts.sketch.md"); text != "orbit note\n" {
				t.Errorf("orbit note = %q, want it untouched", text)
			}
		})
	}
}

func TestRefRejects(t *testing.T) {
	tests := []struct {
		name     string
		selected string
	}{
		{"A file without a note", "src/util.ts"},
		{"A folder", "src"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			model := selectPath(t, newModel(t), tt.selected)

			model, _ = press(model, "r")

			if model.picker.kind != pickNone || model.status != refNeedsNote {
				t.Errorf("picker %+v, status %q, want only %q", model.picker, model.status, refNeedsNote)
			}
		})
	}
}

func TestLinksLine(t *testing.T) {
	files := []string{"src/a.ts", "src/b.ts", "lib/b.ts", "src/d.ts"}
	tests := []struct {
		name     string
		notes    []sketch.Note
		selected string
		expected string
	}{
		{"None either way", []sketch.Note{{Path: "src/a.ts.sketch.md", Target: "src/a.ts"}}, "src/a.ts", ""},
		{"Uses, each marked, the dangling one !",
			[]sketch.Note{{Path: "src/a.ts.sketch.md", Target: "src/a.ts", Edges: []sketch.Edge{
				{To: "src/d.ts", Mark: sketch.MarkChanged}, {To: "src/new.ts", Mark: sketch.MarkNew}, {To: "src/c.ts"}}},
				{Path: "src/new.ts.sketch.md", Target: "src/new.ts"}},
			"src/a.ts", "uses → ~ d.ts, + new.ts, ! c.ts"},
		{"A shared base name shows in full",
			[]sketch.Note{{Path: "src/a.ts.sketch.md", Target: "src/a.ts", Edges: []sketch.Edge{
				{To: "src/b.ts", Mark: sketch.MarkChanged}}}},
			"src/a.ts", "uses → ~ src/b.ts"},
		{"A dangling path sharing a base name shows in full",
			[]sketch.Note{{Path: "src/a.ts.sketch.md", Target: "src/a.ts", Edges: []sketch.Edge{{To: "lib/d.ts"}}}},
			"src/a.ts", "uses → ! lib/d.ts"},
		{"Used by, marked as the linking note's target",
			[]sketch.Note{{Path: "src/a.ts.sketch.md", Target: "src/a.ts", Mark: sketch.MarkChanged,
				Edges: []sketch.Edge{{To: "src/d.ts", Mark: sketch.MarkChanged}}},
				{Path: "src/e.ts.sketch.md", Target: "src/e.ts", Mark: sketch.MarkNew,
					Edges: []sketch.Edge{{To: "src/d.ts", Mark: sketch.MarkChanged}}}},
			"src/d.ts", "used by ← ~ a.ts, + e.ts"},
		{"Both ways",
			[]sketch.Note{{Path: "src/a.ts.sketch.md", Target: "src/a.ts", Mark: sketch.MarkChanged,
				Edges: []sketch.Edge{{To: "src/d.ts", Mark: sketch.MarkChanged}}},
				{Path: "src/d.ts.sketch.md", Target: "src/d.ts", Mark: sketch.MarkChanged,
					Edges: []sketch.Edge{{To: "src/a.ts", Mark: sketch.MarkChanged}}}},
			"src/a.ts", "uses → ~ d.ts   used by ← ~ d.ts"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			top := tree.Build(files, tt.notes, nil)
			var selected *tree.Node
			for _, node := range nodes(top) {
				if node.Path == tt.selected {
					selected = node
				}
			}

			if actual := linksLine(indexLinks(top), selected); actual != tt.expected {
				t.Errorf("linksLine() = %q, want %q", actual, tt.expected)
			}
		})
	}
}

func TestViewShowsLinks(t *testing.T) {
	model := newModel(t)
	writeFile(t, model.root, "src/app.ts.sketch.md", "app note\n-> src/util.ts\n-> src/gone.ts\n")
	model.reload("src/app.ts")

	tests := []struct {
		name     string
		selected string
		line     string
	}{
		{"A note's uses", "src/app.ts", "uses → ~ util.ts, ! gone.ts"},
		{"A file's users", "src/util.ts", "used by ← ~ app.ts"},
		{"Neither, no line", "README.md", ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			model := selectPath(t, model, tt.selected)

			lines := strings.Split(ansi.Strip(model.View().Content), "\n")

			if linksRow := lines[len(lines)-2]; linksRow != tt.line {
				t.Errorf("links row = %q, want %q", linksRow, tt.line)
			}
		})
	}
}

// featureModel is newModel with src/app.ts's note in feature camera, a ghost note in feature orbit, and tix
// knowing the features camera and sketch-explorer.
func featureModel(t *testing.T) Model {
	t.Helper()
	model := newModel(t)
	writeFile(t, model.root, "src/app.ts.sketch.md", "---\nfeature: camera\n---\napp note\n? why\n")
	writeFile(t, model.root, "docs/plan.md.sketch.md", "---\nfeature: orbit\n---\n? when\n? where\n")
	model.listFeatures = func() ([]string, error) { return []string{"sketch-explorer", "camera"}, nil }
	model.reload("docs")
	return model
}

func TestFeature(t *testing.T) {
	tests := []struct {
		name     string
		keys     []string
		feature  string
		visible  []string
		contains []string
		excludes []string
	}{
		{"Without a filter, every note", nil, "",
			[]string{"docs", "src", "src/new", "src/new/orbit.ts", "src/app.ts",
				"src/util.ts", "logo.png", "README.md"},
			[]string{"3 notes · 3 draft · 0 ready · 3 open questions"}, []string{"feature:"}},
		{"A feature hides other features' notes and ghosts", []string{"f", "c", "a", "m", "enter"}, "camera",
			[]string{"docs", "src", "src/app.ts", "src/util.ts", "logo.png", "README.md"},
			[]string{"feature: camera", "1 note · 1 draft · 0 ready · 1 open question", "~ app.ts ?1"}, []string{"plan.md", "orbit.ts"}},
		{"A feature only notes name", []string{"f", "o", "r", "enter"}, "orbit",
			[]string{"docs", "docs/guide.md", "docs/plan.md", "src", "src/app.ts", "src/util.ts", "logo.png",
				"README.md"},
			[]string{"feature: orbit", "1 note · 1 draft · 0 ready · 2 open questions", "+ plan.md ?2", "  app.ts"}, nil},
		{"With notes only, the feature's notes and their folders", []string{"f", "c", "a", "m", "enter", "n"},
			"camera", []string{"src", "src/app.ts"}, []string{"feature: camera", "[notes only]"}, nil},
		{"Picking a feature in notes only", []string{"n", "f", "o", "r", "enter"}, "orbit",
			[]string{"docs", "docs/plan.md"}, nil, nil},
		{"all features clears the filter", []string{"f", "c", "a", "m", "enter", "f", "enter"}, "",
			[]string{"docs", "docs/guide.md", "docs/plan.md", "src", "src/new", "src/new/orbit.ts", "src/app.ts",
				"src/util.ts", "logo.png", "README.md"},
			[]string{"3 notes"}, []string{"feature:"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			model, _ := press(featureModel(t), tt.keys...)

			if model.feature != tt.feature || !slices.Equal(visiblePaths(model), tt.visible) {
				t.Errorf("feature %q, visible %q, want %q, %q", model.feature, visiblePaths(model), tt.feature,
					tt.visible)
			}
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

func TestFeatureChoices(t *testing.T) {
	model, _ := press(featureModel(t), "f")

	want := []string{"all features", "camera", "orbit", "sketch-explorer"}
	if !slices.Equal(model.picker.choices, want) {
		t.Errorf("choices = %q, want %q", model.picker.choices, want)
	}
	if view := ansi.Strip(model.View().Content); !strings.Contains(view, "filter to feature") {
		t.Errorf("View() lacks the picker:\n%s", view)
	}
}

func TestFeatureTixFails(t *testing.T) {
	model := featureModel(t)
	model.listFeatures = func() ([]string, error) { return nil, os.ErrPermission }

	model, _ = press(model, "f")

	if model.picker.kind != pickNone || model.status != os.ErrPermission.Error() {
		t.Errorf("picker %+v, status %q, want the error", model.picker, model.status)
	}
}

func TestFeatureLinksNewNotes(t *testing.T) {
	model := featureModel(t)
	recordOpens(&model)

	model, _ = press(model, "f", "c", "a", "m", "enter", "j", "a", "useOrbit.ts", "enter")

	if text := readNote(t, model, "src/useOrbit.ts.sketch.md"); text != "---\nfeature: camera\nstatus: draft\n---\n" {
		t.Errorf("note = %q, want it linked to camera", text)
	}
	if model.selected().Path != "src/useOrbit.ts" {
		t.Errorf("selected = %q, want the new note, which the filter shows", model.selected().Path)
	}
}

func TestLink(t *testing.T) {
	tests := []struct {
		name     string
		keys     []string
		notePath string
		text     string
		status   string
	}{
		{"L adds front matter to a note with none", []string{"j", "j", "j", "L", "c", "a", "m", "enter"},
			"src/new/orbit.ts.sketch.md", "---\nfeature: camera\n---\norbit note\n", "linked src/new/orbit.ts to camera"},
		{"L replaces a note's feature", []string{"j", "j", "j", "j", "L", "o", "r", "enter"},
			"src/app.ts.sketch.md", "---\nfeature: orbit\n---\napp note\n? why\n", "linked src/app.ts to orbit"},
		{"esc leaves the note as it is", []string{"j", "j", "j", "L", "esc"},
			"src/new/orbit.ts.sketch.md", "orbit note\n", ""},
		{"L on a file with no note says so", []string{"j", "j", "j", "j", "j", "L"},
			"src/new/orbit.ts.sketch.md", "orbit note\n", linkNeedsNote},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			model, _ := press(featureModel(t), tt.keys...)

			if text := readNote(t, model, tt.notePath); text != tt.text {
				t.Errorf("note = %q, want %q", text, tt.text)
			}
			if model.status != tt.status {
				t.Errorf("status = %q, want %q", model.status, tt.status)
			}
		})
	}
}

func TestLinkShowsUnderTheFilter(t *testing.T) {
	model, _ := press(featureModel(t), "j", "j", "j", "L", "c", "a", "m", "enter", "f", "c", "a", "m", "enter")

	if !slices.Contains(visiblePaths(model), "src/new/orbit.ts") {
		t.Errorf("visible = %q, want the linked ghost under the camera filter", visiblePaths(model))
	}
}

// questionModel is newModel with open questions in the notes on src/app.ts, src/util.ts and README.md.
func questionModel(t *testing.T) Model {
	t.Helper()
	model := newModel(t)
	writeFile(t, model.root, "src/util.ts.sketch.md", "? how\n")
	writeFile(t, model.root, "README.md.sketch.md", "? who\n")
	model.reload("docs")
	return model
}

func TestNextQuestion(t *testing.T) {
	tests := []struct {
		name     string
		keys     []string
		selected string
	}{
		{"] moves to the first note with a question", []string{"]"}, "src/app.ts"},
		{"] again moves to the next", []string{"]", "]"}, "src/util.ts"},
		{"] goes on past the folders", []string{"]", "]", "]"}, "README.md"},
		{"] wraps around from the last", []string{"]", "]", "]", "]"}, "src/app.ts"},
		{"] opens the folders above a closed note", []string{"j", "h", "]"}, "src/app.ts"},
		{"] works in notes only", []string{"n", "]", "]"}, "src/util.ts"},
		{"[ moves back, wrapping from the first to the last", []string{"["}, "README.md"},
		{"[ again moves to the one before", []string{"[", "["}, "src/util.ts"},
		{"[ undoes ]", []string{"]", "]", "["}, "src/app.ts"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			model, _ := press(questionModel(t), tt.keys...)

			if selected := model.selected().Path; selected != tt.selected {
				t.Errorf("selected = %q, want %q", selected, tt.selected)
			}
		})
	}
}

func TestNextQuestionSkipsOtherFeatures(t *testing.T) {
	model, _ := press(featureModel(t), "f", "o", "r", "enter", "]", "[")

	if model.selected().Path != "docs/plan.md" || model.status != "" {
		t.Errorf("selected %q, status %q, want docs/plan.md every time", model.selected().Path, model.status)
	}
}

func TestNextQuestionWithoutAny(t *testing.T) {
	model := newModel(t)
	writeFile(t, model.root, "src/app.ts.sketch.md", "? why\n> because\n")
	model.reload("docs")

	model, _ = press(model, "]")

	if model.selected().Path != "docs" || model.status != noOpenQuestions {
		t.Errorf("selected %q, status %q, want docs and %q", model.selected().Path, model.status, noOpenQuestions)
	}
}

func TestPaste(t *testing.T) {
	t.Run("Into the name prompt, on one line", func(t *testing.T) {
		model, _ := press(newModel(t), "j", "a", "x")

		model = paste(model, "use\nOrbit.ts\r\n")

		if model.prompt.typed != "xuseOrbit.ts" {
			t.Errorf("typed = %q, want %q", model.prompt.typed, "xuseOrbit.ts")
		}
	})
	t.Run("Into the picker's filter", func(t *testing.T) {
		model, _ := press(selectPath(t, newModel(t), "src/app.ts"), "r", "down")

		model = paste(model, "util\n")

		if model.picker.typed != "util" || model.picker.cursor != 0 {
			t.Errorf("typed %q, cursor %d, want util at the first match", model.picker.typed, model.picker.cursor)
		}
	})
	t.Run("Anywhere else, nothing", func(t *testing.T) {
		model := paste(newModel(t), "q")

		if model.selected().Path != "docs" || model.status != "" {
			t.Errorf("selected %q, status %q, want nothing to happen", model.selected().Path, model.status)
		}
	})
}

func TestHeaderPlurals(t *testing.T) {
	model := newModel(t)
	if err := os.Remove(model.filePath("src/new/orbit.ts.sketch.md")); err != nil {
		t.Fatal(err)
	}
	model.reload("docs")

	if view := ansi.Strip(model.View().Content); !strings.Contains(view, "trial  1 note · 1 draft · 0 ready · 1 open question") {
		t.Errorf("View() lacks the singular counts:\n%s", view)
	}
}

func TestReload(t *testing.T) {
	tests := []struct {
		name    string
		message tea.Msg
	}{
		{"Regaining focus reloads", tea.FocusMsg{}},
		{"R reloads", tea.KeyPressMsg{Code: 'R', Text: "R"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			model, _ := press(newModel(t), "j", "j", "j", "j")
			writeFile(t, model.root, "src/app.ts.sketch.md", "app note\n? why\n? who\n")

			updated, _ := model.Update(tt.message)
			model = updated.(Model)

			if selected := model.selected(); selected.Path != "src/app.ts" || selected.OpenQuestions != 2 {
				t.Errorf("selected = %s with %d open questions, want src/app.ts with 2", selected.Path,
					selected.OpenQuestions)
			}
		})
	}
}

func TestViewReportsFocus(t *testing.T) {
	if !newModel(t).View().ReportFocus {
		t.Error("View() doesn't ask the terminal to report focus")
	}
}

func TestUnreadableNote(t *testing.T) {
	model := newModel(t)
	opened := recordOpens(&model)
	writeFile(t, model.root, "src/util.ts.sketch.md", "---\nfeature: [\n")

	model, _ = press(model, "j", "j", "j", "j", "j", "R")

	view := ansi.Strip(model.View().Content)
	for _, text := range []string{"3 notes · 2 draft · 0 ready · 1 open question · 0 dangling  1 unreadable", "! util.ts",
		"1 note couldn't be read: src/util.ts.sketch.md: bad front matter"} {
		if !strings.Contains(view, text) {
			t.Errorf("View() lacks %q:\n%s", text, view)
		}
	}
	if len(model.Problems()) != 1 {
		t.Errorf("Problems() = %v, want one", model.Problems())
	}
	model, command := press(model, "e")
	if command == nil {
		t.Fatal("e opened nothing")
	}
	if want := []string{model.filePath("src/util.ts.sketch.md")}; !slices.Equal(*opened, want) {
		t.Errorf("opened %q, want %q", *opened, want)
	}
}

func TestAddOnFolderWithNote(t *testing.T) {
	model := newModel(t)
	opened := recordOpens(&model)
	writeFile(t, model.root, "src.sketch.md", "src note\n")
	model.reload("src")

	model, _ = press(model, "a")

	if model.prompt.kind != promptName || model.prompt.path != "src" {
		t.Errorf("prompt = %+v, want a name in src", model.prompt)
	}
	model, _ = press(model, "esc", "e")
	if want := []string{model.filePath("src.sketch.md")}; !slices.Equal(*opened, want) {
		t.Errorf("e opened %q, want %q", *opened, want)
	}
}

func TestReady(t *testing.T) {
	tests := []struct {
		name     string
		before   string
		keys     []string
		notePath string
		text     string
		status   string
	}{
		{"y marks a note with no front matter ready", "", []string{"j", "j", "j", "y"},
			"src/new/orbit.ts.sketch.md", "---\nstatus: ready\n---\norbit note\n", "src/new/orbit.ts is ready"},
		{"y marks a draft note ready, keeping its feature", "---\nfeature: camera\nstatus: draft\n---\norbit note\n",
			[]string{"j", "j", "j", "y"}, "src/new/orbit.ts.sketch.md",
			"---\nfeature: camera\nstatus: ready\n---\norbit note\n", "src/new/orbit.ts is ready"},
		{"y again turns it back to draft", "", []string{"j", "j", "j", "y", "y"},
			"src/new/orbit.ts.sketch.md", "---\nstatus: draft\n---\norbit note\n", "src/new/orbit.ts is draft"},
		{"y on a file with no note says so", "", []string{"j", "j", "j", "j", "j", "y"},
			"src/new/orbit.ts.sketch.md", "orbit note\n", readyNeedsNote},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			model := newModel(t)
			if tt.before != "" {
				writeFile(t, model.root, tt.notePath, tt.before)
				model.reload("docs")
			}

			model, _ = press(model, tt.keys...)

			if text := readNote(t, model, tt.notePath); text != tt.text {
				t.Errorf("note = %q, want %q", text, tt.text)
			}
			if model.status != tt.status {
				t.Errorf("status = %q, want %q", model.status, tt.status)
			}
		})
	}
}

func TestReadyOnUnreadableNote(t *testing.T) {
	model := newModel(t)
	writeFile(t, model.root, "src/new/orbit.ts.sketch.md", "---\nfeature: [\n")
	model.reload("docs")

	model, _ = press(model, "j", "j", "j", "y")

	if text := readNote(t, model, "src/new/orbit.ts.sketch.md"); text != "---\nfeature: [\n" {
		t.Errorf("note = %q, want it left as it is", text)
	}
	if !strings.Contains(model.status, "bad front matter") {
		t.Errorf("status = %q, want the note's problem", model.status)
	}
}

func TestHeaderReadiness(t *testing.T) {
	model := newModel(t)
	writeFile(t, model.root, "src/app.ts.sketch.md", "---\nstatus: ready\n---\n-> src/util.ts\n-> src/gone.ts\n")
	writeFile(t, model.root, "src/new/orbit.ts.sketch.md", "---\nstatus: draft\n---\n-> src/nowhere.ts\n")
	writeFile(t, model.root, "README.md.sketch.md", "no status\n-> src/new/orbit.ts\n")
	model.reload("docs")

	want := "trial  3 notes · 2 draft · 1 ready · 0 open questions · 2 dangling"
	if view := ansi.Strip(model.View().Content); !strings.Contains(view, want) {
		t.Errorf("View() lacks %q:\n%s", want, view)
	}
}

func TestDraftsDim(t *testing.T) {
	model := newModel(t)
	writeFile(t, model.root, "src/app.ts.sketch.md", "---\nstatus: ready\n---\n")
	writeFile(t, model.root, "src/new/orbit.ts.sketch.md", "---\nstatus: ready\n---\n")
	writeFile(t, model.root, "src/util.ts.sketch.md", "---\nstatus: draft\n---\n")
	writeFile(t, model.root, "README.md.sketch.md", "no status\n")
	model.reload("docs")
	tests := []struct {
		path  string
		name  string
		style lipgloss.Style
		dim   bool
	}{
		{"src/app.ts", "app.ts", lipgloss.NewStyle(), false},
		{"src/new/orbit.ts", "orbit.ts", model.styles.planned, false},
		{"src/util.ts", "util.ts", model.styles.draft, true},
		{"README.md", "README.md", model.styles.draft, true},
	}
	for _, tt := range tests {
		t.Run(tt.path, func(t *testing.T) {
			row := model.renderRow(model.rowOf(tt.path), 40)

			if !strings.Contains(row, tt.style.Render(tt.name)) {
				t.Errorf("row %q doesn't draw %s as %q", row, tt.name, tt.style.Render(tt.name))
			}
			if dimmed := strings.Contains(row, model.styles.draft.Render(tt.name)); dimmed != tt.dim {
				t.Errorf("row %q dimmed = %v, want %v", row, dimmed, tt.dim)
			}
		})
	}
}

func TestTabCompletesFolders(t *testing.T) {
	tests := []struct {
		name     string
		selected string
		keys     []string
		typed    string
	}{
		{"A ghost folder completes with a slash", "src", []string{"a", "n", "tab"}, "new/"},
		{"No match leaves the name", "src", []string{"a", "x", "tab"}, "x"},
		{"Files don't complete", "src", []string{"a", "app", "tab"}, "app"},
		{"Several matches complete as far as they agree", "src", []string{"a", "c", "tab"}, "compo"},
		{"Nothing typed and folders that don't agree stay as typed", "src", []string{"a", "tab"}, ""},
		{"A typed folder completes inside it", "src", []string{"a", "components/b", "tab"}, "components/button/"},
		{"At the root, top-level folders", "README.md", []string{"a", "d", "tab"}, "docs/"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			model := newModel(t)
			writeFile(t, model.root, "src/components/button/b.ts", "b\n")
			writeFile(t, model.root, "src/composables/c.ts", "c\n")
			model.reload("docs")
			model = selectPath(t, model, tt.selected)

			model, _ = press(model, tt.keys...)

			if model.prompt.typed != tt.typed {
				t.Errorf("typed = %q, want %q", model.prompt.typed, tt.typed)
			}
		})
	}
}

func TestAIsGone(t *testing.T) {
	model := newModel(t)
	opened := recordOpens(&model)

	model, command := press(model, "j", "j", "j", "j", "j", "A")

	if command != nil || len(*opened) > 0 {
		t.Errorf("A gave %v and opened %q, want nothing", command, *opened)
	}
	if _, err := os.Stat(model.filePath("src/util.ts.sketch.md")); !os.IsNotExist(err) {
		t.Errorf("A made a note: %v", err)
	}
}

func TestEnterOnFolderDoesNotEdit(t *testing.T) {
	model := newModel(t)
	opened := recordOpens(&model)
	writeFile(t, model.root, "src.sketch.md", "src note\n")
	model.reload("src")

	model, _ = press(model, "enter")

	if len(*opened) > 0 || model.expanded["src"] {
		t.Errorf("opened %q, src open %v, want src closed and nothing opened", *opened, model.expanded["src"])
	}
}

func TestHelp(t *testing.T) {
	model, _ := press(newModel(t), "?")

	view := ansi.Strip(model.View().Content)
	for _, binding := range helpKeys {
		if line := fmt.Sprintf(helpLine, binding.Help().Key, binding.Help().Desc); !strings.Contains(view, line) {
			t.Errorf("help lacks %q:\n%s", line, view)
		}
	}
	for _, name := range []string{"esc", "q", "j"} {
		t.Run(name+" closes it and does nothing else", func(t *testing.T) {
			closed, command := press(model, name)

			if closed.help || command != nil || closed.selected().Path != "docs" {
				t.Errorf("help %v, command %v, selected %q, want help closed, nothing else", closed.help, command,
					closed.selected().Path)
			}
			if view := ansi.Strip(closed.View().Content); strings.Contains(view, helpTitle) {
				t.Errorf("View() still shows help:\n%s", view)
			}
		})
	}
}

func TestAnswer(t *testing.T) {
	tests := []struct {
		name string
		note string
		text string
		line int
	}{
		{"Under the only question", "app note\n? why\n", "app note\n? why\n> \n", 3},
		{"Under the first open one, past answered ones", "---\nstatus: draft\n---\n? what\n> this\n? why\n? who\n",
			"---\nstatus: draft\n---\n? what\n> this\n? why\n> \n? who\n", 7},
		{"With no last newline", "? why", "? why\n> ", 2},
		{"On an empty answer already there, adding none", "? why\n> \nlater\n", "? why\n> \nlater\n", 2},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			model := newModel(t)
			writeFile(t, model.root, "src/app.ts.sketch.md", tt.note)
			model.reload("src/app.ts")
			var opened []string
			var lines []int
			model.open = func(filePath string, line int, done func(error) tea.Msg) tea.Cmd {
				opened = append(opened, filePath)
				lines = append(lines, line)
				return func() tea.Msg { return done(nil) }
			}

			model, _ = press(model, ">")

			if text := readNote(t, model, "src/app.ts.sketch.md"); text != tt.text {
				t.Errorf("note = %q, want %q", text, tt.text)
			}
			if want := []string{model.filePath("src/app.ts.sketch.md")}; !slices.Equal(opened, want) ||
				!slices.Equal(lines, []int{tt.line}) {
				t.Errorf("opened %q on %v, want %q on %d", opened, lines, want, tt.line)
			}
		})
	}
}

func TestAnswerWithoutOpenQuestion(t *testing.T) {
	tests := []struct {
		name     string
		selected string
	}{
		{"A note with none", "src/new/orbit.ts"},
		{"A file with no note", "src/util.ts"},
		{"A folder", "src"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			model := selectPath(t, newModel(t), tt.selected)
			opened := recordOpens(&model)

			model, command := press(model, ">")

			if command != nil || len(*opened) > 0 || model.status != noneToAnswer {
				t.Errorf("command %v, opened %q, status %q, want only %q", command, *opened, model.status, noneToAnswer)
			}
			if text := readNote(t, model, "src/new/orbit.ts.sketch.md"); text != "orbit note\n" {
				t.Errorf("orbit note = %q, want it untouched", text)
			}
		})
	}
}

// gitRepo writes files to a new git repo on branch main and returns its root.
func gitRepo(t *testing.T) string {
	t.Helper()
	root := filepath.Join(t.TempDir(), "trial")
	for file, text := range files {
		writeFile(t, root, file, text)
	}
	writeFile(t, root, "src/app.ts.sketch.md", "---\nfeature: camera\n---\napp note\n")
	command := exec.Command("git", "init", "--quiet", "--initial-branch", "main")
	command.Dir = root
	if output, err := command.CombinedOutput(); err != nil {
		t.Fatalf("git init: %v\n%s", err, output)
	}
	return root
}

func openRepo(t *testing.T, root string) Model {
	t.Helper()
	model, err := New(root)
	if err != nil {
		t.Fatal(err)
	}
	model.listFeatures = func() ([]string, error) { return nil, nil }
	return model
}

func TestFeatureSticksPerBranch(t *testing.T) {
	tests := []struct {
		name    string
		keys    []string
		feature string
	}{
		{"A feature picked is there on reopening", []string{"f", "c", "a", "m", "enter"}, "camera"},
		{"Clearing it clears it on reopening", []string{"f", "c", "a", "m", "enter", "f", "enter"}, ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			root := gitRepo(t)

			_, _ = press(openRepo(t, root), tt.keys...)
			reopened := openRepo(t, root)

			if reopened.feature != tt.feature {
				t.Errorf("feature on reopening = %q, want %q", reopened.feature, tt.feature)
			}
			wantNotes := map[string]int{"camera": 1, "": 2}[tt.feature]
			if reopened.tree.NotesBelow != wantNotes {
				t.Errorf("%d notes show on reopening, want %d", reopened.tree.NotesBelow, wantNotes)
			}
			kept := filepath.Join(root, ".git", "sketch", "filter", "main")
			if _, err := os.Stat(kept); (err == nil) != (tt.feature != "") {
				t.Errorf("stat %s: %v, want it kept only while the filter is set", kept, err)
			}
		})
	}
}

func TestFeatureDoesNotStickOnOtherBranches(t *testing.T) {
	root := gitRepo(t)
	_, _ = press(openRepo(t, root), "f", "c", "a", "m", "enter")
	command := exec.Command("git", "checkout", "--quiet", "-b", "other")
	command.Dir = root
	if output, err := command.CombinedOutput(); err != nil {
		t.Fatalf("git checkout: %v\n%s", err, output)
	}

	if feature := openRepo(t, root).feature; feature != "" {
		t.Errorf("feature on another branch = %q, want none", feature)
	}
}

// checkout switches the repo at root to a new branch.
func checkout(t *testing.T, root, branch string) {
	t.Helper()
	command := exec.Command("git", "checkout", "--quiet", "-b", branch)
	command.Dir = root
	if output, err := command.CombinedOutput(); err != nil {
		t.Fatalf("git checkout: %v\n%s", err, output)
	}
}

func TestFeatureSticksOnBranchesWithSlashes(t *testing.T) {
	tests := []struct {
		name     string
		branches []string // each checked out in turn, with the filter picked on each
	}{
		{"A branch with a slash", []string{"feature/x"}},
		{"A branch named like the folder of a later one", []string{"feature", "feature/x"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			root := gitRepo(t)
			for _, branch := range tt.branches {
				checkout(t, root, branch)

				model, _ := press(openRepo(t, root), "f", "c", "a", "m", "enter")

				if model.status != "" {
					t.Errorf("status on %s = %q, want none", branch, model.status)
				}
				if feature := openRepo(t, root).feature; feature != "camera" {
					t.Errorf("feature on reopening %s = %q, want camera", branch, feature)
				}
			}
		})
	}
}

func TestFeatureOutsideGitIsNotKept(t *testing.T) {
	if model := featureModel(t); model.featureFile != "" {
		t.Errorf("featureFile = %q outside git, want none", model.featureFile)
	}
}

func TestOpensNotesOnly(t *testing.T) {
	tests := []struct {
		name      string
		files     map[string]string
		notesOnly bool
	}{
		{"With notes", map[string]string{"src/app.ts": "app\n", "src/app.ts.sketch.md": "note\n"}, true},
		{"Without notes", map[string]string{"src/app.ts": "app\n"}, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			root := t.TempDir()
			for file, text := range tt.files {
				writeFile(t, root, file, text)
			}

			model, err := New(root)
			if err != nil {
				t.Fatal(err)
			}

			if model.notesOnly != tt.notesOnly {
				t.Errorf("notesOnly = %v, want %v", model.notesOnly, tt.notesOnly)
			}
		})
	}
}
