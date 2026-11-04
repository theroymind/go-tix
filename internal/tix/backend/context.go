package backend

// ContextNote is one ticket or doc as tix context walks it. Type is the note's own type, Links the ids it points
// to through its parent, dependencies and wikilinks, and Backlinks the ids of the notes whose wikilinks point to it.
// A note that StopsWalk is reached but not walked through, unless the walk starts there.
type ContextNote struct {
	ID        string
	Type      string
	Title     string
	Summary   string
	Commits   []string
	Files     []string
	Links     []string
	Backlinks []string
	StopsWalk bool
}

// ContextReader is implemented by a backend whose notes carry summaries, commits, files and wikilinks for tix
// context. A backend without it is walked over its tickets' parents and dependencies alone.
type ContextReader interface {
	// ContextNotes returns every ticket and doc, in id order.
	ContextNotes() ([]ContextNote, error)
}
