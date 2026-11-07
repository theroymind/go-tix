package sketch

const (
	FieldSchemaVersion = "schema_version"
	FieldPath          = "path"
	FieldTarget        = "target"
	FieldFeature       = "feature"
	FieldStatus        = "status"
	FieldEdges         = "edges"
	FieldQuestions     = "questions"
	FieldTo            = "to"
	FieldLabel         = "label"
	FieldText          = "text"
	FieldLine          = "line"
	FieldMark          = "mark"
)

const (
	MarkNew     Mark = "new"
	MarkChanged Mark = "changed"
)

const (
	StatusDraft Status = "draft"
	StatusReady Status = "ready"
)

// SchemaVersion is the only note schema version this package reads and writes.
const SchemaVersion = 2

// NoteSuffix ends every note's file name: the note for useOrbit.ts is useOrbit.ts.sketch.md, in the same folder.
const NoteSuffix = ".sketch.md"

// Marks are the marks a note or edge can carry.
var Marks = []Mark{MarkNew, MarkChanged}

// Statuses are the statuses a note can give in its front matter.
var Statuses = []Status{StatusDraft, StatusReady}

const (
	problemRequired       = "is required"
	problemUnknownVersion = "must be 2"
	problemControl        = "must not hold a control character, such as a tab"
	problemRepoPath       = "must be a clean path relative to the repo root, inside it"
	problemNotTarget      = "must be the target's path followed by " + NoteSuffix
	problemLine           = "must be a line number from 1"
	symbolNew             = "+"
	symbolChanged         = "~"
	fieldIndexFormat      = "%s.%d"
	parentDirectory       = ".."
)
