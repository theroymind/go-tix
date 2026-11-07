package plan

import (
	"slices"
	"strings"
	"testing"

	"github.com/theroymind/go-tix/contracts/sketch"
	"github.com/theroymind/go-tix/contracts/ticket"
)

// note returns a note planning target that links to each of links.
func note(target string, links ...string) sketch.Note {
	value := sketch.Note{Path: target + sketch.NoteSuffix, Target: target}
	for _, link := range links {
		value.Edges = append(value.Edges, sketch.Edge{To: link})
	}
	return value
}

func TestOrder(t *testing.T) {
	tests := []struct {
		name     string
		notes    []sketch.Note
		expected []string
	}{
		{"No notes", nil, []string{}},
		{"Unlinked notes go in path order", []sketch.Note{note("c"), note("a"), note("b")}, []string{"a", "b", "c"}},
		{"A chain goes dependencies first", []sketch.Note{note("a", "b"), note("b", "c"), note("c")},
			[]string{"c", "b", "a"}},
		{"A diamond puts the shared note first and the joining note last",
			[]sketch.Note{note("a", "b", "c"), note("b", "d"), note("c", "d"), note("d")}, []string{"d", "b", "c", "a"}},
		{"A cycle goes together in path order", []sketch.Note{note("b", "a"), note("a", "b")}, []string{"a", "b"}},
		{"A cycle waits for what it links to, and its dependents wait for it",
			[]sketch.Note{note("a", "c"), note("c", "b"), note("b", "c", "d"), note("d")},
			[]string{"d", "b", "c", "a"}},
		{"A cycle takes its place in path order among ready notes",
			[]sketch.Note{note("a"), note("b", "c"), note("c", "b"), note("z")}, []string{"a", "b", "c", "z"}},
		{"Links to files no note plans, and to itself, don't count",
			[]sketch.Note{note("a", "zz.go", "a"), note("b", "a")}, []string{"a", "b"}},
		{"A ready note goes before a later one even when that one has no links",
			[]sketch.Note{note("a", "y"), note("y"), note("z")}, []string{"y", "a", "z"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			actual := []string{}
			for _, value := range Order(tt.notes) {
				actual = append(actual, value.Target)
			}

			if !slices.Equal(actual, tt.expected) {
				t.Errorf("Order() = %q, want %q", actual, tt.expected)
			}
		})
	}
}

func TestEscape(t *testing.T) {
	tests := []struct {
		text     string
		expected string
	}{
		{"camera.ts", "camera.ts"},
		{`say "hi"`, "say #quot;hi#quot;"},
		{"a|b", "a#124;b"},
		{"list[0]", "list#91;0#93;"},
		{"#fn", "#35;fn"},
		{"#quot;", "#35;quot;"},
		{"f(x) {y}", "f(x) #123;y#125;"},
		{"<b>", "#lt;b#gt;"},
		{"`code`", "#96;code#96;"},
		{"überKamera → ✓", "überKamera → ✓"},
	}
	for _, tt := range tests {
		t.Run(tt.text, func(t *testing.T) {
			if actual := escape(tt.text); actual != tt.expected {
				t.Errorf("escape(%q) = %q, want %q", tt.text, actual, tt.expected)
			}
		})
	}
}

func TestMermaid(t *testing.T) {
	orbit := note("src/useOrbit.ts", "src/camera.ts", "gone.ts", "lib/other.ts")
	orbit.Mark = sketch.MarkNew
	orbit.Edges[0] = sketch.Edge{To: "src/camera.ts", Label: `reads "target" | yaw`, Mark: sketch.MarkChanged}
	orbit.Edges[2].Mark = sketch.MarkChanged
	camera := note("src/camera.ts")
	camera.Mark = sketch.MarkChanged
	readme := note("README.md", "src/useOrbit.ts")
	readme.Mark = sketch.MarkChanged
	readme.Edges[0].Mark = sketch.MarkNew

	expected := `flowchart LR
  subgraph f0["repo root"]
    n0["~README.md"]
  end
  subgraph f1["src"]
    n1["~camera.ts"]
    n2["+useOrbit.ts"]
  end
  n3["gone.ts (dangling)"]
  n4["lib/other.ts"]
  n0 --> n2
  n2 -->|"reads #quot;target#quot; #124; yaw"| n1
  n2 --> n3
  n2 --> n4
` + classDefs + `  class n2 new
  class n0,n1 changed
  class n3 dangling
`

	actual := Mermaid([]sketch.Note{orbit, camera, readme})

	if actual != expected {
		t.Errorf("Mermaid() =\n%s\nwant\n%s", actual, expected)
	}
}

func TestMarkdown(t *testing.T) {
	orbit := note("src/useOrbit.ts", "src/camera.ts")
	orbit.Mark = sketch.MarkNew
	orbit.Body = "\nOrbit the camera.\n\n#fn useOrbit()\n-> src/camera.ts\n? how fast\n> 1 rad/s\n> slower zoomed\n? who\n"
	orbit.Questions = []sketch.Question{{Text: "how fast", Answer: "1 rad/s\nslower zoomed", Line: 9},
		{Text: "who", Line: 11}}
	camera := note("src/camera.ts")
	camera.Mark = sketch.MarkChanged
	camera.Body = "Add a target.\n"
	feature := &ticket.Ticket{ID: "camera", Title: "Fight camera", Description: "Frames the fight.",
		Acceptance: []ticket.Criterion{{Text: "It orbits.", Done: true}, {Text: "It eases."}}}
	notes := []sketch.Note{orbit, camera}
	graph := "\n## Graph\n\n```mermaid\n" + Mermaid(notes) + "```\n"
	buildOrder := "\n## Build order\n\n### ~src/camera.ts\n\nAdd a target.\n\n### +src/useOrbit.ts\n\n" +
		"Orbit the camera.\n\n#fn useOrbit()\n-> src/camera.ts\n? how fast\n> 1 rad/s\n> slower zoomed\n? who\n"
	decisions := "\n## Decisions\n\n- src/useOrbit.ts.sketch.md: how fast\n  > 1 rad/s\n  > slower zoomed\n"
	open := "\n## Open questions\n\n- src/useOrbit.ts.sketch.md:11: who\n"

	tests := []struct {
		name       string
		feature    string
		tixFeature *ticket.Ticket
		notes      []sketch.Note
		expected   string
	}{
		{"With tix: the feature, graph, notes in build order, decisions and open questions", "camera", feature,
			notes, "# Plan: camera\n\n## Fight camera\n\nFrames the fight.\n\n### Acceptance criteria\n\n" +
				"1. [x] It orbits.\n2. [ ] It eases.\n" + graph + buildOrder + decisions + open},
		{"Without tix, no feature section", "camera", nil, notes, "# Plan: camera\n" + graph + buildOrder +
			decisions + open},
		{"Without a feature, a plain title", "", nil, notes, "# Plan\n" + graph + buildOrder + decisions + open},
		{"No questions, no question sections", "", nil, []sketch.Note{camera}, "# Plan\n\n## Graph\n\n```mermaid\n" +
			Mermaid([]sketch.Note{camera}) + "```\n\n## Build order\n\n### ~src/camera.ts\n\nAdd a target.\n"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			actual := Markdown(tt.feature, tt.tixFeature, tt.notes)

			if actual != tt.expected {
				t.Errorf("Markdown() =\n%s\nwant\n%s", actual, tt.expected)
			}
			if strings.Contains(actual, "\n\n\n") {
				t.Errorf("Markdown() has a run of blank lines:\n%s", actual)
			}
		})
	}
}
