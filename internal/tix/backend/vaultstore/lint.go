package vaultstore

import (
	"io/fs"
	"maps"
	"path/filepath"
	"slices"
	"strings"

	"github.com/theroymind/go-tix/internal/tix/backend"
	"github.com/theroymind/go-tix/internal/tix/vault"
)

var _ backend.LintReader = Store{}

// LintNotes returns every note whose type is a ticket or doc type, in slug order.
func (store Store) LintNotes() ([]backend.LintNote, error) {
	var notes []backend.LintNote
	err := store.walkNotes(func(note *vault.Note) error {
		noteType := propertyText(note.Property(store.Profile.TypeProperty))
		if !store.Profile.IsNoteType(noteType) {
			return nil
		}
		path, err := filepath.Rel(store.Base, note.Path)
		if err != nil {
			return err
		}
		notes = append(notes, backend.LintNote{
			ID:               note.Slug,
			Type:             noteType,
			Path:             path,
			Summary:          strings.TrimSpace(propertyText(note.Property(store.Profile.SummaryProperty))),
			Headings:         headingTexts(note),
			RequiredSections: store.Profile.RequiredSections[noteType],
			Links:            store.lintLinks(note, noteType),
		})
		return nil
	})
	slices.SortFunc(notes, func(left, right backend.LintNote) int { return strings.Compare(left.ID, right.ID) })
	return notes, err
}

// lintLinks returns the slugs a note's wikilinks name, then those its ticket type's link properties name, once each.
func (store Store) lintLinks(note *vault.Note, noteType string) []string {
	links := vault.Wikilinks(string(note.Render()))
	mapping := store.Profile.Tickets[noteType]
	for _, property := range slices.Sorted(maps.Keys(mapping.Fields)) {
		links = append(links, propertyLinks(note.Property(property))...)
	}
	return uniqueLinks(links)
}

func headingTexts(note *vault.Note) []string {
	var texts []string
	for _, heading := range note.Headings() {
		texts = append(texts, heading.Text)
	}
	return texts
}

// LinkTargets returns the slug of every note under Root and the name of every other file there, sorted. Hidden
// directories are skipped.
func (store Store) LinkTargets() ([]string, error) {
	var targets []string
	err := filepath.WalkDir(store.Root, func(path string, entry fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if entry.IsDir() {
			if path != store.Root && strings.HasPrefix(entry.Name(), hiddenPrefix) {
				return filepath.SkipDir
			}
			return nil
		}
		if filepath.Ext(path) == noteExtension {
			targets = append(targets, vault.SlugOf(path))
			return nil
		}
		targets = append(targets, entry.Name())
		return nil
	})
	slices.Sort(targets)
	return targets, err
}
