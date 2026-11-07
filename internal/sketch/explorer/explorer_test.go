package explorer

import (
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
	"github.com/charmbracelet/x/ansi"
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
	model.open = func(string, func(error) tea.Msg) tea.Cmd {
		t.Error("a note opened outside the test's editor")
		return nil
	}
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
			[]string{"trial  2 notes · 1 open question", "q quit", "n notes ·", "R reload", "▸   docs/", "▾   src/", "+ orbit.ts",
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
	model.open = func(filePath string, done func(error) tea.Msg) tea.Cmd {
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
			"src/useOrbit.ts.sketch.md", "# useOrbit.ts\n\n", "src/useOrbit.ts", "+"},
		{"A name with folders makes them and opens them in the tree", "",
			[]string{"j", "a", "composables/useOrbit.ts", "enter"},
			"src/composables/useOrbit.ts.sketch.md", "# useOrbit.ts\n\n", "src/composables/useOrbit.ts", "+"},
		{"a on a file plans one in its folder", "", []string{"j", "j", "j", "j", "j", "a", "x.ts", "enter"},
			"src/x.ts.sketch.md", "# x.ts\n\n", "src/x.ts", "+"},
		{"a on a top-level file plans one at the root", "", []string{"j", "j", "j", "j", "j", "j", "j", "a", "x.ts", "enter"},
			"x.ts.sketch.md", "# x.ts\n\n", "x.ts", "+"},
		{"backspace takes back a letter", "", []string{"j", "a", "useOrbix", "backspace", "t.ts", "enter"},
			"src/useOrbit.ts.sketch.md", "# useOrbit.ts\n\n", "src/useOrbit.ts", "+"},
		{"A feature filter goes in front matter", "orbit", []string{"j", "a", "useOrbit.ts", "enter"},
			"src/useOrbit.ts.sketch.md", "---\nfeature: orbit\n---\n# useOrbit.ts\n\n", "src/useOrbit.ts", "+"},
		{"A on a file attaches a note to it", "", []string{"j", "j", "j", "j", "j", "A"},
			"src/util.ts.sketch.md", "# util.ts\n\n", "src/util.ts", "~"},
		{"a on a node with a note opens it", "", []string{"j", "j", "j", "a"},
			"src/new/orbit.ts.sketch.md", "orbit note\n", "src/new/orbit.ts", "+"},
		{"A on a node with a note opens it", "", []string{"j", "j", "j", "j", "A"},
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
		{"A note's name", "x.ts.sketch.md", ErrNoteName},
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
	if view := ansi.Strip(model.View().Content); !strings.Contains(view, "q quit") {
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
	if view := ansi.Strip(model.View().Content); !strings.Contains(view, "new file in src/: qdj") {
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

func TestEditWithoutNote(t *testing.T) {
	model := newModel(t)
	opened := recordOpens(&model)

	model, command := press(model, "j", "j", "j", "j", "j", "e")

	if command != nil || len(*opened) > 0 || model.status != noNoteToEdit {
		t.Errorf("command %v, opened %q, status %q, want only %q", command, *opened, model.status, noNoteToEdit)
	}
}

func TestEditorFails(t *testing.T) {
	model := newModel(t)
	model.open = func(_ string, done func(error) tea.Msg) tea.Cmd {
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
			list := picker{kind: pickName, choices: choices}
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

// cameraModel is newModel with an attached note on src/useCamera.ts, whose text is note.
func cameraModel(t *testing.T, note string) Model {
	t.Helper()
	model := newModel(t)
	writeFile(t, model.root, "src/useCamera.ts", "export function make_camera() {\n\treturn new Camera(make_camera)\n}\n")
	writeFile(t, model.root, "src/useCamera.ts.sketch.md", note)
	model.reload("src/useCamera.ts")
	return model
}

func TestRef(t *testing.T) {
	tests := []struct {
		name string
		note string
		text string
	}{
		{"enter adds the pick on a line of its own", "camera note\n", "camera note\n- uses `make_camera`\n"},
		{"A note without a last newline gets one first", "camera note", "camera note\n- uses `make_camera`\n"},
		{"An empty note gets just the line", "", "- uses `make_camera`\n"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			model := cameraModel(t, tt.note)

			model, _ = press(model, "r")
			if want := []string{"Camera", "export", "function", "make_camera", "new", "return"}; !slices.Equal(
				model.picker.choices, want) {
				t.Fatalf("choices = %q, want %q", model.picker.choices, want)
			}
			model, _ = press(model, "m", "a", "k", "e", "_")
			if view := ansi.Strip(model.View().Content); !strings.Contains(view, "> make_camera") ||
				strings.Contains(view, "export") {
				t.Errorf("View() lacks the filtered picker:\n%s", view)
			}
			model, _ = press(model, "enter")

			if text := readNote(t, model, "src/useCamera.ts.sketch.md"); text != tt.text {
				t.Errorf("note = %q, want %q", text, tt.text)
			}
			if model.picker.kind != pickNone || model.selected().Path != "src/useCamera.ts" {
				t.Errorf("picker %+v, selected %q, want the picker closed on the note", model.picker,
					model.selected().Path)
			}
			if view := ansi.Strip(model.View().Content); !strings.Contains(view, "- uses `make_camera`") {
				t.Errorf("the preview lacks the new line:\n%s", view)
			}
		})
	}
}

func TestRefEscCancels(t *testing.T) {
	model := cameraModel(t, "camera note\n")

	model, _ = press(model, "r", "m", "esc")

	if model.picker.kind != pickNone || readNote(t, model, "src/useCamera.ts.sketch.md") != "camera note\n" {
		t.Errorf("picker %+v, want it closed and the note untouched", model.picker)
	}
}

func TestRefRejects(t *testing.T) {
	tests := []struct {
		name     string
		selected string
	}{
		{"A ghost note", "src/new/orbit.ts"},
		{"A file without a note", "src/util.ts"},
		{"A folder", "src"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			model := selectPath(t, newModel(t), tt.selected)

			model, _ = press(model, "r")

			if model.picker.kind != pickNone || model.status != refNeedsAttachedNote {
				t.Errorf("picker %+v, status %q, want only %q", model.picker, model.status, refNeedsAttachedNote)
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
			[]string{"3 notes · 3 open questions"}, []string{"feature:"}},
		{"A feature hides other features' notes and ghosts", []string{"f", "c", "a", "m", "enter"}, "camera",
			[]string{"docs", "src", "src/app.ts", "src/util.ts", "logo.png", "README.md"},
			[]string{"feature: camera", "1 note · 1 open question", "~ app.ts ?1"}, []string{"plan.md", "orbit.ts"}},
		{"A feature only notes name", []string{"f", "o", "r", "enter"}, "orbit",
			[]string{"docs", "docs/guide.md", "docs/plan.md", "src", "src/app.ts", "src/util.ts", "logo.png",
				"README.md"},
			[]string{"feature: orbit", "1 note · 2 open questions", "+ plan.md ?2", "  app.ts"}, nil},
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

	if text := readNote(t, model, "src/useOrbit.ts.sketch.md"); text != "---\nfeature: camera\n---\n# useOrbit.ts\n\n" {
		t.Errorf("note = %q, want it linked to camera", text)
	}
	if model.selected().Path != "src/useOrbit.ts" {
		t.Errorf("selected = %q, want the new note, which the filter shows", model.selected().Path)
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
		{"? moves to the first note with a question", []string{"?"}, "src/app.ts"},
		{"? again moves to the next", []string{"?", "?"}, "src/util.ts"},
		{"? goes on past the folders", []string{"?", "?", "?"}, "README.md"},
		{"? wraps around from the last", []string{"?", "?", "?", "?"}, "src/app.ts"},
		{"? opens the folders above a closed note", []string{"j", "h", "?"}, "src/app.ts"},
		{"? works in notes only", []string{"n", "?", "?"}, "src/util.ts"},
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
	model, _ := press(featureModel(t), "f", "o", "r", "enter", "?", "?")

	if model.selected().Path != "docs/plan.md" || model.status != "" {
		t.Errorf("selected %q, status %q, want docs/plan.md every time", model.selected().Path, model.status)
	}
}

func TestNextQuestionWithoutAny(t *testing.T) {
	model := newModel(t)
	writeFile(t, model.root, "src/app.ts.sketch.md", "? why\n> because\n")
	model.reload("docs")

	model, _ = press(model, "?")

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
		model, _ := press(cameraModel(t, ""), "r", "down")

		model = paste(model, "make\n")

		if model.picker.typed != "make" || model.picker.cursor != 0 {
			t.Errorf("typed %q, cursor %d, want make at the first match", model.picker.typed, model.picker.cursor)
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

	if view := ansi.Strip(model.View().Content); !strings.Contains(view, "trial  1 note · 1 open question") {
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
	for _, text := range []string{"3 notes · 1 open question  1 unreadable", "! util.ts",
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
