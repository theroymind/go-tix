package backend

import "errors"

// Doc is a typed note that isn't a ticket, such as a decision, idea or spec. Kind is the subtype the Profile
// defines, Properties holds the note's other frontmatter, and Links the slugs its wikilinks name.
type Doc struct {
	ID         string         `json:"id"`
	Kind       string         `json:"kind"`
	Title      string         `json:"title"`
	Status     string         `json:"status,omitempty"`
	Properties map[string]any `json:"properties,omitempty"`
	Links      []string       `json:"links,omitempty"`
	Path       string         `json:"path"`
}

// DocStore is implemented by a backend whose store holds docs beside its tickets. Commands type-assert it; a backend
// without docs leaves it out and the doc commands report ErrNoDocs.
type DocStore interface {
	// ResolveDoc returns the doc id that an exact id or a unique prefix refers to.
	ResolveDoc(reference string) (string, error)
	// ListDocs returns the docs of one kind, or of every kind when kind is "".
	ListDocs(kind string) ([]Doc, error)
	ReadDoc(id string) (Doc, error)
	// ReadDocText returns the doc's source text, for tix doc show to print.
	ReadDocText(id string) ([]byte, error)
	// ApplyDoc makes every SetField change to the doc or none of them. Other changes fail with ErrUnsupported.
	ApplyDoc(id string, changes []Change) error
}

// ErrNoDocs reports a doc command run on a backend that doesn't implement DocStore.
var ErrNoDocs = errors.New("the backend has no docs")
