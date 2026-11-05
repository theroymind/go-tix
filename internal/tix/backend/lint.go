package backend

// LintNote is one ticket or doc as tix lint checks it: its id, note type and path relative to the directory holding
// .tix, its summary, the text of every heading it holds, the headings its type requires, and every slug it names,
// through wikilinks or link properties such as a parent or dependencies, once each.
type LintNote struct {
	ID               string
	Type             string
	Path             string
	Summary          string
	Headings         []string
	RequiredSections []string
	Links            []string
}

// LintReader is implemented by a backend whose notes tix lint can check.
type LintReader interface {
	// LintNotes returns every ticket and doc note, in id order, without mapping them onto tickets or docs, so that a
	// note that wouldn't map is still checked.
	LintNotes() ([]LintNote, error)
	// LinkTargets returns every name a link can resolve to: each note's slug and every other file's name.
	LinkTargets() ([]string, error)
}
