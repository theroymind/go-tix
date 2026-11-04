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
)
