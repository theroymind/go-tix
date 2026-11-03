package yamlstore

import "github.com/theroymind/railyard/internal/contracts/ticket"

const (
	doubleQuote = `"`
	singleQuote = "'"
	nullTag     = "!!null"
	sequenceTag = "!!seq"

	problemNotList = "Invalid input: expected array"
)

// requiredFields are the keys every tix v1 ticket file carries, though the contract leaves priority and the timestamps optional.
var requiredFields = []string{
	ticket.FieldID, ticket.FieldTitle, ticket.FieldType, ticket.FieldStatus,
	ticket.FieldPriority, ticket.FieldCreatedAt, ticket.FieldUpdatedAt,
}

// fieldOrder is the order tix v1 writes a ticket's fields in.
var fieldOrder = []string{
	ticket.FieldID, ticket.FieldTitle, ticket.FieldType, ticket.FieldStatus, ticket.FieldPriority, ticket.FieldTags,
	ticket.FieldDependencies, ticket.FieldParent, ticket.FieldDescription, ticket.FieldAcceptance,
	ticket.FieldCreatedAt, ticket.FieldUpdatedAt,
}

// flowFields are the lists tix v1 writes in flow style and leaves out when empty.
var flowFields = map[string]bool{ticket.FieldTags: true, ticket.FieldDependencies: true}
