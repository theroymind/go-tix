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
		{"Free text only", "Orbit the camera around the fight.\n\n- smooth it\n", sketch.Note{}, nil},
		{"Front matter", "---\nfeature: sketch-explorer\ntask: read-notes\n---\nbody\n",
			sketch.Note{Feature: "sketch-explorer", Task: "read-notes"}, nil},
		{"Front matter with keys it doesn't know", "---\nfeature: sketch-explorer\ntags: [a, b]\n---\n",
			sketch.Note{Feature: "sketch-explorer"}, nil},
		{"Empty front matter", "---\n---\n#fn a\n", sketch.Note{Items: []sketch.Item{{Type: sketch.ItemTypeFn, Name: "a"}}},
			nil},
		{"Front matter with no closing line", "---\nfeature: sketch-explorer\n#fn a\n", sketch.Note{}, ErrFrontMatter},
		{"Front matter that isn't YAML", "---\nfeature: [a\n---\n", sketch.Note{}, ErrFrontMatter},
		{"Front matter that isn't a map", "---\njust words\n---\n", sketch.Note{}, ErrFrontMatter},
		{"A --- lower down is free text", "intro\n---\nfeature: x\n---\n", sketch.Note{}, nil},
		{"Every item type", "#fn a\n#type b\n#component c\n#hook d\n#flow e\n#tune f\n#rule g\n#test h\n",
			sketch.Note{Items: []sketch.Item{
				{Type: sketch.ItemTypeFn, Name: "a"}, {Type: sketch.ItemTypeType, Name: "b"},
				{Type: sketch.ItemTypeComponent, Name: "c"}, {Type: sketch.ItemTypeHook, Name: "d"},
				{Type: sketch.ItemTypeFlow, Name: "e"}, {Type: sketch.ItemTypeTune, Name: "f"},
				{Type: sketch.ItemTypeRule, Name: "g"}, {Type: sketch.ItemTypeTest, Name: "h"},
			}}, nil},
		{"An item's rest of line", "#fn useOrbit   target Ref<Vec3>  \r\n",
			sketch.Note{Items: []sketch.Item{{Type: sketch.ItemTypeFn, Name: "useOrbit", Text: "target Ref<Vec3>"}}}, nil},
		{"An unknown #word is a tag", "#todo later\n#Fn a\n", sketch.Note{}, nil},
		{"A heading is free text", "# fn a\n## Plan\n", sketch.Note{}, nil},
		{"An item type with no name is free text", "#fn\n#fn   \n", sketch.Note{}, nil},
		{"An indented marker is free text", "  #fn a\n  ? why\n", sketch.Note{}, nil},
		{"A question without an answer", "intro\n? who owns the camera\n\n> too far down\n",
			sketch.Note{Questions: []sketch.Question{{Text: "who owns the camera", Line: 2}}}, nil},
		{"A question with an answer", "? how fast\n> 1 rad/s\n> more\n",
			sketch.Note{Questions: []sketch.Question{{Text: "how fast", Answer: "1 rad/s", Line: 1}}}, nil},
		{"An empty answer leaves it open", "? how fast\n>\n",
			sketch.Note{Questions: []sketch.Question{{Text: "how fast", Line: 1}}}, nil},
		{"A stray > is free text", "> a quote\n", sketch.Note{}, nil},
		{"A ? with no text is free text", "?\n? \n?no space\n", sketch.Note{}, nil},
		{"Line numbers count the front matter", "---\ntask: t\n---\n#fn a\n? why\n",
			sketch.Note{Task: "t", Items: []sketch.Item{{Type: sketch.ItemTypeFn, Name: "a"}},
				Questions: []sketch.Question{{Text: "why", Line: 5}}}, nil},
		{"An item marker under a question isn't its answer", "? why\n#fn a\n",
			sketch.Note{Items: []sketch.Item{{Type: sketch.ItemTypeFn, Name: "a"}},
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
			expected.SchemaVersion, expected.Path, expected.Target = 1, "src/useOrbit.ts.sketch.md", "src/useOrbit.ts"
			if !reflect.DeepEqual(actual, expected) {
				t.Errorf("Parse() = %+v, want %+v", actual, expected)
			}
			if err := actual.Validate(); err != nil {
				t.Errorf("Parse() gave a note that doesn't validate: %v", err)
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

func TestSetFeature(t *testing.T) {
	tests := []struct {
		name     string
		text     string
		expected string
		err      error
	}{
		{"No front matter", "# app.ts\n", "---\nfeature: camera\n---\n# app.ts\n", nil},
		{"An empty note", "", "---\nfeature: camera\n---\n", nil},
		{"Front matter without a feature", "---\ntask: t\n---\nbody\n", "---\nfeature: camera\ntask: t\n---\nbody\n", nil},
		{"Front matter with a feature", "---\ntask: t\nfeature: orbit\n---\n", "---\ntask: t\nfeature: camera\n---\n", nil},
		{"Empty front matter", "---\n---\nbody\n", "---\nfeature: camera\n---\nbody\n", nil},
		{"A feature line below the front matter stays", "---\n---\nfeature: orbit\n",
			"---\nfeature: camera\n---\nfeature: orbit\n", nil},
		{"Front matter with no closing line", "---\ntask: t\n", "", ErrFrontMatter},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			actual, err := SetFeature(tt.text, "camera")

			if !errors.Is(err, tt.err) {
				t.Fatalf("SetFeature() error = %v, want %v", err, tt.err)
			}
			if actual != tt.expected {
				t.Errorf("SetFeature() = %q, want %q", actual, tt.expected)
			}
			if tt.err == nil {
				if parsed, err := Parse("a.ts.sketch.md", actual); err != nil || parsed.Feature != "camera" {
					t.Errorf("Parse(SetFeature()) = %+v, %v, want feature camera", parsed, err)
				}
			}
		})
	}
}
