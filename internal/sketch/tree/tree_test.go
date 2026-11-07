package tree

import (
	"fmt"
	"maps"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/theroymind/go-tix/contracts/sketch"
)

// writeFiles writes empty files at the slash-separated paths under root.
func writeFiles(t *testing.T, root string, files []string) {
	t.Helper()
	for _, file := range files {
		filePath := filepath.Join(root, filepath.FromSlash(file))
		if err := os.MkdirAll(filepath.Dir(filePath), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filePath, nil, 0o644); err != nil {
			t.Fatal(err)
		}
	}
}

func TestFiles(t *testing.T) {
	tests := []struct {
		name     string
		git      bool
		files    []string
		expected []string
	}{
		{"Git hides ignored files", true,
			[]string{".gitignore", "dist/app.js", "node_modules/p/i.js", "src/main.ts", "src/odd name.ts"},
			[]string{".gitignore", "src/main.ts", "src/odd name.ts"}},
		{"Outside git, hidden folders and node_modules are skipped", false,
			[]string{".cache/x", "node_modules/p/i.js", "web/node_modules/q.js", "dist/app.js", "src/main.ts"},
			[]string{"dist/app.js", "src/main.ts"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			root := t.TempDir()
			writeFiles(t, root, tt.files)
			if tt.git {
				if err := os.WriteFile(filepath.Join(root, ".gitignore"), []byte("dist/\nnode_modules/\n"), 0o644); err != nil {
					t.Fatal(err)
				}
				if output, err := exec.Command("git", "-C", root, "init", "-q").CombinedOutput(); err != nil {
					t.Fatalf("git init: %v %s", err, output)
				}
			}

			actual, err := Files(root)

			if err != nil {
				t.Fatalf("Files() error = %v", err)
			}
			slices.Sort(actual)
			if !slices.Equal(actual, tt.expected) {
				t.Errorf("Files() = %q, want %q", actual, tt.expected)
			}
		})
	}
}

func note(target string, questions ...sketch.Question) sketch.Note {
	return sketch.Note{SchemaVersion: sketch.SchemaVersion, Path: target + sketch.NoteSuffix, Target: target,
		Questions: questions}
}

// describe writes each node under root as one line: its path, / for a folder, then ghost, its note's target, its
// open question count and the counts below it, where set.
func describe(root *Node) []string {
	var lines []string
	var visit func(node *Node)
	visit = func(node *Node) {
		for _, child := range node.Children {
			line := child.Path
			if child.IsDir {
				line += "/"
			}
			if child.Ghost {
				line += " ghost"
			}
			if child.Note != nil {
				line += " note:" + child.Note.Target
			}
			if child.OpenQuestions > 0 {
				line += fmt.Sprintf(" open:%d", child.OpenQuestions)
			}
			if child.NotesBelow > 0 {
				line += fmt.Sprintf(" below:%d/%d", child.NotesBelow, child.OpenBelow)
			}
			lines = append(lines, line)
			visit(child)
		}
	}
	visit(root)
	return lines
}

func TestBuild(t *testing.T) {
	open := sketch.Question{Text: "why", Line: 1}
	answered := sketch.Question{Text: "how", Answer: "so", Line: 2}
	tests := []struct {
		name     string
		files    []string
		notes    []sketch.Note
		expected []string
	}{
		{"Folders first, then files, by name", []string{"b.go", "A.md", "src/z.ts", "lib/a.ts", "a.go"}, nil,
			[]string{"lib/", "lib/a.ts", "src/", "src/z.ts", "a.go", "A.md", "b.go"}},
		{"A note folds into its existing target", []string{"src/app.ts", "src/app.ts.sketch.md"},
			[]sketch.Note{note("src/app.ts")},
			[]string{"src/ below:1/0", "src/app.ts note:src/app.ts"}},
		{"A note for a missing file is a ghost among the files", []string{"src/b.ts", "src/d.ts", "src/c.ts.sketch.md"},
			[]sketch.Note{note("src/c.ts")},
			[]string{"src/ below:1/0", "src/b.ts", "src/c.ts ghost note:src/c.ts", "src/d.ts"}},
		{"Missing folders above a ghost are ghosts too", []string{"src/a.ts", "src/new/deep/x.ts.sketch.md"},
			[]sketch.Note{note("src/new/deep/x.ts")},
			[]string{"src/ below:1/0", "src/new/ ghost below:1/0", "src/new/deep/ ghost below:1/0",
				"src/new/deep/x.ts ghost note:src/new/deep/x.ts", "src/a.ts"}},
		{"A note can plan a folder", []string{"src/a.ts", "src.sketch.md"}, []sketch.Note{note("src")},
			[]string{"src/ note:src", "src/a.ts"}},
		{"Open questions count up the tree, answered ones don't", []string{"a/b/c.ts", "a/d.ts"},
			[]sketch.Note{note("a/b/c.ts", open, answered, open), note("a/d.ts", open)},
			[]string{"a/ below:2/3", "a/b/ below:1/2", "a/b/c.ts note:a/b/c.ts open:2", "a/d.ts note:a/d.ts open:1"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			actual := describe(Build(tt.files, tt.notes, nil))

			if !slices.Equal(actual, tt.expected) {
				t.Errorf("Build() =\n%s\nwant\n%s", strings.Join(actual, "\n"), strings.Join(tt.expected, "\n"))
			}
		})
	}
}

func TestNoteFolders(t *testing.T) {
	root := Build([]string{"a/b/c.ts", "e/f.ts", "g.ts"}, []sketch.Note{note("a/b/c.ts"), note("g.ts")}, nil)

	actual := slices.Sorted(maps.Keys(NoteFolders(root)))

	if expected := []string{".", "a", "a/b"}; !slices.Equal(actual, expected) {
		t.Errorf("NoteFolders() = %q, want %q", actual, expected)
	}
}

func TestFlatten(t *testing.T) {
	root := Build([]string{"a/b/c.ts", "a/d.ts", "e/f.ts", "g.ts"}, []sketch.Note{note("a/b/c.ts"), note("e/new.ts")}, nil)
	tests := []struct {
		name      string
		expanded  []string
		notesOnly bool
		expected  []string
	}{
		{"All folders closed", nil, false, []string{"0 a", "0 e", "0 g.ts"}},
		{"One folder open", []string{"a"}, false, []string{"0 a", "1 a/b", "1 a/d.ts", "0 e", "0 g.ts"}},
		{"A folder open under a closed one stays hidden", []string{"a/b"}, false, []string{"0 a", "0 e", "0 g.ts"}},
		{"Every folder open", []string{"a", "a/b", "e"}, false,
			[]string{"0 a", "1 a/b", "2 a/b/c.ts", "1 a/d.ts", "0 e", "1 e/f.ts", "1 e/new.ts", "0 g.ts"}},
		{"Notes only keeps notes and the folders above them", []string{"a", "a/b", "e"}, true,
			[]string{"0 a", "1 a/b", "2 a/b/c.ts", "0 e", "1 e/new.ts"}},
		{"Notes only still follows what's open", []string{"a"}, true, []string{"0 a", "1 a/b", "0 e"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			expanded := map[string]bool{}
			for _, folder := range tt.expanded {
				expanded[folder] = true
			}

			var actual []string
			for _, row := range Flatten(root, expanded, tt.notesOnly) {
				actual = append(actual, fmt.Sprintf("%d %s", row.Depth, row.Node.Path))
			}

			if !slices.Equal(actual, tt.expected) {
				t.Errorf("Flatten() = %q, want %q", actual, tt.expected)
			}
		})
	}
}
