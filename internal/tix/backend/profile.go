package backend

import (
	"maps"
	"slices"
	"strconv"
	"strings"

	"github.com/theroymind/railyard/internal/contracts/ticket"
	"github.com/theroymind/railyard/internal/contracts/validation"
)

// Profile describes a vault's conventions: where its notes are, which note types are tickets and which are docs,
// and how a ticket note's properties and statuses map onto a ticket. A note whose type is in neither map is ignored.
// Empty Root, TypeProperty, CriteriaHeading, SummaryProperty and CommitsProperty take the defaults WithDefaults fills
// in. An empty FilesHeading means the vault's notes list no files.
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
	// SummaryProperty names the property holding a note's one-line summary, kept apart from a ticket's description,
	// which is the note's body.
	SummaryProperty string `yaml:"summary_property,omitempty"`
	// CommitsProperty names the list property holding the full hashes of the commits that changed a note's code,
	// newest first.
	CommitsProperty string `yaml:"commits_property,omitempty"`
	// FilesHeading names the section listing the files a note expects to touch.
	FilesHeading string `yaml:"files_heading,omitempty"`
	// ContextStopTypes are the note types, such as a vault's home note, that tix context reaches but doesn't walk
	// through, unless the walk starts there.
	ContextStopTypes []string `yaml:"context_stop_types,omitempty,flow"`
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

// WithDefaults returns the profile with an empty Root, TypeProperty, CriteriaHeading, SummaryProperty or
// CommitsProperty set to its default.
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
	if profile.SummaryProperty == "" {
		profile.SummaryProperty = DefaultSummaryProperty
	}
	if profile.CommitsProperty == "" {
		profile.CommitsProperty = DefaultCommitsProperty
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
	for index, noteType := range profile.ContextStopTypes {
		_, isTicket := profile.Tickets[noteType]
		_, isDoc := profile.Docs[noteType]
		if !isTicket && !isDoc {
			problems.Add(profileFieldContextStopTypes+fieldSeparator+strconv.Itoa(index), problemStopTypeUnknown)
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
		status := mapping.Statuses[noteStatus]
		if !status.IsValid() {
			problems.Add(mappingFieldStatuses+fieldSeparator+noteStatus, validation.InvalidOption(ticket.Statuses))
		}
		if named := ticket.Status(noteStatus); named.IsValid() && named != status {
			problems.Add(mappingFieldStatuses+fieldSeparator+noteStatus,
				problemStatusNamedOther+strconv.Quote(noteStatus)+clashMapsTo+strconv.Quote(string(status)))
		}
	}
	if problem := sharedStatuses(mapping.Statuses); problem != "" {
		problems.Add(mappingFieldStatuses, problem)
	}
	for _, property := range slices.Sorted(maps.Keys(mapping.Fields)) {
		if !slices.Contains(LinkFields, mapping.Fields[property]) {
			problems.Add(mappingFieldFields+fieldSeparator+property, validation.InvalidOption(LinkFields))
		}
	}
	return problems.Err()
}

// sharedStatuses describes the ticket statuses that more than one note status maps to, or is "" when the statuses
// map one to one, so that a ticket status always maps back to a single note status.
func sharedStatuses(statuses map[string]ticket.Status) string {
	byTicketStatus := map[ticket.Status][]string{}
	for _, noteStatus := range slices.Sorted(maps.Keys(statuses)) {
		byTicketStatus[statuses[noteStatus]] = append(byTicketStatus[statuses[noteStatus]], strconv.Quote(noteStatus))
	}
	var clashes []string
	for _, status := range slices.Sorted(maps.Keys(byTicketStatus)) {
		if noteStatuses := byTicketStatus[status]; len(noteStatuses) > 1 {
			clashes = append(clashes, strings.Join(noteStatuses, matchSeparator)+clashMapsTo+strconv.Quote(string(status)))
		}
	}
	if len(clashes) == 0 {
		return ""
	}
	return problemSharedStatus + strings.Join(clashes, clashSeparator)
}
