// Package explorer provides the sketch explorer, a Bubble Tea model that shows the repo tree with each note as a +
// ghost node or a ~ on the file it plans and a preview of the selected node beside it, that adds, opens and deletes
// notes, marks them ready or draft, adds names from a note's target to it, filters the tree to one feature, kept per
// branch, jumps between open questions and answers them, and lists its keys.
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

	"github.com/theroymind/go-tix/contracts/sketch"
	"github.com/theroymind/go-tix/internal/sketch/editor"
	"github.com/theroymind/go-tix/internal/sketch/refs"
	"github.com/theroymind/go-tix/internal/sketch/repo"
	"github.com/theroymind/go-tix/internal/sketch/tree"
)

// Open returns the command that opens the file at filePath in an editor, on line when it's above 0, and then sends
// done with the editor's error.
type Open func(filePath string, line int, done func(error) tea.Msg) tea.Cmd

// editedMsg says the editor a note was opened in has returned.
type editedMsg struct{ err error }

// Model is the explorer's state: the tree, which folders are open, whether only notes show, the cursor, the picker
// or help in the preview pane, and the prompt or status line in the footer.
type Model struct {
	root         string
	open         Open
	listFeatures func() ([]string, error) // tix's features
	feature      string                   // the feature filter; a new note links to it when set
	featureFile  string                   // the file that keeps the feature filter for the branch, or "" outside git
	loaded       *tree.Node               // the tree as repo.Load read it
	problems     []error                  // the notes that didn't read then
	tree         *tree.Node               // the tree that shows: loaded, with only the feature's notes
	readiness    readiness                // of the notes in tree
	expanded     map[string]bool
	notesOnly    bool
	rows         []tree.Row
	cursor       int
	offset       int    // the first row the tree pane shows
	toggledAt    string // the path selected when n was last pressed, until the cursor moves
	width        int
	height       int
	styles       styles
	preview      []string
	picker       picker
	help         bool // the help overlay shows in the preview pane
	prompt       prompt
	status       string // a message for the footer, until the next key
}

// New returns the explorer for the repo at root, with the tree repo.Load reads, the feature filter the branch
// last had, every folder above a note open so that every note shows, and only notes showing when there are any.
// Notes open in the editor editor.Open picks.
func New(root string) (Model, error) {
	loaded, err := repo.Load(root)
	if err != nil {
		return Model{}, err
	}
	model := Model{root: root, open: editor.Open, listFeatures: refs.New(root).Features,
		expanded: tree.NoteFolders(loaded.Tree), styles: newStyles(true)}
	model.featureFile = featureFile(root)
	model.feature = readFeature(model.featureFile)
	model.show(loaded)
	model.notesOnly = model.tree.NotesBelow > 0
	model.rows = tree.Flatten(model.tree, model.expanded, model.notesOnly)
	return model, nil
}

// Problems returns an error for each note that didn't read when the tree was last loaded.
func (model Model) Problems() []error {
	return model.problems
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
	case editedMsg:
		model.reload(model.selectedPath())
		if message.err != nil {
			model.status = fmt.Sprintf(editorFailed, message.err)
		}
	case tea.FocusMsg:
		// Coming back from an editor that returned at once, such as nvim --remote, the notes may have changed.
		model.reload(model.selectedPath())
	case tea.PasteMsg:
		model.paste(message.Content)
	case tea.KeyPressMsg:
		model.status = ""
		switch {
		case model.help:
			model.help = false
		case model.picker.kind != pickNone:
			return model, model.answerPicker(message)
		case model.prompt.kind == promptName:
			return model, model.answerName(message)
		case model.prompt.kind == promptDelete:
			model.answerDelete(message)
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
		case key.Matches(message, keys.Enter):
			return model, model.enter()
		case key.Matches(message, keys.NotesOnly):
			model.toggleNotesOnly()
		case key.Matches(message, keys.Add):
			return model, model.add()
		case key.Matches(message, keys.Edit):
			return model, model.edit()
		case key.Matches(message, keys.Delete):
			model.askDelete()
		case key.Matches(message, keys.Ref):
			model.askName()
		case key.Matches(message, keys.Feature):
			model.askFeature()
		case key.Matches(message, keys.Link):
			model.askLink()
		case key.Matches(message, keys.Ready):
			model.toggleReady()
		case key.Matches(message, keys.Answer):
			return model, model.answer()
		case key.Matches(message, keys.NextQuestion):
			model.jumpToQuestion(1)
		case key.Matches(message, keys.PreviousQuestion):
			model.jumpToQuestion(-1)
		case key.Matches(message, keys.Reload):
			model.reload(model.selectedPath())
		case key.Matches(message, keys.Help):
			model.help = true
		}
	}
	return model, nil
}

// paste types pasted text, on one line, into the picker's filter or the name prompt.
func (model *Model) paste(text string) {
	text = strings.NewReplacer("\r", "", "\n", "").Replace(text)
	switch {
	case model.picker.kind != pickNone:
		model.picker.typeText(text)
	case model.prompt.kind == promptName:
		model.prompt.typed += text
	}
}

// add asks for the name of a file to plan in the selected folder, or opens the selected file's note, or asks for
// the name of a file to plan in the selected file's folder.
func (model *Model) add() tea.Cmd {
	node := model.selected()
	folder := rootFolder
	switch {
	case node == nil:
	case node.IsDir:
		folder = node.Path
	case node.Note != nil:
		return model.openNote(node.Note.Path, 0)
	default:
		folder = path.Dir(node.Path)
	}
	model.prompt = prompt{kind: promptName, path: folder}
	return nil
}

// enter opens or closes the selected folder, and on a file opens its note, creating it when there's none.
func (model *Model) enter() tea.Cmd {
	if node := model.selected(); node != nil && node.IsDir {
		model.toggle()
		return nil
	}
	return model.edit()
}

// edit opens the selected node's note, or creates one for the selected file when it has none.
func (model *Model) edit() tea.Cmd {
	node := model.selected()
	switch {
	case node != nil && node.Note != nil:
		return model.openNote(node.Note.Path, 0)
	case node == nil || node.IsDir:
		model.status = noNoteToEdit
		return nil
	}
	return model.createNote(node.Path)
}

// askDelete asks before deleting the selected node's note.
func (model *Model) askDelete() {
	node := model.selected()
	if node == nil || node.Note == nil {
		model.status = noNoteToDelete
		return
	}
	model.prompt = prompt{kind: promptDelete, path: node.Note.Path}
}

// createNote writes a new note planning target, unless one is already there, then shows it in the tree and opens
// it.
func (model *Model) createNote(target string) tea.Cmd {
	notePath := target + sketch.NoteSuffix
	if err := writeNew(model.filePath(notePath), noteText(model.feature, target)); err != nil {
		model.status = err.Error()
		return nil
	}
	model.openFoldersAbove(target)
	model.reload(target)
	return model.openNote(notePath, 0)
}

func (model *Model) openFoldersAbove(nodePath string) {
	for folder := path.Dir(nodePath); folder != rootFolder; folder = path.Dir(folder) {
		model.expanded[folder] = true
	}
}

// openNote opens the note at notePath in the editor, on line when it's above 0, and the editor sends editedMsg
// when it returns.
func (model *Model) openNote(notePath string, line int) tea.Cmd {
	return model.open(model.filePath(notePath), line, func(err error) tea.Msg { return editedMsg{err: err} })
}

// reload reads the tree from disk again and puts the cursor on the node at keep, or the nearest folder above it
// that shows.
func (model *Model) reload(keep string) {
	loaded, err := repo.Load(model.root)
	if err != nil {
		model.status = err.Error()
		return
	}
	model.show(loaded)
	model.showRows(keep)
}

// show puts loaded in the tree, with only the feature's notes, and names the first note that didn't read in the
// status line.
func (model *Model) show(loaded repo.Repo) {
	model.loaded = loaded.Tree
	model.problems = loaded.Problems
	model.applyFeature()
	if len(loaded.Problems) > 0 {
		model.status = fmt.Sprintf(unreadableNotes, count(len(loaded.Problems), "note"), loaded.Problems[0])
	}
}

// showRows lists the rows that show again and puts the cursor on the node at keep, or the nearest folder above it
// that shows.
func (model *Model) showRows(keep string) {
	model.refreshRows()
	model.cursor = model.nearestRow(keep)
	model.toggledAt = ""
	model.settle()
}

func (model Model) filePath(notePath string) string {
	return filepath.Join(model.root, filepath.FromSlash(notePath))
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

// toggle opens the selected folder, or closes it when it's already open.
func (model *Model) toggle() {
	node := model.selected()
	if node == nil || !node.IsDir {
		return
	}
	if model.expanded[node.Path] {
		model.collapse()
		return
	}
	model.expand()
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

func (model Model) selectedPath() string {
	if node := model.selected(); node != nil {
		return node.Path
	}
	return rootFolder
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

	previewLines := model.preview
	switch {
	case model.help:
		previewLines = helpLines()
	case model.picker.kind != pickNone:
		previewLines = model.picker.lines(height)
	}
	var previewText []string
	for _, line := range previewLines {
		previewText = append(previewText, ansi.Truncate(line, previewWidth, ""))
	}
	previewPane := model.styles.preview.Height(height).Render(strings.Join(previewText, "\n"))

	body := lipgloss.JoinHorizontal(lipgloss.Top, treePane, previewPane)
	view := tea.NewView(strings.Join([]string{model.header(), body, model.footer()}, "\n"))
	view.AltScreen = true
	view.ReportFocus = true
	return view
}

// header names the repo and counts the notes that show, how many are draft and ready, their open questions and
// their dangling links, and names the feature filter.
func (model Model) header() string {
	text := fmt.Sprintf(headerCounts, filepath.Base(model.root), count(model.tree.NotesBelow, "note"),
		model.readiness.draft, model.readiness.ready, count(model.tree.OpenBelow, "open question"),
		model.readiness.dangling)
	if model.feature != "" {
		text += "  feature: " + model.feature
	}
	if model.notesOnly {
		text += "  [notes only]"
	}
	if len(model.problems) > 0 {
		text += fmt.Sprintf(unreadableBadge, len(model.problems))
	}
	return model.styles.header.Render(ansi.Truncate(text, model.width, ""))
}

// footer shows the prompt and the status message when there is one, and otherwise names the most used keys.
func (model Model) footer() string {
	line := strings.TrimSpace(model.prompt.line() + "  " + model.status)
	if line != "" {
		return ansi.Truncate(line, model.width, "")
	}
	return model.styles.footer.Render(ansi.Truncate(strings.Join(footerHints, hintSeparator), model.width, ""))
}

// helpLines lists every key with what it does, under a title.
func helpLines() []string {
	lines := []string{helpTitle, ""}
	for _, binding := range helpKeys {
		lines = append(lines, fmt.Sprintf(helpLine, binding.Help().Key, binding.Help().Desc))
	}
	return lines
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
	switch {
	case node.Problem != nil:
		mark = problemMark
	case node.Note != nil:
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
	case node.Problem != nil:
		mark = model.styles.question.Render(mark)
	case node.Note != nil && node.Ghost:
		mark = model.styles.added.Render(mark)
	case node.Note != nil:
		mark = model.styles.changed.Render(mark)
	}
	switch {
	case isDraft(node):
		name = model.styles.draft.Italic(node.Ghost).Render(name)
	case node.Note != nil && node.Ghost:
		name = model.styles.planned.Render(name)
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
	case node.Problem != nil:
		lines := readHead(filepath.Join(root, filepath.FromSlash(node.Note.Path)), limit-1)
		return append([]string{node.Problem.Error()}, lines...)
	case node.Note != nil:
		return readHead(filepath.Join(root, filepath.FromSlash(node.Note.Path)), limit)
	case node.IsDir && node.NotesBelow == 0:
		return []string{"no notes below"}
	case node.IsDir:
		return []string{fmt.Sprintf("%s below, %s", count(node.NotesBelow, "note"),
			count(node.OpenBelow, "open question"))}
	}
	return readHead(filepath.Join(root, filepath.FromSlash(node.Path)), limit)
}

// count returns n and the noun, plural unless n is 1, as in "1 note" or "2 notes".
func count(n int, noun string) string {
	if n == 1 {
		return "1 " + noun
	}
	return fmt.Sprintf("%d %ss", n, noun)
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
