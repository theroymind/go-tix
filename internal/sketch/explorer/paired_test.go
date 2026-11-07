package explorer

import (
	"slices"
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
	"github.com/charmbracelet/x/ansi"
)

// pairedModel is newModel paired with an nvim whose follows it records.
func pairedModel(t *testing.T) (Model, *[]string) {
	t.Helper()
	model := newModel(t)
	model.paired = true
	followed := []string{}
	model.follow = func(filePath string) tea.Cmd {
		followed = append(followed, filePath)
		return nil
	}
	return model, &followed
}

// run runs command, and each command in a batch, and returns the messages they send.
func run(command tea.Cmd) []tea.Msg {
	if command == nil {
		return nil
	}
	message := command()
	batch, isBatch := message.(tea.BatchMsg)
	if !isBatch {
		return []tea.Msg{message}
	}
	var messages []tea.Msg
	for _, each := range batch {
		messages = append(messages, run(each)...)
	}
	return messages
}

func TestPairedLayout(t *testing.T) {
	tests := []struct {
		name     string
		paired   bool
		keys     []string
		contains []string
		excludes []string
	}{
		{"Unpaired, the preview shows beside the tree", false, []string{"j", "j", "j"},
			[]string{"+ orbit.ts", "orbit note"}, nil},
		{"Paired, the tree shows alone", true, []string{"j", "j", "j"},
			[]string{"+ orbit.ts", "README.md"}, []string{"orbit note", "│"}},
		{"Paired, help takes the tree's place", true, []string{"?"},
			[]string{helpTitle}, []string{"README.md"}},
		{"Paired, a picker takes the tree's place", true, []string{"f"},
			[]string{featureTitle}, []string{"README.md"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			model := newModel(t)
			model.paired = tt.paired
			model.listFeatures = func() ([]string, error) { return nil, nil }
			model, _ = press(model, tt.keys...)

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

func TestFollow(t *testing.T) {
	tests := []struct {
		name string
		keys []string
		want []string
	}{
		{"A note the cursor rests on", []string{"j", "j", "j"}, []string{"src/new/orbit.ts.sketch.md"}},
		{"Only the note the cursor ends on", []string{"j", "j", "j", "j"}, []string{"src/app.ts.sketch.md"}},
		{"Back on the note shown, nothing more", []string{"j", "j", "j", "j", "k"}, []string{"src/new/orbit.ts.sketch.md"}},
		{"Not a folder", []string{"j"}, nil},
		{"Not a file without a note", []string{"j", "j", "j", "j", "j"}, nil},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			model, followed := pairedModel(t)
			var commands []tea.Cmd
			for _, name := range tt.keys {
				var command tea.Cmd
				model, command = press(model, name)
				commands = append(commands, command)
			}

			// The ticks fire after every key has landed, as when j is held down.
			for _, command := range commands {
				for _, message := range run(command) {
					updated, _ := model.Update(message)
					model = updated.(Model)
				}
			}

			var want []string
			for _, notePath := range tt.want {
				want = append(want, model.filePath(notePath))
			}
			if !slices.Equal(*followed, want) {
				t.Errorf("followed %q, want %q", *followed, want)
			}
		})
	}
}

func TestPairedFocus(t *testing.T) {
	tests := []struct {
		name     string
		keys     []string
		wantLine int
	}{
		{"enter", []string{"j", "j", "j", "j", "enter"}, 0},
		{"e", []string{"j", "j", "j", "j", "e"}, 0},
		{"a", []string{"j", "j", "j", "j", "a"}, 0},
		{">", []string{"j", "j", "j", "j", ">"}, 3},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			model, followed := pairedModel(t)
			type opening struct {
				filePath string
				line     int
			}
			var opened []opening
			model.open = func(filePath string, line int, done func(error) tea.Msg) tea.Cmd {
				opened = append(opened, opening{filePath, line})
				return func() tea.Msg { return done(nil) }
			}

			model, command := press(model, tt.keys...)

			want := []opening{{model.filePath("src/app.ts.sketch.md"), tt.wantLine}}
			if !slices.Equal(opened, want) {
				t.Errorf("opened %v, want %v", opened, want)
			}
			for _, message := range run(command) {
				if _, isFollow := message.(followMsg); isFollow {
					t.Errorf("focusing %s also follows it", tt.name)
				}
			}
			if len(*followed) > 0 {
				t.Errorf("followed %q, want nothing", *followed)
			}
		})
	}
}
