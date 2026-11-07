// Package repo loads a repo's sketch notes and the tree the explorer shows from one listing of its files.
package repo

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"slices"
	"strings"

	"github.com/theroymind/go-tix/contracts/sketch"
	"github.com/theroymind/go-tix/internal/sketch/note"
	"github.com/theroymind/go-tix/internal/sketch/tree"
)

var ErrTargetFolder = errors.New("plans a folder, not a file")

// Repo is a repo's notes and tree as they are on disk.
type Repo struct {
	Tree     *tree.Node
	Notes    []sketch.Note // the notes that read and validate, marked against the repo, in path order
	Problems []error       // one for each note that didn't, naming it
}

// Load lists the files under root once, reads and validates every note among them, marks each and its edges
// against the repo, and builds the tree. A note that doesn't read joins Problems and still shows in the tree, with its problem, so it
// can be opened and fixed. Only failing to list the files fails Load.
func Load(root string) (Repo, error) {
	files, err := tree.Files(root)
	if err != nil {
		return Repo{}, err
	}
	planned := map[string]bool{}
	for _, file := range files {
		if target, isNote := strings.CutSuffix(file, sketch.NoteSuffix); isNote {
			planned[target] = true
		}
	}
	repo := Repo{Notes: []sketch.Note{}}
	var unread []sketch.Note
	problems := map[string]error{}
	for _, file := range files {
		if !strings.HasSuffix(file, sketch.NoteSuffix) {
			continue
		}
		value, err := loadNote(root, file, planned)
		if err != nil {
			repo.Problems = append(repo.Problems, err)
			problems[file] = err
			unread = append(unread, sketch.Note{Path: file, Target: strings.TrimSuffix(file, sketch.NoteSuffix)})
			continue
		}
		repo.Notes = append(repo.Notes, value)
	}
	repo.Tree = tree.Build(files, slices.Concat(repo.Notes, unread), problems)
	return repo, nil
}

// loadNote loads the note at notePath, checks it against the contract, and marks it against the repo, where
// planned holds the target of every note in it.
func loadNote(root, notePath string, planned map[string]bool) (sketch.Note, error) {
	value, err := note.Load(root, notePath)
	if err != nil {
		return sketch.Note{}, err
	}
	if err := value.Validate(); err != nil {
		return sketch.Note{}, fmt.Errorf("%s: %w", notePath, err)
	}
	if err := mark(root, &value, planned); err != nil {
		return sketch.Note{}, fmt.Errorf("%s: %w", notePath, err)
	}
	return value, nil
}

// mark marks value MarkChanged when its target exists and MarkNew when it doesn't, failing with ErrTargetFolder
// when the target is a folder. Each edge is marked MarkChanged when its file exists, MarkNew when it doesn't but a
// note plans it, and left unmarked otherwise.
func mark(root string, value *sketch.Note, planned map[string]bool) error {
	target, err := stat(root, value.Target)
	if err != nil {
		return err
	}
	switch {
	case target == nil:
		value.Mark = sketch.MarkNew
	case target.IsDir():
		return ErrTargetFolder
	default:
		value.Mark = sketch.MarkChanged
	}
	for index := range value.Edges {
		edge := &value.Edges[index]
		linked, err := stat(root, edge.To)
		if err != nil {
			return err
		}
		switch {
		case linked != nil:
			edge.Mark = sketch.MarkChanged
		case planned[edge.To]:
			edge.Mark = sketch.MarkNew
		}
	}
	return nil
}

// stat returns the file info at the slash-separated repo path, or nil when nothing is there.
func stat(root, repoPath string) (fs.FileInfo, error) {
	info, err := os.Stat(filepath.Join(root, filepath.FromSlash(repoPath)))
	if errors.Is(err, fs.ErrNotExist) {
		return nil, nil
	}
	return info, err
}
