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
	"github.com/theroymind/go-tix/internal/sketch/marks"
	"github.com/theroymind/go-tix/internal/sketch/note"
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
	model, err := New(root, diskLoad(t, root))
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

// diskLoad reads the tree under root the way sketch does: every file, and every note marked against its target.
func diskLoad(t *testing.T, root string) Load {
	return func() (*tree.Node, error) {
		paths, err := tree.Files(root)
		if err != nil {
			return nil, err
		}
		notePaths, err := note.Find(root)
		if err != nil {
			return nil, err
		}
		reader := marks.NewReader(root)
		var notes []sketch.Note
		for _, notePath := range notePaths {
			value, err := note.Load(root, notePath)
			if err != nil {
				return nil, err
			}
			if err := reader.MarkNote(&value); err != nil {
				return nil, err
			}
			notes = append(notes, value)
		}
		return tree.Build(paths, notes), nil
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
