package repo

import (
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"testing"

	"github.com/theroymind/go-tix/internal/sketch/tree"
)

// gitRepo writes files under a new git repo, ignoring dist/, and stages them all.
func gitRepo(t *testing.T, files map[string]string) string {
	t.Helper()
	root := t.TempDir()
	files[".gitignore"] = "dist/\n"
	for file, text := range files {
		filePath := filepath.Join(root, filepath.FromSlash(file))
		if err := os.MkdirAll(filepath.Dir(filePath), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filePath, []byte(text), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	for _, arguments := range [][]string{{"init", "-q"}, {"add", "."}} {
		if output, err := exec.Command("git", append([]string{"-C", root}, arguments...)...).CombinedOutput(); err != nil {
			t.Fatalf("git %v: %v %s", arguments, err, output)
		}
	}
	return root
}

// find returns the node at nodePath under root, or nil.
func find(root *tree.Node, nodePath string) *tree.Node {
	if root.Path == nodePath {
		return root
	}
	for _, child := range root.Children {
		if node := find(child, nodePath); node != nil {
			return node
		}
	}
	return nil
}

func TestLoad(t *testing.T) {
	tests := []struct {
		name     string
		files    map[string]string
		remove   string // a tracked file deleted after staging
		notes    []string
		problems int
		node     string // a node to check in the tree
		ghost    bool
		problem  bool
	}{
		{"A note in a hidden tracked folder shows",
			map[string]string{".github/workflows/ci.yml": "on: push\n", ".github/workflows/ci.yml.sketch.md": "ci\n"},
			"", []string{".github/workflows/ci.yml.sketch.md"}, 0, ".github/workflows/ci.yml", false, false},
		{"A note in an ignored folder shows nowhere",
			map[string]string{"dist/app.js": "x\n", "dist/app.js.sketch.md": "app\n", "src/a.ts": "a\n"},
			"", nil, 0, "", false, false},
		{"A deleted tracked file is a ghost, and its note new",
			map[string]string{"src/a.ts": "a\n", "src/a.ts.sketch.md": "a note\n"},
			"src/a.ts", []string{"src/a.ts.sketch.md"}, 0, "src/a.ts", true, false},
		{"A note that doesn't read is a problem, not a failure, and still shows",
			map[string]string{"src/a.ts": "a\n", "src/a.ts.sketch.md": "---\nfeature: [\n", "src/b.ts.sketch.md": "b\n"},
			"", []string{"src/b.ts.sketch.md"}, 1, "src/a.ts", false, true},
		{"A link to a path under a file is dangling, not a problem",
			map[string]string{"src/app.ts": "a\n", "src/b.ts.sketch.md": "-> src/app.ts/helper.ts\n"},
			"", []string{"src/b.ts.sketch.md"}, 0, "src/b.ts", true, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			root := gitRepo(t, tt.files)
			if tt.remove != "" {
				if err := os.Remove(filepath.Join(root, filepath.FromSlash(tt.remove))); err != nil {
					t.Fatal(err)
				}
			}

			loaded, err := Load(root)

			if err != nil {
				t.Fatalf("Load() error = %v", err)
			}
			var notes []string
			for _, value := range loaded.Notes {
				notes = append(notes, value.Path)
			}
			if !slices.Equal(notes, tt.notes) {
				t.Errorf("Notes = %q, want %q", notes, tt.notes)
			}
			if len(loaded.Problems) != tt.problems {
				t.Errorf("Problems = %v, want %d", loaded.Problems, tt.problems)
			}
			if loaded.Tree.NotesBelow != len(tt.notes)+tt.problems {
				t.Errorf("Tree.NotesBelow = %d, want %d", loaded.Tree.NotesBelow, len(tt.notes)+tt.problems)
			}
			if tt.node == "" {
				return
			}
			node := find(loaded.Tree, tt.node)
			if node == nil || node.Note == nil {
				t.Fatalf("node %s with a note not in the tree", tt.node)
			}
			if node.Ghost != tt.ghost {
				t.Errorf("Ghost = %v, want %v", node.Ghost, tt.ghost)
			}
			if (node.Problem != nil) != tt.problem {
				t.Errorf("Problem = %v, want one: %v", node.Problem, tt.problem)
			}
		})
	}
}
