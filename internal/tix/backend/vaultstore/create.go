package vaultstore

import (
	"errors"
	"io/fs"
	"maps"
	"os"
	"path/filepath"
	"slices"

	"github.com/theroymind/go-tix/contracts/ticket"
	"github.com/theroymind/go-tix/contracts/timestamp"
	"github.com/theroymind/go-tix/internal/tix/backend"
	"github.com/theroymind/go-tix/internal/tix/vault"
)

// Create writes a new note for value and returns its slug, the slug of value's title. The note starts from the
// template of the note type that maps onto value's type, and goes in the directory that holds most notes of that
// type, or the directory the profile's ticket_directories names for it. Its status, created_at, updated_at, parent,
// dependencies, external_id and external_url properties, H1, description paragraph and criteria are filled from
// value; an empty field keeps the template's text. Priority and tags aren't written. The
// note must read back with value's title, status, links and criteria before it is written, and nothing is written
// when its slug is taken.
func (store Store) Create(value ticket.Ticket) (string, error) {
	noteType, mapping, err := store.templatedType(value.Type)
	if err != nil {
		return "", err
	}
	slug := vault.Slugify(value.Title)
	if slug == "" {
		return "", emptySlug(value.Title)
	}
	notes, err := store.ticketNotes()
	if err != nil {
		return "", err
	}
	if existing, exists := notes[slug]; exists {
		return "", slugTaken(slug, existing.note.Path)
	}
	configured := store.Profile.TicketDirectories[noteType]
	directory := store.typeDirectory(store.pathsOfType(notes, noteType), configured)
	if err := os.MkdirAll(directory, backend.DirectoryPermission); err != nil {
		return "", err
	}
	path := filepath.Join(directory, slug+noteExtension)
	if _, err := os.Stat(path); !errors.Is(err, fs.ErrNotExist) {
		if err != nil {
			return "", err
		}
		return "", slugTaken(slug, path)
	}
	note, err := vault.Read(filepath.Join(store.Root, store.Profile.TicketTemplates[noteType]))
	if err != nil {
		return "", err
	}
	note.Path, note.Slug = path, slug
	if err := store.fill(note, mapping, value); err != nil {
		return "", err
	}
	text := note.Render()
	written, err := vault.Parse(path, text)
	if err != nil {
		return "", err
	}
	created, err := toTicket(store.Profile, mapping, written)
	if err != nil {
		return "", err
	}
	if err := checkKept(path, value, created); err != nil {
		return "", err
	}
	return slug, backend.WriteAtomically(path, text)
}

// templatedType returns the first note type, in name order, that maps onto ticketType and has a template.
func (store Store) templatedType(ticketType ticket.Type) (string, backend.TicketMapping, error) {
	for _, noteType := range slices.Sorted(maps.Keys(store.Profile.Tickets)) {
		mapping := store.Profile.Tickets[noteType]
		if mapping.TicketType == ticketType && store.Profile.TicketTemplates[noteType] != "" {
			return noteType, mapping, nil
		}
	}
	return "", backend.TicketMapping{}, noTemplate(ticketType)
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

// pathsOfType returns the paths of the ticket notes whose type property is noteType.
func (store Store) pathsOfType(notes map[string]ticketNote, noteType string) []string {
	var paths []string
	for _, found := range notes {
		if propertyText(found.note.Property(store.Profile.TypeProperty)) == noteType {
			paths = append(paths, found.note.Path)
		}
	}
	return paths
}

// fill writes value's status, timestamps, links, external id and URL, title, description and criteria into the
// template's note.
func (store Store) fill(note *vault.Note, mapping backend.TicketMapping, value ticket.Ticket) error {
	noteStatus, err := noteStatusOf(mapping, note.Path, value.Status)
	if err != nil {
		return err
	}
	if err := note.SetProperty(statusProperty, noteStatus); err != nil {
		return err
	}
	if err := setTimestamps(note, value); err != nil {
		return err
	}
	if value.Parent != "" {
		if err := setLinkProperty(note, mapping, ticket.FieldParent, value.Parent); err != nil {
			return err
		}
	}
	if len(value.Dependencies) > 0 {
		if err := setLinkProperty(note, mapping, ticket.FieldDependencies, value.Dependencies); err != nil {
			return err
		}
	}
	if err := setTextFields(note, mapping, value); err != nil {
		return err
	}
	if err := setTitle(note, value.Title); err != nil {
		return err
	}
	if value.Description != "" {
		if err := setDescription(store.Profile, note, value.Description); err != nil {
			return err
		}
	}
	if len(value.Acceptance) > 0 {
		return setCriteria(store.Profile, note, value.Acceptance)
	}
	return nil
}

func setTimestamps(note *vault.Note, value ticket.Ticket) error {
	stamps := []struct {
		field  string
		moment timestamp.Timestamp
	}{{ticket.FieldCreatedAt, value.CreatedAt}, {ticket.FieldUpdatedAt, value.UpdatedAt}}
	for _, stamp := range stamps {
		if stamp.moment == "" {
			continue
		}
		if err := note.SetProperty(stamp.field, stamp.moment); err != nil {
			return err
		}
	}
	return nil
}

func setTextFields(note *vault.Note, mapping backend.TicketMapping, value ticket.Ticket) error {
	texts := []struct {
		field string
		text  string
	}{{ticket.FieldExternalID, value.ExternalID}, {ticket.FieldExternalURL, value.ExternalURL}}
	for _, text := range texts {
		if text.text == "" {
			continue
		}
		property, isHeld := textProperty(mapping, text.field)
		if !isHeld {
			return fieldUnsupported(text.field)
		}
		if err := note.SetProperty(property, text.text); err != nil {
			return err
		}
	}
	return nil
}

// setLinkProperty writes links, as plain slugs, to the first property by name that the mapping maps to field.
func setLinkProperty(note *vault.Note, mapping backend.TicketMapping, field string, links any) error {
	property, isMapped := linkProperty(mapping, field)
	if !isMapped {
		return fieldUnsupported(field)
	}
	return note.SetProperty(property, links)
}

// setTitle rewrites the first H1 to read title, or inserts one before the body's first text.
func setTitle(note *vault.Note, title string) error {
	for _, heading := range note.Headings() {
		if heading.Level == titleLevel {
			return note.SetHeadingText(heading.Line, title)
		}
	}
	return note.InsertHeading(titleLevel, title)
}

// setDescription replaces the text under the profile's description heading, up to the next heading, with
// description, or under the first H1 when the note has no such heading.
func setDescription(profile backend.Profile, note *vault.Note, description string) error {
	headings := note.Headings()
	for _, heading := range headings {
		if heading.Level > titleLevel && heading.Text == profile.DescriptionHeading {
			return note.SetTextUnder(heading.Line, description)
		}
	}
	for _, heading := range headings {
		if heading.Level == titleLevel {
			return note.SetTextUnder(heading.Line, description)
		}
	}
	return nil
}

// checkKept fails when the created ticket doesn't read back the title, status, links, external id and URL and
// criteria value gives. An empty link or text field or no criteria in value leave the template's.
func checkKept(path string, value, created ticket.Ticket) error {
	switch {
	case created.Title != value.Title:
		return notKept(path, ticket.FieldTitle)
	case created.Status != value.Status:
		return notKept(path, ticket.FieldStatus)
	case value.Parent != "" && created.Parent != value.Parent:
		return notKept(path, ticket.FieldParent)
	case len(value.Dependencies) > 0 && !slices.Equal(created.Dependencies, value.Dependencies):
		return notKept(path, ticket.FieldDependencies)
	case value.ExternalID != "" && created.ExternalID != value.ExternalID:
		return notKept(path, ticket.FieldExternalID)
	case value.ExternalURL != "" && created.ExternalURL != value.ExternalURL:
		return notKept(path, ticket.FieldExternalURL)
	case len(value.Acceptance) > 0 && !slices.Equal(created.Acceptance, ticket.NewCriteria(criterionTexts(value.Acceptance))):
		return notKept(path, ticket.FieldAcceptance)
	}
	return nil
}

func criterionTexts(criteria []ticket.Criterion) []string {
	texts := make([]string, len(criteria))
	for index, criterion := range criteria {
		texts[index] = criterion.Text
	}
	return texts
}
