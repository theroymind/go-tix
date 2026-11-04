package index

import (
	"errors"
	"io/fs"
)

const (
	SchemaVersion    = 1
	Directory        = "index"
	FileExtension    = ".json"
	embeddingsPath   = "/embeddings"
	contentTypeJSON  = "application/json"
	headerContent    = "Content-Type"
	embedBatchSize   = 32
	textSeparator    = "\n"
	keywordSeparator = " "
	temporaryFile    = ".tmp"
	minimumTopK      = 1
)

const (
	claudeExecutable       = "claude"
	claudeModelOption      = "--model="
	claudeResultType       = "result"
	jsonObjectStart        = "{"
	jsonObjectEnd          = "}"
	minimumMatchScore      = 1
	maximumMatchScore      = 3
	promptSectionSeparator = "\n\n"
	promptListItemPrefix   = "- "
	promptFieldJoiner      = " | "
	promptMissingSummary   = "(no summary)"
	promptIDLabel          = "id: "
	promptTypeLabel        = "type: "
	promptTitleLabel       = "title: "
	promptSummaryLabel     = "summary: "
	promptQueryLabel       = "Query: "
	promptNotesHeading     = "Notes:\n"
	promptInstruction      = "You search a project's notes. Score each note below for how well it matches the query: " +
		"3 when it is what the query asks for, 2 when it is closely related, 1 when it is loosely related."
	promptAnswerInstruction = "Answer with JSON only, no prose and no code fence, in the shape " +
		`{"matches":[{"id":"<note id>","score":<1 to 3>}]}` + ". Leave out every note that doesn't match at all."
)

var claudeArguments = []string{
	"--print",
	"--output-format=json",
	"--strict-mcp-config",
	"--tools=",
}

const (
	DirectoryPermission fs.FileMode = 0o777
	FilePermission      fs.FileMode = 0o666
)

var (
	ErrNoIndex             = errors.New("no index; run tix index")
	ErrModelMismatch       = errors.New("the index was built with another model; run tix index")
	ErrSchemaVersion       = errors.New("unknown index schema_version")
	ErrEndpointUnreachable = errors.New("can't reach the embeddings endpoint")
	ErrEndpointFailed      = errors.New("the embeddings endpoint failed")
	ErrEmbeddingCount      = errors.New("the embeddings endpoint returned the wrong number of vectors")
	ErrDimensionMismatch   = errors.New("vectors differ in length")
	ErrEmptyVector         = errors.New("vector is empty")
	ErrClaudeFailed        = errors.New("claude failed")
	ErrInvalidAnswer       = errors.New("claude gave an invalid search answer")
)
