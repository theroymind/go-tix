// Package repo loads a repo's sketch notes and the tree the explorer shows from one listing of its files.
package repo

import (
	"fmt"
	"slices"
	"strings"

	"github.com/theroymind/go-tix/contracts/sketch"
	"github.com/theroymind/go-tix/internal/sketch/marks"
	"github.com/theroymind/go-tix/internal/sketch/note"
	"github.com/theroymind/go-tix/internal/sketch/tree"
)

// Repo is a repo's notes and tree as they are on disk.
type Repo struct {
	Tree     *tree.Node
	Notes    []sketch.Note // the notes that read and validate, marked against their targets, in path order
	Problems []error       // one for each note that didn't, naming it
}

// Load lists the files under root once, reads and validates every note among them, marks each against its target,
// and builds the tree. A note that doesn't read joins Problems and still shows in the tree, with its problem, so it
// can be opened and fixed. Only failing to list the files fails Load.
func Load(root string) (Repo, error) {
	files, err := tree.Files(root)
	if err != nil {
		return Repo{}, err
	}
	reader := marks.NewReader(root)
	repo := Repo{Notes: []sketch.Note{}}
	var unread []sketch.Note
	problems := map[string]error{}
	for _, file := range files {
		if !strings.HasSuffix(file, sketch.NoteSuffix) {
			continue
		}
		value, err := loadNote(reader, file)
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

// loadNote loads the note at notePath, checks it against the contract, and marks it against its target.
func loadNote(reader *marks.Reader, notePath string) (sketch.Note, error) {
	value, err := note.Load(reader.Root, notePath)
	if err != nil {
		return sketch.Note{}, err
	}
	if err := value.Validate(); err != nil {
		return sketch.Note{}, fmt.Errorf("%s: %w", notePath, err)
	}
	if err := reader.MarkNote(&value); err != nil {
		return sketch.Note{}, fmt.Errorf("%s: %w", notePath, err)
	}
	return value, nil
}
