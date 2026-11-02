package vaultstore

import "github.com/theroymind/railyard/internal/contracts/ticket"

const (
	noteExtension  = ".md"
	hiddenPrefix   = "."
	titleLevel     = 1
	statusProperty = "status"
)

var (
	// timestampFields are written only to a note that already holds them.
	timestampFields = map[string]bool{ticket.FieldCreatedAt: true, ticket.FieldUpdatedAt: true}
	// unheldFields are ticket fields a note doesn't keep as a property of the same name: the slug, the H1, the
	// type property, the body and its criteria, the link fields the profile maps from other properties, and
	// priority and tags, which the profile doesn't map yet.
	unheldFields = map[string]bool{
		ticket.FieldID: true, ticket.FieldTitle: true, ticket.FieldType: true, ticket.FieldDescription: true,
		ticket.FieldAcceptance: true, ticket.FieldDependencies: true, ticket.FieldParent: true,
		ticket.FieldPriority: true, ticket.FieldTags: true,
	}
)
