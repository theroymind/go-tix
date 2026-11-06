package sketch

const (
	FieldSchemaVersion = "schema_version"
	FieldTasks         = "tasks"
	FieldName          = "name"
	FieldFiles         = "files"
	FieldPath          = "path"
	FieldItems         = "items"
	FieldType          = "type"
	FieldLines         = "lines"
	FieldText          = "text"
	FieldAnswer        = "answer"
)

const (
	ItemTypeType      ItemType = "type"
	ItemTypeFn        ItemType = "fn"
	ItemTypeComponent ItemType = "component"
	ItemTypeHook      ItemType = "hook"
	ItemTypeFlow      ItemType = "flow"
	ItemTypeTune      ItemType = "tune"
	ItemTypeRule      ItemType = "rule"
	ItemTypeTest      ItemType = "test"
)

const (
	LineTypeField LineType = "field"
	LineTypeIn    LineType = "in"
	LineTypeOut   LineType = "out"
	LineTypeProp  LineType = "prop"
	LineTypeCase  LineType = "case"
	LineTypeAsk   LineType = "ask"
	LineTypeRule  LineType = "rule"
)

// SchemaVersion is the only sketch schema version this package reads and writes.
const SchemaVersion = 1

var (
	// ItemTypes are the kinds of item a file can hold, in the order the builder offers them.
	ItemTypes = []ItemType{ItemTypeType, ItemTypeFn, ItemTypeComponent, ItemTypeHook, ItemTypeFlow, ItemTypeTune,
		ItemTypeRule, ItemTypeTest}
	// LineTypes are the kinds of line an item can hold, in the order the builder offers them.
	LineTypes = []LineType{LineTypeField, LineTypeIn, LineTypeOut, LineTypeProp, LineTypeCase, LineTypeAsk,
		LineTypeRule}
)

const (
	problemRequired       = "is required"
	problemUnknownVersion = "must be 1"
	problemDuplicateTask  = "duplicates an earlier task name"
	problemDuplicateFile  = "duplicates an earlier file path in this task"
	problemDuplicateItem  = "duplicates an earlier item name in this file"
	problemAnswerNotAsk   = "only an ask line holds an answer"
	problemRepoPath       = "must be a clean path relative to the repo root, inside it"
	fieldIndexFormat      = "%s.%d"
	parentDirectory       = ".."
)
