package vaultstore

import (
	"errors"
	"maps"
	"slices"
	"strings"

	"go.yaml.in/yaml/v3"

	"github.com/theroymind/go-tix/contracts/ticket"
	"github.com/theroymind/go-tix/contracts/timestamp"
	"github.com/theroymind/go-tix/internal/tix/backend"
	"github.com/theroymind/go-tix/internal/tix/vault"
)

// ticketMapping returns how the note becomes a ticket, and whether it does, going by its type property.
func ticketMapping(profile backend.Profile, note *vault.Note) (backend.TicketMapping, bool) {
	mapping, isTicket := profile.Tickets[propertyText(note.Property(profile.TypeProperty))]
	return mapping, isTicket
}

// toTicket maps a ticket note onto a ticket through the profile: the slug is its id, the first H1 its title (else
// the slug), its status goes through the mapping's statuses, linked and text properties fill their fields, and the
// checkboxes under the criteria heading become its acceptance criteria.
func toTicket(profile backend.Profile, mapping backend.TicketMapping, note *vault.Note) (ticket.Ticket, error) {
	noteStatus := propertyText(note.Property(statusProperty))
	status, isMapped := mapping.Statuses[noteStatus]
	if !isMapped {
		return ticket.Ticket{}, unknownStatus(note.Path, noteStatus, slices.Sorted(maps.Keys(mapping.Statuses)))
	}
	title, hasTitle := note.Heading(titleLevel)
	if !hasTitle || strings.TrimSpace(title) == "" {
		title = note.Slug
	}
	createdAt := heldTimestamp(note, ticket.FieldCreatedAt)
	updatedAt := heldTimestamp(note, ticket.FieldUpdatedAt)
	mapped := ticket.Ticket{
		ID: note.Slug, Title: title, Type: mapping.TicketType, Status: status, CreatedAt: createdAt, UpdatedAt: updatedAt,
	}
	mapped.ExternalID = textFieldValue(mapping, note, ticket.FieldExternalID)
	mapped.ExternalURL = textFieldValue(mapping, note, ticket.FieldExternalURL)
	for _, property := range slices.Sorted(maps.Keys(mapping.Fields)) {
		links := propertyLinks(note.Property(property))
		switch mapping.Fields[property] {
		case ticket.FieldDependencies:
			mapped.Dependencies = append(mapped.Dependencies, links...)
		case ticket.FieldParent:
			if len(links) > 1 || (len(links) == 1 && mapped.Parent != "") {
				return ticket.Ticket{}, tooManyParents(note.Path, property, links)
			}
			if len(links) == 1 {
				mapped.Parent = links[0]
			}
		}
	}
	criteria, err := criteria(profile, note)
	if err != nil {
		return ticket.Ticket{}, err
	}
	mapped.Acceptance = criteria
	return mapped, backend.Validate(mapped, note.Path)
}

// criteria returns the checkboxes under the profile's criteria heading, or none when the note lacks the heading.
func criteria(profile backend.Profile, note *vault.Note) ([]ticket.Criterion, error) {
	_, boxes, err := criterionBoxes(profile, note)
	if errors.Is(err, vault.ErrSectionNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	var criteria []ticket.Criterion
	for _, box := range boxes {
		criteria = append(criteria, ticket.Criterion{Text: box.Text, Done: box.Done})
	}
	return criteria, nil
}

// criterionBoxes returns the profile's criteria section and the checkboxes in it, which are the ticket's criteria in
// order. A note without the section fails with vault.ErrSectionNotFound.
func criterionBoxes(profile backend.Profile, note *vault.Note) (vault.Range, []vault.Checkbox, error) {
	section, err := note.Section(profile.CriteriaHeading)
	if err != nil {
		return vault.Range{}, nil, err
	}
	return section, note.Checkboxes(section), nil
}

func textFieldValue(mapping backend.TicketMapping, note *vault.Note, field string) string {
	property, isHeld := textProperty(mapping, field)
	if !isHeld {
		return ""
	}
	value := note.Property(property)
	return strings.TrimSpace(propertyText(value))
}

func textProperty(mapping backend.TicketMapping, field string) (string, bool) {
	if property, isMapped := linkProperty(mapping, field); isMapped {
		return property, true
	}
	isTaken := mapping.Fields[field] != ""
	return field, !isTaken
}

func heldTimestamp(note *vault.Note, key string) timestamp.Timestamp {
	property := note.Property(key)
	held := timestamp.Timestamp(propertyText(property))
	if !held.IsValid() {
		return ""
	}
	return held
}

// propertyText is a scalar property's value, or "" for an absent or non-scalar one.
func propertyText(value *yaml.Node) string {
	if value == nil || value.Kind != yaml.ScalarNode {
		return ""
	}
	return value.Value
}

// propertyLinks returns the slugs a property names, as wikilinks ("[[slug]]") or bare slugs, in a single value or
// a list. YAML reads an unquoted [[slug]] as a nested list, so nested lists count too.
func propertyLinks(value *yaml.Node) []string {
	if value == nil {
		return nil
	}
	if value.Kind != yaml.ScalarNode {
		var links []string
		for _, child := range value.Content {
			links = append(links, propertyLinks(child)...)
		}
		return links
	}
	text := strings.TrimSpace(value.Value)
	if text == "" {
		return nil
	}
	if links := vault.Wikilinks(text); len(links) > 0 {
		return links
	}
	return []string{text}
}
