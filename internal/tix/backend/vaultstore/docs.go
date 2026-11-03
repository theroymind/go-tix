package vaultstore

import (
	"maps"
	"path/filepath"
	"slices"
	"strings"

	"go.yaml.in/yaml/v3"

	"github.com/theroymind/railyard/internal/tix/backend"
	"github.com/theroymind/railyard/internal/tix/vault"
)

var _ backend.DocStore = Store{}

// docNote is a note the Profile maps onto a doc.
type docNote struct {
	note    *vault.Note
	kind    string
	mapping backend.DocMapping
}

// docNotes parses every note under Root and returns those whose type is a doc kind, by slug.
func (store Store) docNotes() (map[string]docNote, error) {
	notes := map[string]docNote{}
	err := store.walkNotes(func(note *vault.Note) error {
		kind := propertyText(note.Property(store.Profile.TypeProperty))
		mapping, isDoc := store.Profile.Docs[kind]
		if !isDoc {
			return nil
		}
		if other, exists := notes[note.Slug]; exists {
			return duplicateSlug(note.Path, other.note.Path)
		}
		notes[note.Slug] = docNote{note: note, kind: kind, mapping: mapping}
		return nil
	})
	return notes, err
}

func (store Store) docNote(id string) (docNote, error) {
	notes, err := store.docNotes()
	if err != nil {
		return docNote{}, err
	}
	found, exists := notes[id]
	if !exists {
		return docNote{}, backend.NoMatch(id)
	}
	return found, nil
}

// ResolveDoc returns the slug of the doc note that reference names exactly or is the unique prefix of.
func (store Store) ResolveDoc(reference string) (string, error) {
	notes, err := store.docNotes()
	if err != nil {
		return "", err
	}
	return resolveReference(reference, slices.Sorted(maps.Keys(notes)))
}

// ListDocs returns the docs of kind, or of every kind when kind is "", in slug order. A kind the profile doesn't
// define fails naming the kinds it does.
func (store Store) ListDocs(kind string) ([]backend.Doc, error) {
	kinds := slices.Sorted(maps.Keys(store.Profile.Docs))
	if kind != "" && !slices.Contains(kinds, kind) {
		return nil, unknownKind(kind, kinds)
	}
	notes, err := store.docNotes()
	if err != nil {
		return nil, err
	}
	docs := []backend.Doc{}
	for _, slug := range slices.Sorted(maps.Keys(notes)) {
		if kind != "" && notes[slug].kind != kind {
			continue
		}
		mapped, err := store.toDoc(notes[slug])
		if err != nil {
			return nil, err
		}
		docs = append(docs, mapped)
	}
	return docs, nil
}

func (store Store) ReadDoc(id string) (backend.Doc, error) {
	found, err := store.docNote(id)
	if err != nil {
		return backend.Doc{}, err
	}
	return store.toDoc(found)
}

// ReadDocText returns the note's Markdown, once it has mapped onto a valid doc.
func (store Store) ReadDocText(id string) ([]byte, error) {
	found, err := store.docNote(id)
	if err != nil {
		return nil, err
	}
	if _, err := store.toDoc(found); err != nil {
		return nil, err
	}
	return found.note.Render(), nil
}

// ApplyDoc sets properties on the doc note in memory, checks it still maps onto a valid doc, and writes it
// atomically; on any failure the note is left as it was. A status must be one of the kind's statuses.
func (store Store) ApplyDoc(id string, changes []backend.Change) error {
	found, err := store.docNote(id)
	if err != nil {
		return err
	}
	for _, change := range changes {
		if err := store.applyDocChange(found, change); err != nil {
			return err
		}
	}
	if _, err := store.toDoc(found); err != nil {
		return err
	}
	return backend.WriteAtomically(found.note.Path, found.note.Render())
}

func (store Store) applyDocChange(found docNote, change backend.Change) error {
	set, isSetField := change.(backend.SetField)
	if !isSetField {
		return backend.Unsupported(change, backend.KindVault)
	}
	switch {
	case set.Field == statusProperty:
		status, err := docStatusOf(found, set.Value)
		if err != nil {
			return err
		}
		return found.note.SetProperty(statusProperty, status)
	case timestampFields[set.Field] && found.note.Property(set.Field) == nil:
		// Vault notes carry no timestamps; one is written only where the note already keeps it.
		return nil
	case unheldDocFields[set.Field] || set.Field == store.Profile.TypeProperty:
		return fieldUnsupported(set.Field)
	default:
		return found.note.SetProperty(set.Field, set.Value)
	}
}

// docStatusOf returns the status value names, which must be one of the kind's statuses when the kind lists any.
func docStatusOf(found docNote, value any) (string, error) {
	text := propertyTextOf(value)
	if len(found.mapping.Statuses) > 0 && !slices.Contains(found.mapping.Statuses, text) {
		return "", unknownStatus(found.note.Path, text, found.mapping.Statuses)
	}
	return text, nil
}

// toDoc maps a doc note onto a doc: the slug is its id, the first H1 its title (else the slug), its status must be
// one of the kind's statuses when the kind lists any, its other frontmatter fills Properties, the wikilinks in the
// whole note fill Links, and Path is relative to Base.
func (store Store) toDoc(found docNote) (backend.Doc, error) {
	note := found.note
	status := propertyText(note.Property(statusProperty))
	if len(found.mapping.Statuses) > 0 && !slices.Contains(found.mapping.Statuses, status) {
		return backend.Doc{}, unknownStatus(note.Path, status, found.mapping.Statuses)
	}
	title, hasTitle := note.Heading(titleLevel)
	if !hasTitle || strings.TrimSpace(title) == "" {
		title = note.Slug
	}
	properties, err := docProperties(store.Profile, note)
	if err != nil {
		return backend.Doc{}, err
	}
	path, err := filepath.Rel(store.Base, note.Path)
	if err != nil {
		return backend.Doc{}, err
	}
	return backend.Doc{
		ID:         note.Slug,
		Kind:       found.kind,
		Title:      title,
		Status:     status,
		Properties: properties,
		Links:      uniqueLinks(vault.Wikilinks(string(note.Render()))),
		Path:       path,
	}, nil
}

// docProperties decodes the frontmatter, less the type property and status, which the doc carries as fields.
func docProperties(profile backend.Profile, note *vault.Note) (map[string]any, error) {
	if note.Frontmatter == nil {
		return nil, nil
	}
	var properties map[string]any
	if err := note.Frontmatter.Decode(&properties); err != nil {
		return nil, err
	}
	delete(properties, profile.TypeProperty)
	delete(properties, statusProperty)
	if len(properties) == 0 {
		return nil, nil
	}
	return properties, nil
}

// uniqueLinks keeps each slug once, in order of first appearance.
func uniqueLinks(links []string) []string {
	var unique []string
	for _, link := range links {
		if !slices.Contains(unique, link) {
			unique = append(unique, link)
		}
	}
	return unique
}

// propertyTextOf is the text a property is set to, as a note writes it.
func propertyTextOf(value any) string {
	if text, isText := value.(string); isText {
		return text
	}
	node := &yaml.Node{}
	if err := node.Encode(value); err != nil || node.Kind != yaml.ScalarNode {
		return ""
	}
	return node.Value
}
