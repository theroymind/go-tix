package vaultstore

import (
	"fmt"
	"maps"
	"slices"

	"go.yaml.in/yaml/v3"

	"github.com/theroymind/railyard/internal/contracts/ticket"
	"github.com/theroymind/railyard/internal/tix/backend"
	"github.com/theroymind/railyard/internal/tix/vault"
)

// Apply makes every change to the note in memory, checks it then maps onto a valid ticket, and writes it atomically;
// on any failure the note is left as it was. The note need not map before the changes, so a change can repair it.
// Criterion changes edit the checkboxes under the profile's criteria heading, numbered from 1 as Read returns them,
// and fail on a note without that heading. Setting dependencies takes a list of ids and replaces them all.
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
	if remove, isRemoveField := change.(backend.RemoveField); isRemoveField {
		return store.removeTicketProperty(found, remove.Field)
	}
	set, isSetField := change.(backend.SetField)
	if !isSetField {
		return store.applyCriterionChange(found.note, change)
	}
	switch {
	case set.Field == ticket.FieldStatus:
		noteStatus, err := noteStatusOf(found.mapping, found.note.Path, set.Value)
		if err != nil {
			return err
		}
		return found.note.SetProperty(statusProperty, noteStatus)
	case set.Field == ticket.FieldDependencies:
		return setDependencies(found, set.Value)
	case timestampFields[set.Field] && found.note.Property(set.Field) == nil:
		// Vault notes carry no timestamps; one is written only where the note already keeps it.
		return nil
	case found.mapping.Fields[set.Field] != "":
		return mappedPropertyUnsupported(set.Field, found.mapping.Fields[set.Field])
	case unheldFields[set.Field] || set.Field == store.Profile.TypeProperty:
		return fieldUnsupported(set.Field)
	default:
		return found.note.SetProperty(set.Field, propertyValue(set.Value))
	}
}

// removeTicketProperty removes a property a ticket note holds. Its status and type property are required, its
// description is its body rather than a property, and a property the profile maps onto a field is changed through
// that field.
func (store Store) removeTicketProperty(found ticketNote, field string) error {
	switch {
	case field == statusProperty || field == store.Profile.TypeProperty:
		return backend.RequiredField(found.note.Path, field)
	case field == ticket.FieldDescription:
		return descriptionIsBody(found.note.Path)
	case found.mapping.Fields[field] != "":
		return mappedPropertyUnsupported(field, found.mapping.Fields[field])
	case unheldFields[field]:
		return fieldUnsupported(field)
	default:
		return removeProperty(found.note, field)
	}
}

// removeProperty removes key from the note's frontmatter, failing with backend.ErrNoField when it isn't there.
func removeProperty(note *vault.Note, key string) error {
	if note.Property(key) == nil {
		return backend.NoField(note.Path, key)
	}
	return note.RemoveProperty(key)
}

// propertyValue is value as a property holds it: a list of strings is a flow list, as a vault's lists are written,
// unless the property already holds a block list. Any other value is written as it is.
func propertyValue(value any) any {
	items, isList := value.([]string)
	if !isList {
		return value
	}
	list := &yaml.Node{Kind: yaml.SequenceNode, Tag: sequenceTag, Style: yaml.FlowStyle}
	for _, item := range items {
		list.Content = append(list.Content, &yaml.Node{Kind: yaml.ScalarNode, Tag: stringTag, Value: item})
	}
	return list
}

// noteStatusOf returns the note status that value names: the note status a ticket status maps back from, or a note
// status as written. An unmapped value fails naming path and the mapping's note statuses.
func noteStatusOf(mapping backend.TicketMapping, path string, value any) (string, error) {
	text := fmt.Sprint(value)
	for noteStatus, status := range mapping.Statuses {
		if string(status) == text {
			return noteStatus, nil
		}
	}
	if _, isNoteStatus := mapping.Statuses[text]; isNoteStatus {
		return text, nil
	}
	return "", unknownStatus(path, text, slices.Sorted(maps.Keys(mapping.Statuses)))
}

// setDependencies writes ids as the note's dependencies, replacing them all, to the first property by name that the
// mapping maps to dependencies. A dependency the property already holds keeps the way it was written, such as a
// wikilink; a new one is a plain slug. The list is a flow list unless the property already holds a block list. The
// note must then read back exactly those dependencies, which fails when another property also holds some.
func setDependencies(found ticketNote, value any) error {
	ids, isIDList := value.([]string)
	if !isIDList {
		return notIDList(found.note.Path, ticket.FieldDependencies)
	}
	property, isMapped := linkProperty(found.mapping, ticket.FieldDependencies)
	if !isMapped {
		return fieldUnsupported(ticket.FieldDependencies)
	}
	existing := linkNodes(found.note.Property(property))
	list := &yaml.Node{Kind: yaml.SequenceNode, Tag: sequenceTag, Style: yaml.FlowStyle}
	for _, id := range ids {
		item, isKept := existing[id]
		if !isKept {
			item = &yaml.Node{Kind: yaml.ScalarNode, Tag: stringTag, Value: id}
		}
		list.Content = append(list.Content, item)
	}
	if err := found.note.SetProperty(property, list); err != nil {
		return err
	}
	return checkDependenciesKept(found, ids)
}

// linkProperty returns the first property by name that the mapping maps to field.
func linkProperty(mapping backend.TicketMapping, field string) (string, bool) {
	for _, property := range slices.Sorted(maps.Keys(mapping.Fields)) {
		if mapping.Fields[property] == field {
			return property, true
		}
	}
	return "", false
}

// linkNodes returns the items of a list property, by the one slug each names. An item naming no slug or several
// is left out, and so is a property that isn't a list.
func linkNodes(value *yaml.Node) map[string]*yaml.Node {
	nodes := map[string]*yaml.Node{}
	if value == nil || value.Kind != yaml.SequenceNode {
		return nodes
	}
	for _, item := range value.Content {
		links := propertyLinks(item)
		if len(links) == 1 {
			nodes[links[0]] = item
		}
	}
	return nodes
}

func checkDependenciesKept(found ticketNote, ids []string) error {
	var read []string
	for _, property := range slices.Sorted(maps.Keys(found.mapping.Fields)) {
		if found.mapping.Fields[property] == ticket.FieldDependencies {
			read = append(read, propertyLinks(found.note.Property(property))...)
		}
	}
	if !slices.Equal(read, ids) {
		return notKept(found.note.Path, ticket.FieldDependencies)
	}
	return nil
}
