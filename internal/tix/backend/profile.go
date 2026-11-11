package backend

import (
	"maps"
	"path/filepath"
	"slices"
	"strconv"
	"strings"

	"github.com/theroymind/go-tix/contracts/ticket"
	"github.com/theroymind/go-tix/contracts/validation"
)

// Profile describes a vault's conventions: where its notes are, which note types are tickets and which are docs,
// and how a ticket note's properties and statuses map onto a ticket. A note whose type is in neither map is ignored.
// Empty Root, TypeProperty, CriteriaHeading, DescriptionHeading, SummaryProperty and CommitsProperty take the
// defaults WithDefaults fills in. An empty FilesHeading means the vault's notes list no files.
type Profile struct {
	// Root is the vault's directory, relative to the directory holding .tix.
	Root         string `yaml:"root,omitempty"`
	TypeProperty string `yaml:"type_property,omitempty"`
	// Tickets maps a note type to the tickets its notes become.
	Tickets map[string]TicketMapping `yaml:"tickets"`
	// Docs maps a note type to the docs its notes become.
	Docs            map[string]DocMapping `yaml:"docs,omitempty"`
	CriteriaHeading string                `yaml:"criteria_heading,omitempty"`
	// DescriptionHeading names the section a new ticket's description goes in, when its template has one. Without
	// it, the description goes under the note's H1.
	DescriptionHeading string `yaml:"description_heading,omitempty"`
	// TicketTemplates maps a ticket note type to the template, relative to Root, that new notes of the type start from.
	TicketTemplates map[string]string `yaml:"ticket_templates,omitempty"`
	// TicketDirectories maps a ticket note type to the directory, relative to Root, that new notes of the type go in.
	// A type it leaves out goes where most of its notes already are, or in Root when it has none. A directory can
	// hold {slug}, the note's slug, and {parent}, its parent note's directory, so a note can sit beside its parent.
	TicketDirectories map[string]string `yaml:"ticket_directories,omitempty"`
	// UncategorizedDirectory is the directory, relative to Root, that {parent} names for a note with no parent.
	UncategorizedDirectory string `yaml:"uncategorized_directory,omitempty"`
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
	// ListProperties are the note properties, such as a task's scope, that tix ls and tix ready show as columns.
	ListProperties []string `yaml:"list_properties,omitempty,flow"`
	// RequiredSections maps a ticket or doc note type to the headings every note of the type must hold, which tix
	// lint checks.
	RequiredSections map[string][]string `yaml:"required_sections,omitempty"`
}

// TicketMapping is how notes of one type become tickets.
type TicketMapping struct {
	TicketType ticket.Type `yaml:"ticket_type"`
	// Statuses maps a note status to a ticket status. A note with any other status is invalid.
	Statuses map[string]ticket.Status `yaml:"statuses"`
	// Fields maps a note property to the ticket field it fills: one of LinkFields, filled from its wikilinks or
	// slugs, or one of TextFields, filled from its text.
	Fields map[string]string `yaml:"fields,omitempty"`
}

// DocMapping is how notes of one type become docs.
type DocMapping struct {
	Statuses []string `yaml:"statuses,omitempty"`
	// Template is the path, relative to Root, that new docs of the type start from.
	Template string `yaml:"template,omitempty"`
	// Directory is the directory, relative to Root, that new docs of the type go in. Empty means where most of its
	// notes already are.
	Directory string `yaml:"directory,omitempty"`
}

// WithDefaults returns the profile with an empty Root, TypeProperty, CriteriaHeading, DescriptionHeading,
// SummaryProperty or CommitsProperty set to its default.
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
	if profile.DescriptionHeading == "" {
		profile.DescriptionHeading = DefaultDescriptionHeading
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
	for _, noteType := range slices.Sorted(maps.Keys(profile.TicketDirectories)) {
		field := profileFieldTicketDirectories + fieldSeparator + noteType
		if _, isTicket := profile.Tickets[noteType]; !isTicket {
			problems.Add(field, problemTemplateNotTicket)
		}
		if !isLocalPattern(profile.TicketDirectories[noteType]) {
			problems.Add(field, problemDirectoryNotLocal)
		}
	}
	if profile.UncategorizedDirectory != "" && !filepath.IsLocal(profile.UncategorizedDirectory) {
		problems.Add(profileFieldUncategorized, problemDirectoryNotLocal)
	}
	for _, noteType := range slices.Sorted(maps.Keys(profile.Docs)) {
		if !filepath.IsLocal(profile.Docs[noteType].Directory) && profile.Docs[noteType].Directory != "" {
			problems.Add(profileFieldDocs+fieldSeparator+noteType+fieldSeparator+docFieldDirectory, problemDirectoryNotLocal)
		}
	}
	for index, noteType := range profile.ContextStopTypes {
		if !profile.IsNoteType(noteType) {
			problems.Add(profileFieldContextStopTypes+fieldSeparator+strconv.Itoa(index), problemStopTypeUnknown)
		}
	}
	for _, noteType := range slices.Sorted(maps.Keys(profile.RequiredSections)) {
		field := profileFieldRequiredSections + fieldSeparator + noteType
		if !profile.IsNoteType(noteType) {
			problems.Add(field, problemStopTypeUnknown)
		}
		for index, heading := range profile.RequiredSections[noteType] {
			if strings.TrimSpace(heading) == "" {
				problems.Add(field+fieldSeparator+strconv.Itoa(index), problemEmptyHeading)
			}
		}
	}
	return problems.Err()
}

// IsNoteType reports whether the profile maps noteType onto tickets or docs.
func (profile Profile) IsNoteType(noteType string) bool {
	_, isTicket := profile.Tickets[noteType]
	_, isDoc := profile.Docs[noteType]
	return isTicket || isDoc
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
		if !slices.Contains(MappedFields, mapping.Fields[property]) {
			problems.Add(mappingFieldFields+fieldSeparator+property, validation.InvalidOption(MappedFields))
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

// isLocalPattern reports whether a ticket_directories entry stays inside the vault root whatever its placeholders
// hold, and names no placeholder but {slug} and {parent}.
func isLocalPattern(pattern string) bool {
	filled := strings.ReplaceAll(pattern, PlaceholderSlug, placeholderSample)
	filled = strings.ReplaceAll(filled, PlaceholderParent, placeholderSample)
	if strings.ContainsAny(filled, placeholderBraces) {
		return false
	}
	return filepath.IsLocal(filled)
}
