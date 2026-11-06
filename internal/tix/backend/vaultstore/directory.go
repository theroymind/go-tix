package vaultstore

import (
	"errors"
	"io/fs"
	"maps"
	"os"
	"path/filepath"
	"slices"
	"strings"

	"github.com/theroymind/go-tix/internal/tix/backend"
)

// ticketDirectory returns the directory a ticket note of noteType named slug, under parent, belongs in. A configured
// directory fills its placeholders: {slug} with slug, and {parent} with the parent note's directory, or with the
// profile's uncategorized directory when there's no parent. Without one, the note goes where most notes of its type
// already are, or in the vault's root when the type has none yet.
func (store Store) ticketDirectory(notes map[string]ticketNote, noteType, slug, parent string) (string, error) {
	pattern := store.Profile.TicketDirectories[noteType]
	if pattern == "" {
		return store.typeDirectory(store.pathsOfType(notes, noteType), ""), nil
	}
	parentDirectory := store.Profile.UncategorizedDirectory
	if parent != "" {
		parentNote, exists := notes[parent]
		if !exists {
			return "", backend.NoMatch(parent)
		}
		relative, err := filepath.Rel(store.Root, filepath.Dir(parentNote.note.Path))
		if err != nil {
			return "", err
		}
		parentDirectory = relative
	}
	directory := strings.ReplaceAll(pattern, backend.PlaceholderSlug, slug)
	directory = strings.ReplaceAll(directory, backend.PlaceholderParent, parentDirectory)
	return filepath.Join(store.Root, filepath.Clean(directory)), nil
}

// typeDirectory returns the configured directory when there is one, or else the directory holding the most of the
// type's notes among paths, the first by name on a tie, or else the vault's root when the type has no notes yet.
func (store Store) typeDirectory(paths []string, configured string) string {
	if configured != "" {
		return filepath.Join(store.Root, configured)
	}
	counts := map[string]int{}
	for _, path := range paths {
		counts[filepath.Dir(path)]++
	}
	directory := ""
	for _, candidate := range slices.Sorted(maps.Keys(counts)) {
		if counts[candidate] > counts[directory] {
			directory = candidate
		}
	}
	if directory == "" {
		return store.Root
	}
	return directory
}

// moveNote writes text to the note's path in directory, removes it from where it was, and removes each directory it
// left empty, up to the vault's root.
func (store Store) moveNote(from, directory string, text []byte) (string, error) {
	path := filepath.Join(directory, filepath.Base(from))
	if _, err := os.Stat(path); !errors.Is(err, fs.ErrNotExist) {
		if err != nil {
			return "", err
		}
		return "", slugTaken(strings.TrimSuffix(filepath.Base(from), noteExtension), path)
	}
	if err := os.MkdirAll(directory, backend.DirectoryPermission); err != nil {
		return "", err
	}
	if err := backend.WriteAtomically(path, text); err != nil {
		return "", err
	}
	if err := os.Remove(from); err != nil {
		return "", err
	}
	store.removeEmptyDirectories(filepath.Dir(from))
	return path, nil
}

func (store Store) removeEmptyDirectories(directory string) {
	for directory != store.Root && strings.HasPrefix(directory, store.Root) {
		if os.Remove(directory) != nil {
			return
		}
		directory = filepath.Dir(directory)
	}
}
