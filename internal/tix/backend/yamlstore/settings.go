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
