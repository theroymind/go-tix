// Package vaultstore provides the vault backend, which reads tickets from the notes of an Obsidian vault, mapping
// them through the Profile in .tix/config.yml.
package vaultstore

import (
	"io/fs"
	"maps"
	"path/filepath"
	"slices"
	"strings"

	"github.com/theroymind/railyard/internal/contracts/ticket"
	"github.com/theroymind/railyard/internal/tix/backend"
	"github.com/theroymind/railyard/internal/tix/vault"
)

// Store reads the ticket notes under Root. Its Profile has its defaults filled in.
type Store struct {
	Root    string
	Profile backend.Profile
}

// ticketNote is a note the Profile maps onto a ticket.
type ticketNote struct {
	note    *vault.Note
	mapping backend.TicketMapping
}

var (
	_ backend.Backend          = Store{}
	_ backend.TextReader       = Store{}
	_ backend.CriterionTracker = Store{}
)

// TracksCriteria marks a vault's criteria as carrying a done state: each is a checkbox, ticked or not.
func (Store) TracksCriteria() {}

// Open is the vault backend's backend.Opener. The vault's root is relative to the directory holding .tix.
func Open(tixDirectory string, config backend.Config) (backend.Backend, error) {
	var profile backend.Profile
	if config.Vault != nil {
		profile = *config.Vault
	}
	profile = profile.WithDefaults()
	return Store{Root: filepath.Join(filepath.Dir(tixDirectory), profile.Root), Profile: profile}, nil
}

// ticketNotes parses every note under Root and returns those whose type is a ticket type, by slug. Hidden
// directories and the templates the Profile names are skipped.
func (store Store) ticketNotes() (map[string]ticketNote, error) {
	templates := map[string]bool{}
	for _, template := range store.Profile.Templates() {
		templates[filepath.Join(store.Root, template)] = true
	}
	notes := map[string]ticketNote{}
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
		if filepath.Ext(path) != noteExtension || templates[path] {
			return nil
		}
		note, err := vault.Read(path)
		if err != nil {
			return err
		}
		mapping, isTicket := ticketMapping(store.Profile, note)
		if !isTicket {
			return nil
		}
		if other, exists := notes[note.Slug]; exists {
			return duplicateSlug(path, other.note.Path)
		}
		notes[note.Slug] = ticketNote{note: note, mapping: mapping}
		return nil
	})
	return notes, err
}

func (store Store) ticketNote(id string) (ticketNote, error) {
	notes, err := store.ticketNotes()
	if err != nil {
		return ticketNote{}, err
	}
	found, exists := notes[id]
	if !exists {
		return ticketNote{}, backend.NoMatch(id)
	}
	return found, nil
}

// Resolve returns the slug of the ticket note that reference names exactly or is the unique prefix of.
func (store Store) Resolve(reference string) (string, error) {
	notes, err := store.ticketNotes()
	if err != nil {
		return "", err
	}
	return resolveReference(reference, slices.Sorted(maps.Keys(notes)))
}

func resolveReference(reference string, slugs []string) (string, error) {
	if reference == "" {
		return "", backend.NoMatch(reference)
	}
	if slices.Contains(slugs, reference) {
		return reference, nil
	}
	var matches []string
	for _, slug := range slugs {
		if strings.HasPrefix(slug, reference) {
			matches = append(matches, slug)
		}
	}
	switch len(matches) {
	case 0:
		return "", backend.NoMatch(reference)
	case 1:
		return matches[0], nil
	default:
		return "", backend.Ambiguous(reference, matches)
	}
}

// List returns every ticket note's ticket, in slug order.
func (store Store) List() ([]ticket.Ticket, error) {
	notes, err := store.ticketNotes()
	if err != nil {
		return nil, err
	}
	tickets := make([]ticket.Ticket, 0, len(notes))
	for _, slug := range slices.Sorted(maps.Keys(notes)) {
		mapped, err := toTicket(store.Profile, notes[slug].mapping, notes[slug].note)
		if err != nil {
			return nil, err
		}
		tickets = append(tickets, mapped)
	}
	return tickets, nil
}

func (store Store) Read(id string) (ticket.Ticket, error) {
	found, err := store.ticketNote(id)
	if err != nil {
		return ticket.Ticket{}, err
	}
	return toTicket(store.Profile, found.mapping, found.note)
}

// ReadText returns the note's Markdown, once it has mapped onto a valid ticket.
func (store Store) ReadText(id string) ([]byte, error) {
	found, err := store.ticketNote(id)
	if err != nil {
		return nil, err
	}
	if _, err := toTicket(store.Profile, found.mapping, found.note); err != nil {
		return nil, err
	}
	return found.note.Render(), nil
}
