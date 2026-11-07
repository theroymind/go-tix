// Package explorer provides the sketch explorer, a Bubble Tea model that shows the repo tree with each note as a +
// ghost node or a ~ on the file it plans, and a preview of the selected node beside it.
package explorer

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"path"
	"path/filepath"
	"strings"

	"charm.land/bubbles/v2/key"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/x/ansi"

	"github.com/theroymind/go-tix/internal/sketch/tree"
)

// Model is the explorer's state: the tree, which folders are open, whether only notes show, and the cursor.
type Model struct {
	root      string
	tree      *tree.Node
	expanded  map[string]bool
	notesOnly bool
	rows      []tree.Row
	cursor    int
	offset    int    // the first row the tree pane shows
	toggledAt string // the path selected when n was last pressed, until the cursor moves
	width     int
	height    int
	styles    styles
	preview   []string
}

// New returns the explorer for the tree of the repo at root, with every folder above a note open so that every
// note shows.
func New(root string, top *tree.Node) Model {
	model := Model{root: root, tree: top, expanded: tree.NoteFolders(top), styles: newStyles(true)}
	model.rows = tree.Flatten(top, model.expanded, model.notesOnly)
	return model
}

// Init asks the terminal for its background color, to pick colors that read on it.
func (model Model) Init() tea.Cmd {
	return tea.RequestBackgroundColor
}

func (model Model) Update(message tea.Msg) (tea.Model, tea.Cmd) {
	switch message := message.(type) {
	case tea.BackgroundColorMsg:
		model.styles = newStyles(message.IsDark())
	case tea.WindowSizeMsg:
		model.width, model.height = message.Width, message.Height
		model.settle()
	case tea.KeyPressMsg:
		switch {
		case key.Matches(message, keys.Quit):
			return model, tea.Quit
		case key.Matches(message, keys.Down):
			model.moveTo(model.cursor + 1)
		case key.Matches(message, keys.Up):
			model.moveTo(model.cursor - 1)
		case key.Matches(message, keys.Expand):
			model.expand()
		case key.Matches(message, keys.Collapse):
			model.collapse()
		case key.Matches(message, keys.NotesOnly):
			model.toggleNotesOnly()
		}
	}
	return model, nil
}

func (model *Model) moveTo(row int) {
	model.cursor = row
	model.toggledAt = ""
	model.settle()
}

// expand opens the selected folder, or steps into it when it's already open.
func (model *Model) expand() {
	node := model.selected()
	if node == nil || !node.IsDir {
		return
	}
	if !model.expanded[node.Path] {
		model.expanded[node.Path] = true
		model.refreshRows()
		model.toggledAt = ""
		model.settle()
		return
	}
	next := model.cursor + 1
	if next < len(model.rows) && model.rows[next].Depth > model.rows[model.cursor].Depth {
		model.moveTo(next)
	}
}

// collapse closes the selected folder, or goes to the folder above when it's a file or already closed.
func (model *Model) collapse() {
	node := model.selected()
	if node == nil {
		return
	}
	if node.IsDir && model.expanded[node.Path] {
		delete(model.expanded, node.Path)
		model.refreshRows()
		model.toggledAt = ""
		model.settle()
		return
	}
	if parent := model.rowOf(path.Dir(node.Path)); parent >= 0 {
		model.moveTo(parent)
	}
}

// toggleNotesOnly switches between the whole tree and notes only. The cursor stays on the selected node, or on the
// nearest folder above it that still shows, and toggling back before moving returns to the node it started on.
func (model *Model) toggleNotesOnly() {
	target := model.toggledAt
	if target == "" && model.selected() != nil {
		target = model.selected().Path
	}
	model.notesOnly = !model.notesOnly
	model.refreshRows()
	model.cursor = model.nearestRow(target)
	model.toggledAt = target
	model.settle()
}

func (model *Model) refreshRows() {
	model.rows = tree.Flatten(model.tree, model.expanded, model.notesOnly)
}

// settle keeps the cursor on a row and in view, and loads the preview of the row it's on.
func (model *Model) settle() {
	model.cursor = max(0, min(model.cursor, len(model.rows)-1))
	height := model.paneHeight()
	if model.cursor < model.offset {
		model.offset = model.cursor
	}
	if model.cursor >= model.offset+height {
		model.offset = model.cursor - height + 1
	}
	model.preview = previewLines(model.root, model.selected(), height)
}

func (model Model) selected() *tree.Node {
	if len(model.rows) == 0 {
		return nil
	}
	return model.rows[model.cursor].Node
}

// rowOf returns the row showing the node at nodePath, or -1 when none does.
func (model Model) rowOf(nodePath string) int {
	for index, row := range model.rows {
		if row.Node.Path == nodePath {
			return index
		}
	}
	return -1
}

// nearestRow returns the row of the node at nodePath, or of the nearest folder above it that shows, or the first
// row.
func (model Model) nearestRow(nodePath string) int {
	for nodePath != "." && nodePath != "" {
		if row := model.rowOf(nodePath); row >= 0 {
			return row
		}
		nodePath = path.Dir(nodePath)
	}
	return 0
}

func (model Model) paneHeight() int {
	return max(1, model.height-chromeHeight)
}

func (model Model) View() tea.View {
	if model.width == 0 {
		return tea.NewView("")
	}
	height := model.paneHeight()
	treeWidth := min(model.width, max(minTreeWidth, model.width/treeShare))
	previewWidth := max(0, model.width-treeWidth-1-previewGap)

	var treeLines []string
	for index := model.offset; index < len(model.rows) && index < model.offset+height; index++ {
		treeLines = append(treeLines, model.renderRow(index, treeWidth))
	}
	treePane := lipgloss.NewStyle().Width(treeWidth).Height(height).Render(strings.Join(treeLines, "\n"))

	var previewText []string
	for _, line := range model.preview {
		previewText = append(previewText, ansi.Truncate(line, previewWidth, ""))
	}
	previewPane := model.styles.preview.Height(height).Render(strings.Join(previewText, "\n"))

	body := lipgloss.JoinHorizontal(lipgloss.Top, treePane, previewPane)
	view := tea.NewView(strings.Join([]string{model.header(), body, model.footer()}, "\n"))
	view.AltScreen = true
	return view
}

// header names the repo and counts its notes and open questions.
func (model Model) header() string {
	text := fmt.Sprintf("%s  %d notes · %d open questions", filepath.Base(model.root), model.tree.NotesBelow,
		model.tree.OpenBelow)
	if model.notesOnly {
		text += "  [notes only]"
	}
	return model.styles.header.Render(ansi.Truncate(text, model.width, ""))
}

// footer lists the keys.
func (model Model) footer() string {
	var hints []string
	for _, binding := range []key.Binding{keys.Down, keys.Up, keys.Expand, keys.Collapse, keys.NotesOnly, keys.Quit} {
		hints = append(hints, binding.Help().Key+" "+binding.Help().Desc)
	}
	return model.styles.footer.Render(ansi.Truncate(strings.Join(hints, hintSeparator), model.width, ""))
}

// renderRow draws one tree row: its indent, an arrow on a folder, the note's + or ~, the name, and how many open
// questions the note asks. The cursor row is drawn plain and reversed, the full width of the pane.
func (model Model) renderRow(index, width int) string {
	row := model.rows[index]
	node := row.Node
	icon := noFolder
	if node.IsDir {
		icon = closedFolder
		if model.expanded[node.Path] {
			icon = openFolder
		}
	}
	mark, name, questions := noMark, node.Name, ""
	if node.Note != nil {
		mark = node.Note.Mark.Symbol() + " "
	}
	if node.IsDir {
		name += folderSuffix
	}
	if node.OpenQuestions > 0 {
		questions = fmt.Sprintf(questionBadge, node.OpenQuestions)
	}
	lead := strings.Repeat(indent, row.Depth) + icon

	if index == model.cursor {
		plain := ansi.Truncate(lead+mark+name+questions, width, "")
		return model.styles.cursor.Render(plain + strings.Repeat(" ", max(0, width-ansi.StringWidth(plain))))
	}
	switch {
	case node.Note != nil && node.Ghost:
		mark = model.styles.added.Render(mark)
	case node.Note != nil:
		mark = model.styles.changed.Render(mark)
	}
	switch {
	case node.Ghost:
		name = model.styles.ghost.Render(name)
	case node.IsDir:
		name = model.styles.folder.Render(name)
	}
	return ansi.Truncate(lead+mark+name+model.styles.question.Render(questions), width, "")
}

// previewLines returns up to limit lines describing node: its note's text when it has one, a count of the notes
// below a folder, or a file's first lines.
func previewLines(root string, node *tree.Node, limit int) []string {
	switch {
	case node == nil:
		return nil
	case node.Note != nil:
		return readHead(filepath.Join(root, filepath.FromSlash(node.Note.Path)), limit)
	case node.IsDir && node.NotesBelow == 0:
		return []string{"no notes below"}
	case node.IsDir:
		return []string{fmt.Sprintf("%d notes below, %d open questions", node.NotesBelow, node.OpenBelow)}
	}
	return readHead(filepath.Join(root, filepath.FromSlash(node.Path)), limit)
}

// readHead returns the first limit lines of the file at filePath, with escape codes stripped and tabs expanded so
// the file can't upset the terminal, or one line saying it's binary or couldn't be read.
func readHead(filePath string, limit int) []string {
	file, err := os.Open(filePath)
	if err != nil {
		return []string{err.Error()}
	}
	defer func() { _ = file.Close() }()
	head, err := io.ReadAll(io.LimitReader(file, previewMaxLen))
	if err != nil {
		return []string{err.Error()}
	}
	if bytes.IndexByte(head, nullByte) >= 0 {
		return []string{binaryFile}
	}
	lines := strings.Split(string(head), "\n")
	lines = lines[:min(limit, len(lines))]
	for index, line := range lines {
		line = strings.ReplaceAll(strings.TrimRight(line, "\r"), "\t", indent+indent)
		lines[index] = ansi.Strip(line)
	}
	return lines
}
