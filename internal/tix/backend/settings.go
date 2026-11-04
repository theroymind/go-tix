package backend

import (
	"io/fs"

	"github.com/theroymind/railyard/internal/contracts/ticket"
)

const (
	configFieldPrefix     = "prefix"
	configFieldBackend    = "backend"
	configFieldVault      = "vault"
	problemPrefixMismatch = "prefix must match "
	problemVaultRequired  = "Invalid input: the vault backend needs a profile"
	temporaryFileSuffix   = ".tmp"
	yamlIndent            = 2
	fieldSeparator        = "."
	clashMapsTo           = " map to "
	clashSeparator        = "; "
)

const (
	profileFieldTickets         = "tickets"
	profileFieldDocs            = "docs"
	profileFieldTicketTemplates = "ticket_templates"
	mappingFieldTicketType      = "ticket_type"
	mappingFieldStatuses        = "statuses"
	mappingFieldFields          = "fields"

	problemNoTicketTypes     = "Invalid input: expected at least one ticket note type"
	problemNoStatuses        = "Invalid input: expected at least one status"
	problemTicketAndDoc      = "Invalid input: a note type can't be both a ticket and a doc"
	problemTemplateNotTicket = "Invalid input: expected a note type listed in tickets"
	problemStatusNamedOther  = "Invalid input: expected a note status named after a ticket status to map to it, but "
	problemSharedStatus      = "Invalid input: expected each note status to map to its own ticket status, but "
)

// Profile defaults, which suit an Obsidian vault kept in docs/ whose notes carry a type property.
const (
	DefaultVaultRoot       = "docs"
	DefaultTypeProperty    = "type"
	DefaultCriteriaHeading = "Acceptance criteria"
	DefaultSummaryProperty = "summary"
	DefaultCommitsProperty = "commits"
)

const (
	DirectoryPermission fs.FileMode = 0o777
	FilePermission      fs.FileMode = 0o666
)

const (
	KindYAML  Kind = "yaml"
	KindVault Kind = "vault"
)

// Kinds are the backends a config may name.
var Kinds = []Kind{KindYAML, KindVault}

// LinkFields are the ticket fields a Profile can fill from a note property holding wikilinks or slugs.
var LinkFields = []string{ticket.FieldDependencies, ticket.FieldParent}
