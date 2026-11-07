// Package tree builds the repo tree the sketch explorer shows: the repo's files with each note folded into the
// node of the file it plans, a ghost node when that file doesn't exist yet, and flattens it into display rows.
package tree

import (
	"bytes"
	"cmp"
	"io/fs"
	"os/exec"
	"path"
	"path/filepath"
	"slices"
	"strings"

	"github.com/theroymind/go-tix/contracts/sketch"
	"github.com/theroymind/go-tix/internal/sketch/questions"
)

// Node is a file or folder in the tree. A ghost is a file a note plans that doesn't exist yet, or a folder that
// exists only to hold one.
type Node struct {
	Path          string // slash-separated from the repo root; the root itself is "."
	Name          string
	IsDir         bool
	Ghost         bool
	Note          *sketch.Note // the note that plans this node, if any
	Problem       error        // why that note didn't read, when it didn't
	OpenQuestions int          // in this node's own note
	NotesBelow    int          // notes on the nodes below this folder
	OpenBelow     int          // open questions in those notes
	Children      []*Node
}

// Row is one visible line of the tree: a node and how deep it sits under the root.
type Row struct {
	Node  *Node
	Depth int
}

// Files returns the slash-separated path from root of every file under it. Inside a git work tree it asks git for
// the tracked and untracked files that aren't ignored, so folders like node_modules and dist stay hidden, leaving
// out tracked files deleted from the work tree; outside one it walks root, skipping hidden folders and
// node_modules.
func Files(root string) ([]string, error) {
	files, err := gitFiles(root, "--cached", "--others", "--exclude-standard")
	if err != nil {
		return walk(root)
	}
	deleted, err := gitFiles(root, "--deleted")
	if err != nil {
		return nil, err
	}
	return slices.DeleteFunc(files, func(file string) bool { return slices.Contains(deleted, file) }), nil
}

// gitFiles returns the paths git ls-files prints with options, run in root.
func gitFiles(root string, options ...string) ([]string, error) {
	command := exec.Command("git", append([]string{"ls-files", "-z"}, options...)...)
	command.Dir = root
	output, err := command.Output()
	if err != nil {
		return nil, err
	}
	var files []string
	for _, file := range bytes.Split(output, []byte{0}) {
		if len(file) > 0 {
			files = append(files, string(file))
		}
	}
	return files, nil
}

// walk lists the files under root the way Files does outside git.
func walk(root string) ([]string, error) {
	var files []string
	err := filepath.WalkDir(root, func(filePath string, entry fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		name := entry.Name()
		if entry.IsDir() {
			if filePath != root && (strings.HasPrefix(name, hiddenPrefix) || name == nodeModules) {
				return filepath.SkipDir
			}
			return nil
		}
		relative, err := filepath.Rel(root, filePath)
		if err != nil {
			return err
		}
		files = append(files, filepath.ToSlash(relative))
		return nil
	})
	return files, err
}

// Build makes the tree of files and folds each note into the node of its target. A note never gets a row of its
// own: a target that isn't among files becomes a ghost node, under ghost folders for any folders that are missing
// too. problems holds, by note path, why a note didn't read; that note's node carries it. Folders sort before
// files, each by name.
func Build(files []string, notes []sketch.Note, problems map[string]error) *Node {
	root := &Node{Path: rootPath, IsDir: true}
	nodes := map[string]*Node{rootPath: root}
	for _, file := range files {
		if !strings.HasSuffix(file, sketch.NoteSuffix) {
			add(nodes, file, false, false)
		}
	}
	for index := range notes {
		value := &notes[index]
		node := add(nodes, value.Target, false, true)
		node.Note = value
		node.Problem = problems[value.Path]
		node.OpenQuestions = len(questions.Open([]sketch.Note{*value}))
		for folder := path.Dir(value.Target); ; folder = path.Dir(folder) {
			nodes[folder].NotesBelow++
			nodes[folder].OpenBelow += node.OpenQuestions
			if folder == rootPath {
				break
			}
		}
	}
	sortChildren(root)
	return root
}

// add returns the node at nodePath, first making it and any missing folders above it, each a ghost when ghost is
// set.
func add(nodes map[string]*Node, nodePath string, isDir, ghost bool) *Node {
	if node, exists := nodes[nodePath]; exists {
		return node
	}
	parent := add(nodes, path.Dir(nodePath), true, ghost)
	node := &Node{Path: nodePath, Name: path.Base(nodePath), IsDir: isDir, Ghost: ghost}
	parent.Children = append(parent.Children, node)
	nodes[nodePath] = node
	return node
}

func sortChildren(node *Node) {
	slices.SortFunc(node.Children, func(a, b *Node) int {
		if a.IsDir != b.IsDir {
			if a.IsDir {
				return -1
			}
			return 1
		}
		return cmp.Or(cmp.Compare(strings.ToLower(a.Name), strings.ToLower(b.Name)), cmp.Compare(a.Name, b.Name))
	})
	for _, child := range node.Children {
		sortChildren(child)
	}
}

// NoteFolders returns the path of every folder with a note somewhere below it, the folders to expand so that
// every note shows.
func NoteFolders(root *Node) map[string]bool {
	folders := map[string]bool{}
	var visit func(node *Node)
	visit = func(node *Node) {
		if node.NotesBelow == 0 {
			return
		}
		folders[node.Path] = true
		for _, child := range node.Children {
			visit(child)
		}
	}
	visit(root)
	return folders
}

// Flatten returns the rows under root that show when the folders in expanded are open, the root's children at
// depth 0. With notesOnly, only nodes with a note and the folders above them show.
func Flatten(root *Node, expanded map[string]bool, notesOnly bool) []Row {
	var rows []Row
	var visit func(node *Node, depth int)
	visit = func(node *Node, depth int) {
		for _, child := range node.Children {
			if notesOnly && child.Note == nil && child.NotesBelow == 0 {
				continue
			}
			rows = append(rows, Row{Node: child, Depth: depth})
			if child.IsDir && expanded[child.Path] {
				visit(child, depth+1)
			}
		}
	}
	visit(root, 0)
	return rows
}
