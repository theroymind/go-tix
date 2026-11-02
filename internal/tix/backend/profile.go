package backend

import (
	"maps"
	"slices"

	"github.com/theroymind/railyard/internal/contracts/ticket"
	"github.com/theroymind/railyard/internal/contracts/validation"
)

// Profile describes a vault's conventions: where its notes are, which note types are tickets and which are docs,
// and how a ticket note's properties and statuses map onto a ticket. A note whose type is in neither map is ignored.
// Empty Root, TypeProperty and CriteriaHeading take the defaults WithDefaults fills in.
type Profile struct {
	// Root is the vault's directory, relative to the directory holding .tix.
	Root         string `yaml:"root,omitempty"`
	TypeProperty string `yaml:"type_property,omitempty"`
	// Tickets maps a note type to the tickets its notes become.
	Tickets map[string]TicketMapping `yaml:"tickets"`
	// Docs maps a note type to the docs its notes become.
	Docs            map[string]DocMapping `yaml:"docs,omitempty"`
	CriteriaHeading string                `yaml:"criteria_heading,omitempty"`
	// TicketTemplates maps a ticket note type to the template, relative to Root, that new notes of the type start from.
	TicketTemplates map[string]string `yaml:"ticket_templates,omitempty"`
}

// TicketMapping is how notes of one type become tickets.
type TicketMapping struct {
	TicketType ticket.Type `yaml:"ticket_type"`
	// Statuses maps a note status to a ticket status. A note with any other status is invalid.
	Statuses map[string]ticket.Status `yaml:"statuses"`
	// Fields maps a note property to the ticket field, one of LinkFields, that its wikilinks or slugs fill.
	Fields map[string]string `yaml:"fields,omitempty"`
}

// DocMapping is how notes of one type become docs.
type DocMapping struct {
	Statuses []string `yaml:"statuses,omitempty"`
	// Template is the path, relative to Root, that new docs of the type start from.
	Template string `yaml:"template,omitempty"`
}

// WithDefaults returns the profile with an empty Root, TypeProperty or CriteriaHeading set to its default.
func (profile Profile) WithDefaults() Profile {
	if profile.Root == "" {
		profile.Root = DefaultVaultRoot
	}
	if profile.TypeProperty == "" {
		profile.TypeProperty = DefaultTypeProperty
	}
	if profile.CriteriaHeading == "" {
		profile.CriteriaHeading = DefaultCriteriaHeading
	}
	return profile
}

// Templates returns the paths, relative to Root, of every template the profile names.
func (profile Profile) Templates() []string {
	var templates []string
	for _, template := range profile.TicketTemplates {
		templates = append(templates, template)
	}
	for _, doc := range profile.Docs {
		if doc.Template != "" {
			templates = append(templates, doc.Template)
		}
	}
	return templates
}

func (profile Profile) Validate() error {
	var problems validation.Errors
	if len(profile.Tickets) == 0 {
		problems.Add(profileFieldTickets, problemNoTicketTypes)
	}
	for _, noteType := range slices.Sorted(maps.Keys(profile.Tickets)) {
		problems.AddNested(profileFieldTickets+fieldSeparator+noteType, profile.Tickets[noteType].Validate())
		if _, isDoc := profile.Docs[noteType]; isDoc {
			problems.Add(profileFieldDocs+fieldSeparator+noteType, problemTicketAndDoc)
		}
	}
	for _, noteType := range slices.Sorted(maps.Keys(profile.TicketTemplates)) {
		if _, isTicket := profile.Tickets[noteType]; !isTicket {
			problems.Add(profileFieldTicketTemplates+fieldSeparator+noteType, problemTemplateNotTicket)
		}
	}
	return problems.Err()
}

func (mapping TicketMapping) Validate() error {
	var problems validation.Errors
	if !mapping.TicketType.IsValid() {
		problems.Add(mappingFieldTicketType, validation.InvalidOption(ticket.Types))
	}
	if len(mapping.Statuses) == 0 {
		problems.Add(mappingFieldStatuses, problemNoStatuses)
	}
	for _, noteStatus := range slices.Sorted(maps.Keys(mapping.Statuses)) {
		if !mapping.Statuses[noteStatus].IsValid() {
			problems.Add(mappingFieldStatuses+fieldSeparator+noteStatus, validation.InvalidOption(ticket.Statuses))
		}
	}
	for _, property := range slices.Sorted(maps.Keys(mapping.Fields)) {
		if !slices.Contains(LinkFields, mapping.Fields[property]) {
			problems.Add(mappingFieldFields+fieldSeparator+property, validation.InvalidOption(LinkFields))
		}
	}
	return problems.Err()
}
