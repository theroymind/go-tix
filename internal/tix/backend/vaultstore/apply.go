package vaultstore

import (
	"fmt"
	"maps"
	"slices"

	"github.com/theroymind/railyard/internal/contracts/ticket"
	"github.com/theroymind/railyard/internal/tix/backend"
)

// Apply makes every change to the note in memory, checks it then maps onto a valid ticket, and writes it atomically;
// on any failure the note is left as it was. The note need not map before the changes, so a change can repair it.
// Criterion changes edit the checkboxes under the profile's criteria heading, numbered from 1 as Read returns them,
// and fail on a note without that heading.
func (store Store) Apply(id string, changes []backend.Change) error {
	found, err := store.ticketNote(id)
	if err != nil {
		return err
	}
	if len(changes) == 0 {
		_, err := toTicket(store.Profile, found.mapping, found.note)
		return err
	}
	for _, change := range changes {
		if err := store.applyChange(found, change); err != nil {
			return err
		}
	}
	if _, err := toTicket(store.Profile, found.mapping, found.note); err != nil {
		return err
	}
	return backend.WriteAtomically(found.note.Path, found.note.Render())
}

func (store Store) applyChange(found ticketNote, change backend.Change) error {
	set, isSetField := change.(backend.SetField)
	if !isSetField {
		return store.applyCriterionChange(found.note, change)
	}
	switch {
	case set.Field == ticket.FieldStatus:
		noteStatus, err := noteStatusOf(found, set.Value)
		if err != nil {
			return err
		}
		return found.note.SetProperty(statusProperty, noteStatus)
	case timestampFields[set.Field] && found.note.Property(set.Field) == nil:
		// Vault notes carry no timestamps; one is written only where the note already keeps it.
		return nil
	case unheldFields[set.Field] || set.Field == store.Profile.TypeProperty || found.mapping.Fields[set.Field] != "":
		return fieldUnsupported(set.Field)
	default:
		return found.note.SetProperty(set.Field, set.Value)
	}
}

// noteStatusOf returns the note status that value names: the note status a ticket status maps back from, or a note
// status as written.
func noteStatusOf(found ticketNote, value any) (string, error) {
	text := fmt.Sprint(value)
	for noteStatus, status := range found.mapping.Statuses {
		if string(status) == text {
			return noteStatus, nil
		}
	}
	if _, isNoteStatus := found.mapping.Statuses[text]; isNoteStatus {
		return text, nil
	}
	return "", unknownStatus(found.note.Path, text, slices.Sorted(maps.Keys(found.mapping.Statuses)))
}
