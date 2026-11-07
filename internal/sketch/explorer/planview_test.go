package explorer

import (
	"os"
	"slices"
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
	"github.com/charmbracelet/x/ansi"

	"github.com/theroymind/go-tix/contracts/sketch"
	"github.com/theroymind/go-tix/contracts/ticket"
	"github.com/theroymind/go-tix/internal/sketch/plan"
)

// planModel is newModel with src/app.ts's note linking to src/new/orbit.ts, so orbit.ts builds first although
// app.ts comes first by path.
func planModel(t *testing.T) Model {
	t.Helper()
	model := newModel(t)
	writeFile(t, model.root, "src/app.ts.sketch.md", "---\nstatus: ready\n---\napp note\n-> src/new/orbit.ts\n")
	model.reload("docs")
	return model
}

func planPaths(model Model) []string {
	var paths []string
	for _, node := range model.plan.notes {
		paths = append(paths, node.Path)
	}
	return paths
}

func TestPlanView(t *testing.T) {
	tests := []struct {
		name      string
		start     string
		keys      []string
		active    bool
		selected  string // the plan view's selected note, or the tree's selected node when it's left
		firstLine string // the first line the right pane shows
	}{
		{"v shows the notes in build order, from the first", "docs", []string{"v"}, true, "src/new/orbit.ts",
			"+ src/new/orbit.ts"},
		{"v starts on the tree's selected note", "src/app.ts", []string{"v"}, true, "src/app.ts", "~ src/app.ts"},
		{"j scrolls the right pane to the next note", "docs", []string{"v", "j"}, true, "src/app.ts",
			"~ src/app.ts"},
		{"k scrolls back, and stops at the top", "docs", []string{"v", "j", "k", "k"}, true, "src/new/orbit.ts",
			"+ src/new/orbit.ts"},
		{"v goes back to the node the tree was on", "src/util.ts", []string{"v", "j", "v"}, false, "src/util.ts", ""},
		{"esc goes back to the node the tree was on", "src/app.ts", []string{"v", "k", "esc"}, false, "src/app.ts",
			""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			model, _ := press(selectPath(t, planModel(t), tt.start), tt.keys...)

			if model.plan.active != tt.active {
				t.Fatalf("plan view active = %v, want %v", model.plan.active, tt.active)
			}
			if !tt.active {
				if model.selected().Path != tt.selected {
					t.Errorf("tree selected = %q, want %q", model.selected().Path, tt.selected)
				}
				return
			}
			if model.plan.selected().Path != tt.selected {
				t.Errorf("plan selected = %q, want %q", model.plan.selected().Path, tt.selected)
			}
			if line := model.plan.lines[model.plan.scroll]; line != tt.firstLine {
				t.Errorf("first line = %q, want %q", line, tt.firstLine)
			}
		})
	}
}

func TestPlanViewMatchesPlanOrder(t *testing.T) {
	model := planModel(t)
	writeFile(t, model.root, "src/util.ts.sketch.md", "util note\n-> src/app.ts\n")
	writeFile(t, model.root, "docs/guide.md.sketch.md", "guide note\n")
	model.reload("docs")

	model, _ = press(model, "v")

	var notes []sketch.Note
	for _, node := range nodes(model.tree) {
		if node.Note != nil {
			notes = append(notes, *node.Note)
		}
	}
	var want []string
	for _, value := range plan.Order(notes) {
		want = append(want, value.Target)
	}
	if got := planPaths(model); !slices.Equal(got, want) {
		t.Errorf("plan view order = %q, want plan.Order's %q", got, want)
	}
}

func TestPlanViewScrolls(t *testing.T) {
	ctrlD := tea.KeyPressMsg{Code: 'd', Mod: tea.ModCtrl}
	ctrlU := tea.KeyPressMsg{Code: 'u', Mod: tea.ModCtrl}
	tests := []struct {
		name     string
		messages []tea.Msg
		scroll   func(view planView) int
	}{
		{"ctrl+d scrolls down, stopping at the last line", []tea.Msg{ctrlD, ctrlD, ctrlD},
			func(view planView) int { return len(view.lines) - 1 }},
		{"ctrl+u scrolls up, stopping at the top", []tea.Msg{ctrlD, ctrlU, ctrlU}, func(planView) int { return 0 }},
		{"Moving after scrolling puts the note's heading at the top", []tea.Msg{ctrlD, ctrlD, tea.KeyPressMsg{Code: 'k', Text: "k"}},
			func(view planView) int { return view.headings[0] }},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			model, _ := press(planModel(t), "v")
			for _, message := range tt.messages {
				updated, _ := model.Update(message)
				model = updated.(Model)
			}

			if want := tt.scroll(model.plan); model.plan.scroll != want {
				t.Errorf("scroll = %d, want %d", model.plan.scroll, want)
			}
		})
	}
}

func TestPlanViewShows(t *testing.T) {
	model, _ := press(planModel(t), "v")

	view := ansi.Strip(model.View().Content)
	for _, text := range []string{"+ src/new/orbit.ts  →0 ←1", "~ src/app.ts  →1 ←0", "orbit note", "app note",
		"used by ← ~ app.ts", "v/esc tree", "1 draft · 1 ready"} {
		if !strings.Contains(view, text) {
			t.Errorf("View() lacks %q:\n%s", text, view)
		}
	}
	if strings.Contains(view, "a add") {
		t.Errorf("View() shows the tree's footer:\n%s", view)
	}
}

func TestPlanViewFeatureFilter(t *testing.T) {
	model, _ := press(featureModel(t), "f", "c", "a", "m", "enter", "v")

	if got := planPaths(model); !slices.Equal(got, []string{"src/app.ts"}) {
		t.Errorf("plan view notes = %q, want only camera's", got)
	}
}

func TestPlanViewWithoutNotes(t *testing.T) {
	model := newModel(t)
	for _, file := range []string{"src/app.ts.sketch.md", "src/new/orbit.ts.sketch.md"} {
		if err := os.Remove(model.filePath(file)); err != nil {
			t.Fatal(err)
		}
	}
	model.reload("docs")

	model, _ = press(model, "v")

	if model.plan.active || model.status != noNotesToPlan {
		t.Errorf("plan view active %v, status %q, want the tree and %q", model.plan.active, model.status,
			noNotesToPlan)
	}
}

func TestPlanViewTixFeature(t *testing.T) {
	camera := &ticket.Ticket{Title: "Camera", Acceptance: []ticket.Criterion{{Text: "orbits", Done: true},
		{Text: "zooms"}}}
	tests := []struct {
		name     string
		feature  string
		found    *ticket.Ticket
		err      error
		contains []string
		excludes []string
		status   string
	}{
		{"The feature's title and criteria head the notes", "camera", camera, nil,
			[]string{"Camera", "1. [x] orbits", "2. [ ] zooms", "~ src/app.ts"}, nil, ""},
		{"Without tix, just the notes", "camera", nil, nil, []string{"~ src/app.ts"}, []string{"Camera"}, ""},
		{"When tix fails, the notes and the error", "camera", nil, os.ErrPermission, []string{"~ src/app.ts"},
			[]string{"Camera"}, os.ErrPermission.Error()},
		{"Without a filter, tix isn't asked", "", camera, nil, []string{"+ src/new/orbit.ts"}, []string{"Camera"},
			""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			model := featureModel(t)
			model.lookUpFeature = func(feature string) (*ticket.Ticket, error) {
				if feature != tt.feature {
					t.Errorf("looked up %q, want %q", feature, tt.feature)
				}
				return tt.found, tt.err
			}
			if tt.feature != "" {
				model, _ = press(model, "f", "c", "a", "m", "enter")
			}

			model, command := press(model, "v")
			if (command == nil) != (tt.feature == "") {
				t.Fatalf("command = %v, want one only with a feature filter", command)
			}
			if command != nil {
				updated, _ := model.Update(command())
				model = updated.(Model)
			}

			text := strings.Join(model.plan.lines, "\n")
			for _, want := range tt.contains {
				if !strings.Contains(text, want) {
					t.Errorf("plan lacks %q:\n%s", want, text)
				}
			}
			for _, unwanted := range tt.excludes {
				if strings.Contains(text, unwanted) {
					t.Errorf("plan has %q:\n%s", unwanted, text)
				}
			}
			if model.status != tt.status {
				t.Errorf("status = %q, want %q", model.status, tt.status)
			}
			if line := model.plan.lines[model.plan.scroll]; line != "~ src/app.ts" && tt.feature != "" {
				t.Errorf("first line = %q, want the selected note's heading", line)
			}
		})
	}
}
