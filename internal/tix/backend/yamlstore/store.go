// Package yamlstore provides the yaml backend, which keeps each ticket as a tix v1 YAML file in the .tix directory.
package yamlstore

import (
	"crypto/rand"
	"os"
	"path/filepath"
	"strings"

	"github.com/theroymind/go-tix/contracts/ticket"
	"github.com/theroymind/go-tix/contracts/validation"
	"github.com/theroymind/go-tix/internal/tix/backend"
)

type Store struct {
	Directory string
	Prefix    string
}

type TicketFile struct {
	Path   string
	Text   []byte
	Ticket ticket.Ticket
}

var (
	_ backend.Backend    = Store{}
	_ backend.TextReader = Store{}
)

// Open is the yaml backend's backend.Opener.
func Open(tixDirectory string, config backend.Config) (backend.Backend, error) {
	return Store{Directory: tixDirectory, Prefix: config.Prefix}, nil
}

func (store Store) TicketPath(id string) string {
	return filepath.Join(store.Directory, ticket.FileName(id))
}

func (store Store) HasTicket(id string) bool {
	_, err := os.Stat(store.TicketPath(id))
	return err == nil
}

func (store Store) ListTicketIDs() ([]string, error) {
	entries, err := os.ReadDir(store.Directory)
	if err != nil {
		return nil, err
	}
	var ids []string
	for _, entry := range entries {
		name := entry.Name()
		if id, isTicketFile := strings.CutSuffix(name, ticket.FileExtension); isTicketFile && name != ticket.ConfigFileName {
			ids = append(ids, id)
		}
	}
	return ids, nil
}

func (store Store) Resolve(reference string) (string, error) {
	ids, err := store.ListTicketIDs()
	if err != nil {
		return "", err
	}
	return ResolveReference(reference, store.Prefix, ids)
}

func (store Store) ReadTicketFile(id string) (TicketFile, error) {
	path := store.TicketPath(id)
	text, err := os.ReadFile(path)
	if err != nil {
		return TicketFile{}, err
	}
	parsed, err := ParseTicket(text, path)
	if err != nil {
		return TicketFile{}, err
	}
	if parsed.ID != id {
		return TicketFile{}, idMismatch(path, parsed.ID, id)
	}
	return TicketFile{Path: path, Text: text, Ticket: parsed}, nil
}

func (store Store) Read(id string) (ticket.Ticket, error) {
	file, err := store.ReadTicketFile(id)
	return file.Ticket, err
}

// ReadText returns the ticket file's bytes, once they've parsed as a valid ticket.
func (store Store) ReadText(id string) ([]byte, error) {
	file, err := store.ReadTicketFile(id)
	return file.Text, err
}

func (store Store) List() ([]ticket.Ticket, error) {
	ids, err := store.ListTicketIDs()
	if err != nil {
		return nil, err
	}
	tickets := make([]ticket.Ticket, 0, len(ids))
	for _, id := range ids {
		file, err := store.ReadTicketFile(id)
		if err != nil {
			return nil, err
		}
		tickets = append(tickets, file.Ticket)
	}
	return tickets, nil
}

// Create gives value a new random id under the store's prefix and writes it.
func (store Store) Create(value ticket.Ticket) (string, error) {
	id, err := ticket.GenerateID(store.Prefix, rand.Reader, store.HasTicket)
	if err != nil {
		return "", err
	}
	value.ID = id
	return id, store.CreateTicket(value)
}

// CreateTicket writes value under its own id, refusing a ticket that lacks a field every tix v1 file carries.
func (store Store) CreateTicket(value ticket.Ticket) error {
	path := store.TicketPath(value.ID)
	if err := validation.RequireFields(requiredFields, hasValue(value)); err != nil {
		return backend.InvalidFile(path, err)
	}
	if err := backend.Validate(value, path); err != nil {
		return err
	}
	text, err := SerializeTicket(value)
	if err != nil {
		return err
	}
	return backend.WriteAtomically(path, text)
}

func (store Store) Apply(id string, changes []backend.Change) error {
	file, err := store.ReadTicketFile(id)
	if err != nil {
		return err
	}
	text, err := UpdateTicketText(file.Text, changes, file.Path)
	if err != nil {
		return err
	}
	return backend.WriteAtomically(file.Path, text)
}
