package store

import (
	"io/fs"

	"github.com/theroymind/railyard/internal/contracts/ticket"
)

const (
	temporaryFileSuffix = ".tmp"
	yamlIndent          = 2
)

const (
	DirectoryPermission fs.FileMode = 0o777
	filePermission      fs.FileMode = 0o666
)

// requiredFields are the keys every tix v1 ticket file carries, though the contract leaves priority and the timestamps optional.
var requiredFields = []string{
	ticket.FieldID, ticket.FieldTitle, ticket.FieldType, ticket.FieldStatus,
	ticket.FieldPriority, ticket.FieldCreatedAt, ticket.FieldUpdatedAt,
}
