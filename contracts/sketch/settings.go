package sketch

const (
	FieldSchemaVersion = "schema_version"
	FieldPath          = "path"
	FieldTarget        = "target"
	FieldFeature       = "feature"
	FieldTask          = "task"
	FieldItems         = "items"
	FieldQuestions     = "questions"
	FieldType          = "type"
	FieldName          = "name"
	FieldText          = "text"
	FieldLine          = "line"
)

const (
	ItemTypeFn        ItemType = "fn"
	ItemTypeType      ItemType = "type"
	ItemTypeComponent ItemType = "component"
	ItemTypeHook      ItemType = "hook"
	ItemTypeFlow      ItemType = "flow"
	ItemTypeTune      ItemType = "tune"
	ItemTypeRule      ItemType = "rule"
	ItemTypeTest      ItemType = "test"
)

// SchemaVersion is the only note schema version this package reads and writes.
const SchemaVersion = 1

// NoteSuffix ends every note's file name: the note for useOrbit.ts is useOrbit.ts.sketch.md, in the same folder.
const NoteSuffix = ".sketch.md"

// ItemTypes are the kinds of item a note can name.
var ItemTypes = []ItemType{ItemTypeFn, ItemTypeType, ItemTypeComponent, ItemTypeHook, ItemTypeFlow, ItemTypeTune,
	ItemTypeRule, ItemTypeTest}

const (
	problemRequired       = "is required"
	problemUnknownVersion = "must be 1"
	problemControl        = "must not hold a control character, such as a tab"
	problemRepoPath       = "must be a clean path relative to the repo root, inside it"
	problemNotTarget      = "must be the target's path followed by " + NoteSuffix
	problemLine           = "must be a line number from 1"
	fieldIndexFormat      = "%s.%d"
	parentDirectory       = ".."
)
