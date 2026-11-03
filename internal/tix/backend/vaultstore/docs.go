package vaultstore

import (
	"errors"
	"io/fs"
	"maps"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"time"

	"go.yaml.in/yaml/v3"

	"github.com/theroymind/railyard/internal/tix/backend"
	"github.com/theroymind/railyard/internal/tix/vault"
)

var _ backend.DocStore = Store{}

// docNote is a note the Profile maps onto a doc.
type docNote struct {
	note    *vault.Note
	docType string
	mapping backend.DocMapping
}

// docNotes parses every note under Root and returns those whose type is a doc type, by slug.
func (store Store) docNotes() (map[string]docNote, error) {
	notes := map[string]docNote{}
	err := store.walkNotes(func(note *vault.Note) error {
		docType := propertyText(note.Property(store.Profile.TypeProperty))
		mapping, isDoc := store.Profile.Docs[docType]
		if !isDoc {
			return nil
		}
		if other, exists := notes[note.Slug]; exists {
			return duplicateSlug(note.Path, other.note.Path)
		}
		notes[note.Slug] = docNote{note: note, docType: docType, mapping: mapping}
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
		return docNote{}, backend.NoDocMatch(id)
	}
	return found, nil
}

// ResolveDoc returns the slug of the doc note that reference names exactly or is the unique prefix of.
func (store Store) ResolveDoc(reference string) (string, error) {
	notes, err := store.docNotes()
	if err != nil {
		return "", err
	}
	id, err := resolveReference(reference, slices.Sorted(maps.Keys(notes)))
	if errors.Is(err, backend.ErrNoMatch) {
		return "", backend.NoDocMatch(reference)
	}
	return id, err
}

// ListDocs returns the docs of docType, or of every type when it is "", in slug order. A type the profile doesn't
// define fails naming the types it does.
func (store Store) ListDocs(docType string) ([]backend.Doc, error) {
	types := slices.Sorted(maps.Keys(store.Profile.Docs))
	if docType != "" && !slices.Contains(types, docType) {
		return nil, unknownType(docType, types)
	}
	notes, err := store.docNotes()
	if err != nil {
		return nil, err
	}
	docs := []backend.Doc{}
	for _, slug := range slices.Sorted(maps.Keys(notes)) {
		if docType != "" && notes[slug].docType != docType {
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
// atomically; on any failure the note is left as it was. A status must be one of the type's statuses.
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

// docStatusOf returns the status value names, which must be one of the type's statuses when the type lists any.
func docStatusOf(found docNote, value any) (string, error) {
	text := propertyTextOf(value)
	if len(found.mapping.Statuses) > 0 && !slices.Contains(found.mapping.Statuses, text) {
		return "", unknownStatus(found.note.Path, text, found.mapping.Statuses)
	}
	return text, nil
}

// CreateDoc writes a new note of type and returns its slug, the slug of value.Title. The note starts from the type's
// template with {{title}} and {{date}} filled in, goes in the directory holding most notes of the type, and takes
// value.Status, or the type's first status, and value.Properties. Nothing is written when the slug is taken by a
// ticket, a doc or a file, or when the type has no template.
func (store Store) CreateDoc(docType string, value backend.Doc) (string, error) {
	mapping, isDoc := store.Profile.Docs[docType]
	if !isDoc {
		return "", unknownType(docType, slices.Sorted(maps.Keys(store.Profile.Docs)))
	}
	if mapping.Template == "" {
		return "", noDocTemplate(docType, store.templatedDocTypes())
	}
	slug := vault.Slugify(value.Title)
	if slug == "" {
		return "", emptySlug(value.Title)
	}
	tickets, err := store.ticketNotes()
	if err != nil {
		return "", err
	}
	if existing, exists := tickets[slug]; exists {
		return "", slugTaken(slug, existing.note.Path)
	}
	docs, err := store.docNotes()
	if err != nil {
		return "", err
	}
	if existing, exists := docs[slug]; exists {
		return "", slugTaken(slug, existing.note.Path)
	}
	var paths []string
	for _, found := range docs {
		if found.docType == docType {
			paths = append(paths, found.note.Path)
		}
	}
	directory, err := store.typeDirectory(paths, docType)
	if err != nil {
		return "", err
	}
	path := filepath.Join(directory, slug+noteExtension)
	if _, err := os.Stat(path); !errors.Is(err, fs.ErrNotExist) {
		if err != nil {
			return "", err
		}
		return "", slugTaken(slug, path)
	}
	template, err := os.ReadFile(filepath.Join(store.Root, mapping.Template))
	if err != nil {
		return "", err
	}
	text := strings.ReplaceAll(string(template), backend.PlaceholderTitle, value.Title)
	text = strings.ReplaceAll(text, backend.PlaceholderDate, time.Now().Format(backend.DateLayout))
	note, err := vault.Parse(path, []byte(text))
	if err != nil {
		return "", err
	}
	status := value.Status
	if status == "" && len(mapping.Statuses) > 0 {
		status = mapping.Statuses[0]
	}
	if status != "" {
		if err := note.SetProperty(statusProperty, status); err != nil {
			return "", err
		}
	}
	for _, property := range slices.Sorted(maps.Keys(value.Properties)) {
		if err := note.SetProperty(property, value.Properties[property]); err != nil {
			return "", err
		}
	}
	rendered := note.Render()
	written, err := vault.Parse(path, rendered)
	if err != nil {
		return "", err
	}
	if _, err := store.toDoc(docNote{note: written, docType: docType, mapping: mapping}); err != nil {
		return "", err
	}
	return slug, backend.WriteAtomically(path, rendered)
}

// Backlinks returns the slugs of every note under Root whose wikilinks name id, in slug order.
func (store Store) Backlinks(id string) ([]string, error) {
	var slugs []string
	err := store.walkNotes(func(note *vault.Note) error {
		if slices.Contains(vault.Wikilinks(string(note.Render())), id) {
			slugs = append(slugs, note.Slug)
		}
		return nil
	})
	slices.Sort(slugs)
	return slugs, err
}

// toDoc maps a doc note onto a doc: the slug is its id, the first H1 its title (else the slug), its status must be
// one of the type's statuses when the type lists any, its other frontmatter fills Properties, the wikilinks in the
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
		Type:       found.docType,
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

// templatedDocTypes returns the doc types the profile gives a template, sorted.
func (store Store) templatedDocTypes() []string {
	var templated []string
	for _, docType := range slices.Sorted(maps.Keys(store.Profile.Docs)) {
		if store.Profile.Docs[docType].Template != "" {
			templated = append(templated, docType)
		}
	}
	return templated
}
