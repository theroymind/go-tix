package note

import (
	"errors"
	"reflect"
	"testing"

	"github.com/theroymind/go-tix/contracts/sketch"
)

func TestParse(t *testing.T) {
	tests := []struct {
		name     string
		text     string
		expected sketch.Note
		err      error
	}{
		{"An empty note", "", sketch.Note{}, nil},
		{"Free text is the body", "Orbit the camera around the fight.\n\n- smooth it\n",
			sketch.Note{Body: "Orbit the camera around the fight.\n\n- smooth it\n"}, nil},
		{"The body keeps its spacing and line ends", "fn useOrbit(target)  \r\n\treturn x\n\n",
			sketch.Note{Body: "fn useOrbit(target)  \r\n\treturn x\n\n"}, nil},
		{"Front matter", "---\nfeature: sketch-explorer\nstatus: ready\n---\nbody\n",
			sketch.Note{Feature: "sketch-explorer", Status: sketch.StatusReady, Body: "body\n"}, nil},
		{"Front matter with keys it doesn't know, task among them",
			"---\nfeature: sketch-explorer\ntask: read-notes\ntags: [a, b]\n---\n",
			sketch.Note{Feature: "sketch-explorer"}, nil},
		{"Empty front matter", "---\n---\nhi", sketch.Note{Body: "hi"}, nil},
		{"Front matter with no closing line", "---\nfeature: sketch-explorer\n", sketch.Note{}, ErrFrontMatter},
		{"Front matter that isn't YAML", "---\nfeature: [a\n---\n", sketch.Note{}, ErrFrontMatter},
		{"Front matter that isn't a map", "---\njust words\n---\n", sketch.Note{}, ErrFrontMatter},
		{"A --- lower down is body", "intro\n---\nfeature: x\n---\n",
			sketch.Note{Body: "intro\n---\nfeature: x\n---\n"}, nil},
		{"Markers are body and nothing more", "#fn useOrbit target\n#type Orbit\n",
			sketch.Note{Body: "#fn useOrbit target\n#type Orbit\n"}, nil},
		{"An edge from the repo root", "-> src/camera.ts\n",
			sketch.Note{Body: "-> src/camera.ts\n", Edges: []sketch.Edge{{To: "src/camera.ts"}}}, nil},
		{"An edge with a label", "-> src/camera.ts : reads the target \n",
			sketch.Note{Body: "-> src/camera.ts : reads the target \n",
				Edges: []sketch.Edge{{To: "src/camera.ts", Label: "reads the target"}}}, nil},
		{"An edge from the note's folder", "->./camera.ts:reads",
			sketch.Note{Body: "->./camera.ts:reads", Edges: []sketch.Edge{{To: "src/camera.ts", Label: "reads"}}}, nil},
		{"An edge from the parent folder", "-> ../web/app.tsx",
			sketch.Note{Body: "-> ../web/app.tsx", Edges: []sketch.Edge{{To: "web/app.tsx"}}}, nil},
		{"An edge is stored clean", "-> lib//x/../y.ts",
			sketch.Note{Body: "-> lib//x/../y.ts", Edges: []sketch.Edge{{To: "lib/y.ts"}}}, nil},
		{"An edge escaping the repo is free text", "-> ../../x.ts\n-> ../x.ts\n",
			sketch.Note{Body: "-> ../../x.ts\n-> ../x.ts\n", Edges: []sketch.Edge{{To: "x.ts"}}}, nil},
		{"An edge with no path, or an absolute one, is free text", "->\n-> : label\n-> /etc/x\n-> .\n",
			sketch.Note{Body: "->\n-> : label\n-> /etc/x\n-> .\n"}, nil},
		{"An indented marker is free text", "  -> a.ts\n  ? why\n", sketch.Note{Body: "  -> a.ts\n  ? why\n"}, nil},
		{"A question without an answer", "intro\n? who owns the camera\n\n\n> too far down\n",
			sketch.Note{Body: "intro\n? who owns the camera\n\n\n> too far down\n",
				Questions: []sketch.Question{{Text: "who owns the camera", Line: 2}}}, nil},
		{"A question with an answer", "? how fast\n> 1 rad/s\n",
			sketch.Note{Body: "? how fast\n> 1 rad/s\n",
				Questions: []sketch.Question{{Text: "how fast", Answer: "1 rad/s", Line: 1}}}, nil},
		{"An answer over several lines", "? how fast\n> 1 rad/s\n>   then ease out  \nlater\n",
			sketch.Note{Body: "? how fast\n> 1 rad/s\n>   then ease out  \nlater\n",
				Questions: []sketch.Question{{Text: "how fast", Answer: "1 rad/s\nthen ease out", Line: 1}}}, nil},
		{"An answer after one blank line", "? how fast\n\n> 1 rad/s\n",
			sketch.Note{Body: "? how fast\n\n> 1 rad/s\n",
				Questions: []sketch.Question{{Text: "how fast", Answer: "1 rad/s", Line: 1}}}, nil},
		{"An empty answer leaves it open", "? how fast\n>\n",
			sketch.Note{Body: "? how fast\n>\n", Questions: []sketch.Question{{Text: "how fast", Line: 1}}}, nil},
		{"A question under a blank line after a question", "? a\n\n? b\n",
			sketch.Note{Body: "? a\n\n? b\n",
				Questions: []sketch.Question{{Text: "a", Line: 1}, {Text: "b", Line: 3}}}, nil},
		{"A stray > is free text", "> a quote\n", sketch.Note{Body: "> a quote\n"}, nil},
		{"A ? with no text is free text", "?\n? \n?no space\n", sketch.Note{Body: "?\n? \n?no space\n"}, nil},
		{"Line numbers count the front matter", "---\nstatus: draft\n---\n-> a.ts\n? why\n",
			sketch.Note{Status: sketch.StatusDraft, Body: "-> a.ts\n? why\n", Edges: []sketch.Edge{{To: "a.ts"}},
				Questions: []sketch.Question{{Text: "why", Line: 5}}}, nil},
		{"An edge under a question isn't its answer", "? why\n-> a.ts\n",
			sketch.Note{Body: "? why\n-> a.ts\n", Edges: []sketch.Edge{{To: "a.ts"}},
				Questions: []sketch.Question{{Text: "why", Line: 1}}}, nil},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			actual, err := Parse("src/useOrbit.ts.sketch.md", tt.text)

			if !errors.Is(err, tt.err) {
				t.Fatalf("Parse() error = %v, want %v", err, tt.err)
			}
			if tt.err != nil {
				return
			}
			expected := tt.expected
			expected.SchemaVersion, expected.Path, expected.Target = sketch.SchemaVersion, "src/useOrbit.ts.sketch.md", "src/useOrbit.ts"
			if !reflect.DeepEqual(actual, expected) {
				t.Errorf("Parse() = %+v, want %+v", actual, expected)
			}
			if err := actual.Validate(); err != nil {
				t.Errorf("Parse() gave a note that doesn't validate: %v", err)
			}
		})
	}
}

// TestParseStatus checks that Parse keeps any status as written and leaves rejecting an unknown one to Validate.
func TestParseStatus(t *testing.T) {
	tests := []struct {
		name     string
		status   string
		expected sketch.Status
		valid    bool
	}{
		{"Draft", "draft", sketch.StatusDraft, true},
		{"Ready, padded", "  ready ", sketch.StatusReady, true},
		{"Unset", "", "", true},
		{"Unknown", "done", "done", false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			actual, err := Parse("a.ts.sketch.md", "---\nstatus: "+tt.status+"\n---\n")
			if err != nil {
				t.Fatalf("Parse() error = %v", err)
			}

			if actual.Status != tt.expected {
				t.Errorf("Parse() status = %q, want %q", actual.Status, tt.expected)
			}
			if valid := actual.Validate() == nil; valid != tt.valid {
				t.Errorf("Validate() passes = %v, want %v", valid, tt.valid)
			}
		})
	}
}

func TestTarget(t *testing.T) {
	tests := []struct {
		name     string
		notePath string
		expected string
		err      error
	}{
		{"A note at the root", "useOrbit.ts.sketch.md", "useOrbit.ts", nil},
		{"A note in nested folders", "src/composables/useOrbit.ts.sketch.md", "src/composables/useOrbit.ts", nil},
		{"A name with several dots", "a/a.test.ts.sketch.md", "a/a.test.ts", nil},
		{"A folder name with dots", "v1.sketch.md/b.go.sketch.md", "v1.sketch.md/b.go", nil},
		{"A file that isn't a note", "src/useOrbit.ts.md", "", ErrNotANote},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			actual, err := Target(tt.notePath)

			if !errors.Is(err, tt.err) {
				t.Fatalf("Target() error = %v, want %v", err, tt.err)
			}
			if actual != tt.expected {
				t.Errorf("Target() = %q, want %q", actual, tt.expected)
			}
		})
	}
}

func TestSetField(t *testing.T) {
	tests := []struct {
		name     string
		key      string
		value    string
		text     string
		expected string
		err      error
	}{
		{"No front matter", FeatureKey, "camera", "# app.ts\n", "---\nfeature: camera\n---\n# app.ts\n", nil},
		{"An empty note", FeatureKey, "camera", "", "---\nfeature: camera\n---\n", nil},
		{"Front matter without a feature", FeatureKey, "camera", "---\ntask: t\n---\nbody\n",
			"---\nfeature: camera\ntask: t\n---\nbody\n", nil},
		{"Front matter with a feature", FeatureKey, "camera", "---\ntask: t\nfeature: orbit\n---\n",
			"---\ntask: t\nfeature: camera\n---\n", nil},
		{"Empty front matter", FeatureKey, "camera", "---\n---\nbody\n", "---\nfeature: camera\n---\nbody\n", nil},
		{"A feature line below the front matter stays", FeatureKey, "camera", "---\n---\nfeature: orbit\n",
			"---\nfeature: camera\n---\nfeature: orbit\n", nil},
		{"Front matter with no closing line", FeatureKey, "camera", "---\ntask: t\n", "", ErrFrontMatter},
		{"A status on a note with none", StatusKey, "ready", "body\n", "---\nstatus: ready\n---\nbody\n", nil},
		{"A status added beside the feature", StatusKey, "ready", "---\nfeature: camera\n---\n",
			"---\nstatus: ready\nfeature: camera\n---\n", nil},
		{"A status replaced, leaving the feature", StatusKey, "draft", "---\nfeature: camera\nstatus: ready\n---\n",
			"---\nfeature: camera\nstatus: draft\n---\n", nil},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			actual, err := SetField(tt.text, tt.key, tt.value)

			if !errors.Is(err, tt.err) {
				t.Fatalf("SetField() error = %v, want %v", err, tt.err)
			}
			if actual != tt.expected {
				t.Errorf("SetField() = %q, want %q", actual, tt.expected)
			}
			if tt.err != nil {
				return
			}
			parsed, err := Parse("a.ts.sketch.md", actual)
			if err != nil {
				t.Fatalf("Parse(SetField()) error = %v", err)
			}
			if read := map[string]string{FeatureKey: parsed.Feature, StatusKey: string(parsed.Status)}; read[tt.key] != tt.value {
				t.Errorf("Parse(SetField()) = %+v, want %s %s", parsed, tt.key, tt.value)
			}
		})
	}
}
