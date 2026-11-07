package explorer

import (
	"image/color"

	"charm.land/bubbles/v2/key"
	"charm.land/lipgloss/v2"
)

// keys are the explorer's key bindings, in the order the footer lists them.
var keys = struct {
	Down, Up, Expand, Collapse, NotesOnly, Add, Attach, Edit, Delete, Quit key.Binding
}{
	Down:      key.NewBinding(key.WithKeys("j", "down"), key.WithHelp("j", "down")),
	Up:        key.NewBinding(key.WithKeys("k", "up"), key.WithHelp("k", "up")),
	Expand:    key.NewBinding(key.WithKeys("l", "right", "enter"), key.WithHelp("l", "open")),
	Collapse:  key.NewBinding(key.WithKeys("h", "left"), key.WithHelp("h", "close")),
	NotesOnly: key.NewBinding(key.WithKeys("n"), key.WithHelp("n", "notes only")),
	Add:       key.NewBinding(key.WithKeys("a"), key.WithHelp("a", "add")),
	Attach:    key.NewBinding(key.WithKeys("A"), key.WithHelp("A", "attach")),
	Edit:      key.NewBinding(key.WithKeys("e"), key.WithHelp("e", "edit")),
	Delete:    key.NewBinding(key.WithKeys("d"), key.WithHelp("d", "delete")),
	Quit:      key.NewBinding(key.WithKeys("q", "ctrl+c"), key.WithHelp("q", "quit")),
}

const (
	indent        = "  "
	openFolder    = "▾ "
	closedFolder  = "▸ "
	noFolder      = "  "
	noMark        = "  "
	folderSuffix  = "/"
	questionBadge = " ?%d"
	hintSeparator = " · "

	treeShare     = 3 // the tree pane takes a third of the width
	minTreeWidth  = 24
	chromeHeight  = 2 // the header and footer lines
	previewGap    = 1 // columns between the preview's border and its text
	previewMaxLen = 64 * 1024

	binaryFile = "binary file"
	nullByte   = 0

	rootFolder = "."
	folderMode = 0o755
	fileMode   = 0o644

	// A new note starts with front matter naming the feature filter, when one is set, and then a heading naming
	// the file it plans.
	noteFrontMatter = "---\nfeature: %s\n---\n"
	noteHeading     = "# %s\n\n"

	namePrompt      = "new file in %s/: %s█"
	deletePrompt    = "delete %s? y/n"
	editorFailed    = "editor: %v"
	attachNeedsFile = "A attaches a note to a file; a adds one in a folder"
	noNoteToEdit    = "no note; a or A to add one"
	noNoteToDelete  = "no note to delete"
)

// styles are the colors and text styles the explorer draws with, picked for a light or dark background.
type styles struct {
	header, footer, cursor, ghost, folder, added, changed, question, preview lipgloss.Style
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
		folder:   lipgloss.NewStyle().Foreground(pick("#005fd7", "#5fafff")),
		added:    lipgloss.NewStyle().Foreground(pick("#008700", "#5fd75f")),
		changed:  lipgloss.NewStyle().Foreground(pick("#af8700", "#d7d75f")),
		question: lipgloss.NewStyle().Foreground(pick("#af005f", "#ff87af")),
		preview: lipgloss.NewStyle().Border(lipgloss.NormalBorder(), false, false, false, true).
			BorderForeground(dim).PaddingLeft(previewGap),
	}
}
