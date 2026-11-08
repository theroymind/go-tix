package vaultstore

import (
	"errors"
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
	templates := map[string]map[string]string{}
	err := store.walkNotes(func(note *vault.Note) error {
		noteType := propertyText(note.Property(store.Profile.TypeProperty))
		if !store.Profile.IsNoteType(noteType) {
			return nil
		}
		path, err := filepath.Rel(store.Base, note.Path)
		if err != nil {
			return err
		}
		templateSections, err := store.templateSections(templates, noteType)
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
			TemplateSections: unchangedSections(note, templateSections, store.Profile.RequiredSections[noteType]),
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
		if !slices.Contains(backend.LinkFields, mapping.Fields[property]) {
			continue
		}
		links = append(links, propertyLinks(note.Property(property))...)
	}
	return uniqueLinks(links)
}

// templateSections returns the text under each heading below the title in the template noteType starts from, by
// heading, skipping empty ones, reading each template once into cache. A type without a template has none.
func (store Store) templateSections(cache map[string]map[string]string, noteType string) (map[string]string, error) {
	if sections, isCached := cache[noteType]; isCached {
		return sections, nil
	}
	path := store.Profile.TicketTemplates[noteType]
	if path == "" {
		path = store.Profile.Docs[noteType].Template
	}
	sections := map[string]string{}
	cache[noteType] = sections
	if path == "" {
		return sections, nil
	}
	template, err := vault.Read(filepath.Join(store.Root, path))
	if errors.Is(err, fs.ErrNotExist) {
		return sections, nil
	}
	if err != nil {
		return nil, err
	}
	for _, heading := range template.Headings() {
		if text := sectionText(template, heading); heading.Level > titleLevel && text != "" {
			sections[heading.Text] = text
		}
	}
	return sections, nil
}

// unchangedSections returns the note's required headings whose text is still the template's. A section the type
// doesn't require may keep its template's text on purpose.
func unchangedSections(note *vault.Note, templateSections map[string]string, required []string) []string {
	var unchanged []string
	for _, heading := range note.Headings() {
		template, hasTemplate := templateSections[heading.Text]
		isRequired := slices.Contains(required, heading.Text)
		if heading.Level > titleLevel && hasTemplate && isRequired && sectionText(note, heading) == template {
			unchanged = append(unchanged, heading.Text)
		}
	}
	return unchanged
}

// sectionText is the text under heading, up to the next heading of any level, trimmed of surrounding space.
func sectionText(note *vault.Note, heading vault.HeadingLine) string {
	end := len(note.Body)
	for _, next := range note.Headings() {
		if next.Line > heading.Line {
			end = next.Line
			break
		}
	}
	return strings.TrimSpace(strings.Join(note.Body[heading.Line+1:end], ""))
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
