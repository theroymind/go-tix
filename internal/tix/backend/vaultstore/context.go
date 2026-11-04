package vaultstore

import (
	"errors"
	"maps"
	"slices"
	"strings"

	"go.yaml.in/yaml/v3"

	"github.com/theroymind/railyard/internal/tix/backend"
	"github.com/theroymind/railyard/internal/tix/vault"
)

var _ backend.ContextReader = Store{}

// ContextNotes returns every ticket and doc note, in slug order, read in one pass over the vault. A note's links are
// its parent, its dependencies and its wikilinks; its backlinks are the ticket and doc notes whose wikilinks name it.
func (store Store) ContextNotes() ([]backend.ContextNote, error) {
	tickets, err := store.ticketNotes()
	if err != nil {
		return nil, err
	}
	docs, err := store.docNotes()
	if err != nil {
		return nil, err
	}
	notes := map[string]backend.ContextNote{}
	wikilinks := map[string][]string{}
	for slug, found := range tickets {
		mapped, err := toTicket(store.Profile, found.mapping, found.note)
		if err != nil {
			return nil, err
		}
		var links []string
		if mapped.Parent != "" {
			links = append(links, mapped.Parent)
		}
		links = append(links, mapped.Dependencies...)
		noteType := propertyText(found.note.Property(store.Profile.TypeProperty))
		notes[slug], wikilinks[slug], err = store.contextNote(found.note, noteType, mapped.Title, links)
		if err != nil {
			return nil, err
		}
	}
	for slug, found := range docs {
		if _, isTicket := notes[slug]; isTicket {
			return nil, duplicateSlug(found.note.Path, tickets[slug].note.Path)
		}
		mapped, err := store.toDoc(found)
		if err != nil {
			return nil, err
		}
		notes[slug], wikilinks[slug], err = store.contextNote(found.note, mapped.Type, mapped.Title, nil)
		if err != nil {
			return nil, err
		}
	}
	return withBacklinks(notes, wikilinks), nil
}

// contextNote reads a note's summary, commits and files through the profile, and links it to the given ids and to
// every note its wikilinks name, each once and never to itself. It returns the wikilinks apart as well.
func (store Store) contextNote(note *vault.Note, noteType, title string, links []string) (backend.ContextNote, []string, error) {
	files, err := store.files(note)
	if err != nil {
		return backend.ContextNote{}, nil, err
	}
	wikilinks := withoutSelf(vault.Wikilinks(string(note.Render())), note.Slug)
	links = withoutSelf(uniqueLinks(append(links, wikilinks...)), note.Slug)
	return backend.ContextNote{
		ID:      note.Slug,
		Type:    noteType,
		Title:   title,
		Summary: strings.TrimSpace(propertyText(note.Property(store.Profile.SummaryProperty))),
		Commits: propertyTexts(note.Property(store.Profile.CommitsProperty)),
		Files:   files,
		Links:   links,
	}, wikilinks, nil
}

func withoutSelf(links []string, slug string) []string {
	return slices.DeleteFunc(links, func(link string) bool { return link == slug })
}

// files returns the paths listed under the profile's files heading, or none when the profile names no heading or
// the note lacks the section.
func (store Store) files(note *vault.Note) ([]string, error) {
	if store.Profile.FilesHeading == "" {
		return nil, nil
	}
	section, err := note.Section(store.Profile.FilesHeading)
	if errors.Is(err, vault.ErrSectionNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return listedFiles(note.Body[section.Start:section.End]), nil
}

// listedFiles reads the paths in a files section, written as a comma-separated line or as list items, with or
// without code spans around each, once each in order.
func listedFiles(lines []string) []string {
	var files []string
	for _, line := range lines {
		item := strings.TrimSpace(line)
		for _, marker := range listMarkers {
			item = strings.TrimPrefix(item, marker)
		}
		for entry := range strings.SplitSeq(item, fileSeparator) {
			file := strings.Trim(strings.TrimSpace(entry), codeSpanMarker)
			if file != "" && !slices.Contains(files, file) {
				files = append(files, file)
			}
		}
	}
	return files
}

// withBacklinks gives each note the slugs of the notes whose wikilinks, by slug, name it, and returns the notes in slug order.
// A link from a parent or dependency property isn't a wikilink, so it makes no backlink, as tix backlinks has it.
func withBacklinks(notes map[string]backend.ContextNote, wikilinks map[string][]string) []backend.ContextNote {
	backlinks := map[string][]string{}
	for _, slug := range slices.Sorted(maps.Keys(wikilinks)) {
		for _, link := range wikilinks[slug] {
			backlinks[link] = append(backlinks[link], slug)
		}
	}
	ordered := make([]backend.ContextNote, 0, len(notes))
	for _, slug := range slices.Sorted(maps.Keys(notes)) {
		note := notes[slug]
		note.Backlinks = backlinks[slug]
		ordered = append(ordered, note)
	}
	return ordered
}

// propertyTexts returns the scalar texts a property holds, as a single value or a list.
func propertyTexts(value *yaml.Node) []string {
	if value == nil {
		return nil
	}
	if value.Kind == yaml.ScalarNode {
		if text := strings.TrimSpace(value.Value); text != "" {
			return []string{text}
		}
		return nil
	}
	var texts []string
	for _, child := range value.Content {
		texts = append(texts, propertyTexts(child)...)
	}
	return texts
}
