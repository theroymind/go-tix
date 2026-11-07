package explorer

import (
	"image/color"

	"charm.land/bubbles/v2/key"
	"charm.land/lipgloss/v2"
)

// keys are the explorer's key bindings, in the order the help overlay lists them.
var keys = struct {
	Down, Up, Expand, Collapse, Enter, NotesOnly, Add, Edit, Delete, Ref, Feature, Link, Ready, Answer, NextQuestion,
	PreviousQuestion, Plan, Back, ScrollDown, ScrollUp, Reload, Help, Quit key.Binding
}{
	Down:             key.NewBinding(key.WithKeys("j", "down"), key.WithHelp("j ↓", "down")),
	Up:               key.NewBinding(key.WithKeys("k", "up"), key.WithHelp("k ↑", "up")),
	Expand:           key.NewBinding(key.WithKeys("l", "right"), key.WithHelp("l →", "open a folder, or step into it")),
	Collapse:         key.NewBinding(key.WithKeys("h", "left"), key.WithHelp("h ←", "close a folder, or go up to it")),
	Enter:            key.NewBinding(key.WithKeys("enter"), key.WithHelp("enter", "toggle a folder; on a file, edit its note")),
	NotesOnly:        key.NewBinding(key.WithKeys("n"), key.WithHelp("n", "notes only, or the whole tree")),
	Add:              key.NewBinding(key.WithKeys("a"), key.WithHelp("a", "add a note planning a new file")),
	Edit:             key.NewBinding(key.WithKeys("e"), key.WithHelp("e", "edit the note, adding one to a file without")),
	Delete:           key.NewBinding(key.WithKeys("d"), key.WithHelp("d", "delete the note")),
	Ref:              key.NewBinding(key.WithKeys("r"), key.WithHelp("r", "link the note to a file")),
	Feature:          key.NewBinding(key.WithKeys("f"), key.WithHelp("f", "filter to a feature")),
	Link:             key.NewBinding(key.WithKeys("L"), key.WithHelp("L", "link the note to a feature")),
	Ready:            key.NewBinding(key.WithKeys("y"), key.WithHelp("y", "mark the note ready, or draft")),
	Answer:           key.NewBinding(key.WithKeys(">"), key.WithHelp(">", "answer the note's first open question")),
	NextQuestion:     key.NewBinding(key.WithKeys("]"), key.WithHelp("]", "next note with an open question")),
	PreviousQuestion: key.NewBinding(key.WithKeys("["), key.WithHelp("[", "previous note with an open question")),
	Plan:             key.NewBinding(key.WithKeys("v"), key.WithHelp("v", "plan view: the notes joined in build order")),
	Back:             key.NewBinding(key.WithKeys("esc"), key.WithHelp("esc", "in the plan view, back to the tree")),
	ScrollDown:       key.NewBinding(key.WithKeys("ctrl+d", "pgdown"), key.WithHelp("ctrl+d", "in the plan view, scroll down")),
	ScrollUp:         key.NewBinding(key.WithKeys("ctrl+u", "pgup"), key.WithHelp("ctrl+u", "in the plan view, scroll up")),
	Reload:           key.NewBinding(key.WithKeys("R"), key.WithHelp("R", "reload")),
	Help:             key.NewBinding(key.WithKeys("?"), key.WithHelp("?", "help")),
	Quit:             key.NewBinding(key.WithKeys("q", "ctrl+c"), key.WithHelp("q", "quit")),
}

// helpKeys are the bindings the help overlay lists, every one of them.
var helpKeys = []key.Binding{keys.Down, keys.Up, keys.Expand, keys.Collapse, keys.Enter, keys.NotesOnly, keys.Add,
	keys.Edit, keys.Delete, keys.Ref, keys.Feature, keys.Link, keys.Ready, keys.Answer, keys.NextQuestion,
	keys.PreviousQuestion, keys.Plan, keys.Back, keys.ScrollDown, keys.ScrollUp, keys.Reload, keys.Help, keys.Quit}

// footerHints are the few keys the footer names, each with a short name, ending with ? for the rest.
var footerHints = []string{"n notes", "a add", "e edit", "f feature", "y ready", "> answer", "] question", "v plan",
	"? help"}

// planFooterHints are the keys the footer names in the plan view.
var planFooterHints = []string{"j/k note", "ctrl+d/ctrl+u scroll", "v/esc tree", "? help", "q quit"}

const (
	indent        = "  "
	openFolder    = "▾ "
	closedFolder  = "▸ "
	noFolder      = "  "
	noMark        = "  "
	problemMark   = "! "
	folderSuffix  = "/"
	questionBadge = " ?%d"
	hintSeparator = " · "

	treeShare     = 3 // the tree pane takes a third of the width
	minTreeWidth  = 24
	chromeHeight  = 3 // the header, links and footer lines
	previewGap    = 1 // columns between the preview's border and its text
	previewMaxLen = 64 * 1024

	binaryFile = "binary file"
	nullByte   = 0

	rootFolder = "."
	folderMode = 0o755
	fileMode   = 0o644

	// A new note starts with front matter naming the feature filter, when one is set, and marking it draft, and
	// then a heading naming the file it plans.
	noteFence   = "---\n"
	noteFeature = "feature: %s\n"
	noteDraft   = "status: draft\n"
	noteHeading = "# %s\n\n"

	namePrompt     = "plan file: %s/%s█"
	deletePrompt   = "delete %s? y/n"
	editorFailed   = "editor: %v"
	noNoteToEdit   = "no note; a adds one in a folder"
	noNoteToDelete = "no note to delete"

	// r adds a -> line linking the note to the file picked, and the links line shows a note's links both ways.
	linkMarker    = "-> "
	fileTitle     = "link %s to file"
	alreadyLinked = "%s already links to %s"
	refNeedsNote  = "r links a note to a file; a or e to add one"
	usesLabel     = "uses → "
	usedByLabel   = "used by ← "
	danglingMark  = "!"
	linkSeparator = ", "
	linksGap      = "   "

	featureTitle    = "filter to feature"
	linkTitle       = "link %s to feature"
	linkNeedsNote   = "L links a note to a feature; a or e to add one"
	linkedNote      = "linked %s to %s"
	allFeatures     = "all features"
	featureFolder   = "sketch/filter" // under the git folder, a file per branch keeps its feature filter
	noOpenQuestions = "no open questions"
	noneToAnswer    = "> answers a note's open question; this has none"
	answerLine      = "> "

	helpTitle      = "keys (any key closes this)"
	helpLine       = "%-6s %s"
	sketchExt      = ".sketch" // a stray one, as in useOrbit.sketch.ts, comes out of a typed name
	readyNeedsNote = "y marks a note ready; a or e to add one"
	markedNote     = "%s is %s"

	noNotesToPlan = "no notes to plan; a adds one"
	criterionLine = "%d. [%s] %s"
	planRule      = "────────────────────────────────────────"
	linkCounts    = "  →%d ←%d"

	headerCounts    = "%s  %s · %d draft · %d ready · %s · %d dangling"
	unreadableNotes = "%s couldn't be read: %v"
	unreadableBadge = "  %d unreadable"

	pickerFilter = "filter: %s█"
	pickMarker   = "> "
	noPick       = "  "
	noMatches    = "no matches"
)

// styles are the colors and text styles the explorer draws with, picked for a light or dark background.
type styles struct {
	header, footer, cursor, ghost, planned, draft, folder, added, changed, question, preview lipgloss.Style
}

func newStyles(isDark bool) styles {
	lightDark := lipgloss.LightDark(isDark)
	pick := func(light, dark string) color.Color {
		return lightDark(lipgloss.Color(light), lipgloss.Color(dark))
	}
	dim := pick("#8a8a8a", "#6c6c6c")
	return styles{
		header:   lipgloss.NewStyle().Bold(true),
		footer:   lipgloss.NewStyle().Foreground(dim),
		cursor:   lipgloss.NewStyle().Reverse(true),
		ghost:    lipgloss.NewStyle().Foreground(dim).Italic(true),
		planned:  lipgloss.NewStyle().Italic(true),
		draft:    lipgloss.NewStyle().Foreground(dim),
		folder:   lipgloss.NewStyle().Foreground(pick("#005fd7", "#5fafff")),
		added:    lipgloss.NewStyle().Foreground(pick("#008700", "#5fd75f")),
		changed:  lipgloss.NewStyle().Foreground(pick("#af8700", "#d7d75f")),
		question: lipgloss.NewStyle().Foreground(pick("#af005f", "#ff87af")),
		preview: lipgloss.NewStyle().Border(lipgloss.NormalBorder(), false, false, false, true).
			BorderForeground(dim).PaddingLeft(previewGap),
	}
}
