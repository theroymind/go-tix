package explorer

import (
	"os"
	"slices"
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
	"github.com/charmbracelet/x/ansi"
)

// padModel is newModel opening notes in the preview pane, as it does outside nvim.
func padModel(t *testing.T) Model {
	t.Helper()
	model := newModel(t)
	model.open = paneOpen
	return model
}

// openPad presses keys and runs the command they return, which opens a note in the pad.
func openPad(t *testing.T, model Model, keys ...string) Model {
	t.Helper()
	model, command := press(model, keys...)
	if command == nil {
		t.Fatalf("%q opened nothing; status %q", keys, model.status)
	}
	message, isPad := command().(padMsg)
	if !isPad {
		t.Fatalf("%q sent %T, want a padMsg", keys, message)
	}
	updated, _ := model.Update(message)
	return updated.(Model)
}

func TestPad(t *testing.T) {
	tests := []struct {
		name   string
		note   string // src/app.ts.sketch.md before
		open   []string
		edit   []string
		text   string // src/app.ts.sketch.md after
		active bool   // the pad still shows
		status string
	}{
		{"esc saves what's typed and goes back to the tree", "app note\n? why\n", []string{"e"},
			[]string{"X", "esc"}, "Xapp note\n? why\n", false, savedNote},
		{"ctrl+s saves and keeps editing", "app note\n? why\n", []string{"e"},
			[]string{"X", "ctrl+s"}, "Xapp note\n? why\n", true, savedNote},
		{"The tree's keys type into the note", "app note\n? why\n", []string{"enter"},
			[]string{"q", "j", "d", "esc"}, "qjdapp note\n? why\n", false, savedNote},
		{"esc on an untouched note leaves it as it was, tabs and all", "a\tb\n", []string{"e"},
			[]string{"esc"}, "a\tb\n", false, ""},
		{"> puts the cursor on the answer line", "app note\n? why\n", []string{">"},
			[]string{"b", "e", "c", "a", "u", "s", "e", "esc"}, "app note\n? why\n> because\n", false, savedNote},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			model := padModel(t)
			writeFile(t, model.root, "src/app.ts.sketch.md", tt.note)
			model.reload("src/app.ts")
			model = selectPath(t, model, "src/app.ts")
			model = openPad(t, model, tt.open...)

			model, _ = press(model, tt.edit...)

			if text := readNote(t, model, "src/app.ts.sketch.md"); text != tt.text {
				t.Errorf("note = %q, want %q", text, tt.text)
			}
			if model.pad.active() != tt.active || model.status != tt.status {
				t.Errorf("pad active %v, status %q, want %v and %q", model.pad.active(), model.status, tt.active,
					tt.status)
			}
			if model.selected().Path != "src/app.ts" {
				t.Errorf("selected = %q, want src/app.ts", model.selected().Path)
			}
		})
	}
}

func TestPadShowsBesideTree(t *testing.T) {
	model := openPad(t, padModel(t), "j", "j", "j", "e")

	model, _ = press(model, "Z")

	view := ansi.Strip(model.View().Content)
	for _, text := range []string{"orbit.ts", "app.ts", "Zorbit note", "esc save and close"} {
		if !strings.Contains(view, text) {
			t.Errorf("View() lacks %q while editing:\n%s", text, view)
		}
	}
	model, _ = press(model, "esc")
	view = ansi.Strip(model.View().Content)
	if !strings.Contains(view, "Zorbit note") || strings.Contains(view, "esc save and close") {
		t.Errorf("View() after esc doesn't show the saved note as a preview:\n%s", view)
	}
}

func TestPadPaste(t *testing.T) {
	model := openPad(t, padModel(t), "j", "j", "j", "e")

	model = paste(model, "one\ntwo ")
	model, _ = press(model, "esc")

	if text := readNote(t, model, "src/new/orbit.ts.sketch.md"); text != "one\ntwo orbit note\n" {
		t.Errorf("note = %q, want the paste at the top, newline kept", text)
	}
}

func TestPadCtrlCSavesAndQuits(t *testing.T) {
	model := openPad(t, padModel(t), "j", "j", "j", "e")

	model, command := press(model, "X", "ctrl+c")

	if command == nil {
		t.Fatal("ctrl+c returned no command, want quit")
	}
	if _, quits := command().(tea.QuitMsg); !quits {
		t.Error("ctrl+c didn't quit")
	}
	if text := readNote(t, model, "src/new/orbit.ts.sketch.md"); text != "Xorbit note\n" {
		t.Errorf("note = %q, want it saved before quitting", text)
	}
}

func TestPadKeepsOpenWhenSaveFails(t *testing.T) {
	model := openPad(t, padModel(t), "j", "j", "j", "e")
	if err := os.RemoveAll(model.filePath("src/new")); err != nil {
		t.Fatal(err)
	}

	model, _ = press(model, "X", "esc")

	if !model.pad.active() || model.status == "" || model.status == savedNote {
		t.Errorf("pad active %v, status %q, want it open with the write's error", model.pad.active(), model.status)
	}
}

func TestAddSaysWhichNote(t *testing.T) {
	tests := []struct {
		name   string
		keys   []string
		status string
	}{
		{"A new note names itself and the file it plans", []string{"j", "a", "test", "enter"},
			"created test.sketch.md, the note planning test"},
		{"Opening a note already there says nothing", []string{"j", "a", "app.ts", "enter"}, ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			model := openPad(t, padModel(t), tt.keys...)

			if model.status != tt.status {
				t.Errorf("status = %q, want %q", model.status, tt.status)
			}
		})
	}
}

func TestExternal(t *testing.T) {
	tests := []struct {
		name string
		keys []string
		note string
		text string
	}{
		{"E opens the note in $EDITOR", []string{"j", "j", "j", "j", "E"}, "src/app.ts.sketch.md", "app note\n? why\n"},
		{"E on a file without a note adds one", []string{"j", "j", "j", "j", "j", "E"}, "src/util.ts.sketch.md",
			"---\nstatus: draft\n---\n"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			model := padModel(t)
			var opened []string
			model.external = func(filePath string, _ int, done func(error) tea.Msg) tea.Cmd {
				opened = append(opened, filePath)
				return func() tea.Msg { return done(nil) }
			}

			model, command := press(model, tt.keys...)
			updated, _ := model.Update(command())
			model = updated.(Model)

			if want := []string{model.filePath(tt.note)}; !slices.Equal(opened, want) {
				t.Errorf("opened %q, want %q", opened, want)
			}
			if text := readNote(t, model, tt.note); text != tt.text {
				t.Errorf("note = %q, want %q", text, tt.text)
			}
			if model.pad.active() {
				t.Error("the pad opened too")
			}
		})
	}
}
