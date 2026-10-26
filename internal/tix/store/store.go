// Package store provides tix's ticket store: the .tix directory, its config, and the ticket files in it.
package store

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/theroymind/railyard/internal/contracts/ticket"
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

func Open(workingDirectory string) (Store, error) {
	directory, err := FindDirectory(workingDirectory)
	if err != nil {
		return Store{}, err
	}
	config, err := ReadConfig(directory)
	if err != nil {
		return Store{}, err
	}
	return Store{Directory: directory, Prefix: config.Prefix}, nil
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

func (store Store) ResolveTicketID(reference string) (string, error) {
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

func (store Store) CreateTicket(value ticket.Ticket) error {
	text, err := SerializeTicket(value)
	if err != nil {
		return err
	}
	return writeAtomically(store.TicketPath(value.ID), text)
}

func (store Store) UpdateTicket(id string, changes []Change) error {
	file, err := store.ReadTicketFile(id)
	if err != nil {
		return err
	}
	text, err := UpdateTicketText(file.Text, changes, file.Path)
	if err != nil {
		return err
	}
	return writeAtomically(file.Path, text)
}
