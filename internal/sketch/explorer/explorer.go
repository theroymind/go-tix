// Package explorer provides the sketch explorer, a Bubble Tea model that shows the repo tree with each note as a +
// ghost node or a ~ on the file it plans and a preview of the selected node beside it, which edits the note when
// it opens, or inside nvim the selected note in nvim's main window, that adds, opens and deletes notes, marks them
// ready or draft, links a note to a file and shows its links both ways, filters the tree to one feature, kept per
// branch, jumps between open questions and answers them, shows the notes joined in build order, and lists its
// keys.
package explorer

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"path"
	"path/filepath"
	"strings"
	"time"

	"charm.land/bubbles/v2/key"
	"charm.land/bubbles/v2/textarea"
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

// Follow returns the command that shows the file at filePath in the paired nvim, leaving focus in sketch.
type Follow func(filePath string) tea.Cmd

// editedMsg says the editor a note was opened in has returned.
type editedMsg struct{ err error }

// followMsg says the cursor was on the note at notePath a moment ago; the paired nvim shows it if it still is.
type followMsg struct{ notePath string }

// followDelay is how long the cursor rests on a note before the paired nvim shows it, so holding j doesn't run nvim
// for every row passed.
const followDelay = 100 * time.Millisecond

// Model is the explorer's state: the tree, which folders are open, whether only notes show, the cursor, the picker
// or help in the preview pane, and the prompt or status line in the footer. Paired with an nvim, the selected note
// shows there instead of in the preview pane.
type Model struct {
	root          string
	open          Open                     // edits a note: in the preview pane, or inside nvim in nvim
	external      Open                     // edits a note in $EDITOR, or inside nvim in nvim
	pad           pad                      // the note being edited in the preview pane
	paired        bool                     // inside an nvim, which shows the selected note
	follow        Follow                   // shows a note in the paired nvim
	followed      string                   // the note the paired nvim was last asked to show
	listFeatures  func() ([]string, error) // tix's features
	lookUpFeature LookUpFeature            // the feature filter's ticket, for the plan view
	feature       string                   // the feature filter; a new note links to it when set
	featureFile   string                   // the file that keeps the feature filter for the branch, or "" outside git
	loaded        *tree.Node               // the tree as repo.Load read it
	linkIndex     linkIndex                // of loaded, for the links line
	problems      []error                  // the notes that didn't read then
	tree          *tree.Node               // the tree that shows: loaded, with only the feature's notes
	readiness     readiness                // of the notes in tree
	expanded      map[string]bool
	notesOnly     bool
	rows          []tree.Row
	cursor        int
	offset        int    // the first row the tree pane shows
	toggledAt     string // the path selected when n was last pressed, until the cursor moves
	width         int
	height        int
	styles        styles
	isDark        bool // the terminal's background is dark
	preview       []string
	picker        picker
	help          bool // the help overlay shows in the preview pane
	plan          planView
	prompt        prompt
	status        string // a message for the footer, until the next key
}

// New returns the explorer for the repo at root, with the tree repo.Load reads, the feature filter the branch
// last had, every folder above a note open so that every note shows, and only notes showing when there are any.
// Notes open in the preview pane, or in $EDITOR on E; inside nvim the explorer pairs with it, and notes open
// there.
func New(root string) (Model, error) {
	loaded, err := repo.Load(root)
	if err != nil {
		return Model{}, err
	}
	resolver := refs.New(root)
	model := Model{root: root, open: paneOpen, external: editor.Open, paired: editor.Paired(), follow: editor.Follow,
		listFeatures: resolver.Features, lookUpFeature: tixFeature(resolver),
		expanded: tree.NoteFolders(loaded.Tree), styles: newStyles(true), isDark: true}
	if model.paired {
		model.open = editor.Open
	}
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

// Update handles message and, when paired and the cursor has moved onto a note the nvim isn't showing, asks for a
// followMsg once the cursor has rested there. Off a note, it forgets the note the nvim was showing, since the user
// may open another file there before coming back.
func (model Model) Update(message tea.Msg) (tea.Model, tea.Cmd) {
	updated, command := model.update(message)
	if updated.followTarget() == "" {
		updated.followed = ""
	}
	return updated, tea.Batch(command, updated.followLater())
}

func (model Model) update(message tea.Msg) (Model, tea.Cmd) {
	switch message := message.(type) {
	case tea.BackgroundColorMsg:
		model.isDark = message.IsDark()
		model.styles = newStyles(model.isDark)
		model.pad.area.SetStyles(textarea.DefaultStyles(model.isDark))
	case tea.WindowSizeMsg:
		model.width, model.height = message.Width, message.Height
		model.settle()
	case editedMsg:
		model.reload(model.selectedPath())
		if message.err != nil {
			model.status = fmt.Sprintf(editorFailed, message.err)
		}
	case followMsg:
		return model, model.followNow(message.notePath)
	case tea.FocusMsg:
		// Coming back from an editor that returned at once, such as nvim --remote, the notes may have changed.
		model.reload(model.selectedPath())
		model.refreshPlan()
	case planFeatureMsg:
		model.showPlanFeature(message)
	case padMsg:
		return model, model.startPad(message)
	case tea.PasteMsg:
		if model.pad.active() {
			return model, model.updatePad(message)
		}
		model.paste(message.Content)
	case tea.KeyPressMsg:
		model.status = ""
		switch {
		case model.pad.active():
			return model, model.updatePad(message)
		case model.help:
			model.help = false
		case model.picker.kind != pickNone:
			return model, model.answerPicker(message)
		case model.prompt.kind == promptName:
			return model, model.answerName(message)
		case model.prompt.kind == promptDelete:
			model.answerDelete(message)
		case model.plan.active:
			return model, model.updatePlan(message)
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
		case key.Matches(message, keys.External):
			return model, model.editExternally()
		case key.Matches(message, keys.Delete):
			model.askDelete()
		case key.Matches(message, keys.Ref):
			model.askLinkTarget()
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
		case key.Matches(message, keys.Plan):
			return model, model.enterPlan()
		case key.Matches(message, keys.Reload):
			model.reload(model.selectedPath())
		case key.Matches(message, keys.Help):
			model.help = true
		}
	default:
		// The pad's cursor blinks on messages of its own.
		if model.pad.active() {
			return model, model.updatePad(message)
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

// editExternally opens the selected node's note in $EDITOR, or creates one for the selected file when it has
// none, and opens that.
func (model *Model) editExternally() tea.Cmd {
	node := model.selected()
	switch {
	case node == nil || node.IsDir:
		model.status = noNoteToEdit
		return nil
	case node.Note == nil && !model.addNote(node.Path):
		return nil
	}
	notePath := node.Path + sketch.NoteSuffix
	model.followed = notePath
	return model.external(model.filePath(notePath), 0, func(err error) tea.Msg { return editedMsg{err: err} })
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
	if !model.addNote(target) {
		return nil
	}
	return model.openNote(target+sketch.NoteSuffix, 0)
}

// addNote writes a new note planning target, unless one is already there, shows it in the tree with the cursor on
// it, and says in the status line which file the new note is and which it plans. It reports whether the note is
// there, putting the error in the status line when it isn't.
func (model *Model) addNote(target string) bool {
	notePath := target + sketch.NoteSuffix
	created, err := writeNew(model.filePath(notePath), noteText(model.feature))
	if err != nil {
		model.status = err.Error()
		return false
	}
	model.openFoldersAbove(target)
	model.reload(target)
	if created && model.status == "" {
		model.status = fmt.Sprintf(createdNote, path.Base(notePath), path.Base(target))
	}
	return true
}

func (model *Model) openFoldersAbove(nodePath string) {
	for folder := path.Dir(nodePath); folder != rootFolder; folder = path.Dir(folder) {
		model.expanded[folder] = true
	}
}

// openNote opens the note at notePath in the editor, on line when it's above 0, and the editor sends editedMsg
// when it returns.
func (model *Model) openNote(notePath string, line int) tea.Cmd {
	model.followed = notePath
	return model.open(model.filePath(notePath), line, func(err error) tea.Msg { return editedMsg{err: err} })
}

// followTarget returns the path of the note the paired nvim should show: the plan view's selected note while it
// shows, otherwise the selected node's note, or "" when there's none.
func (model Model) followTarget() string {
	if model.plan.active {
		return model.plan.selected().Note.Path
	}
	return model.selectedNotePath()
}

// followLater returns a command that sends followMsg for the note to follow after followDelay, when paired and the
// nvim isn't already showing it, and otherwise nil.
func (model Model) followLater() tea.Cmd {
	notePath := model.followTarget()
	if !model.paired || notePath == "" || notePath == model.followed {
		return nil
	}
	return tea.Tick(followDelay, func(time.Time) tea.Msg { return followMsg{notePath: notePath} })
}

// followNow shows the note at notePath in the paired nvim, unless the cursor has moved off it since or the nvim
// shows it already.
func (model *Model) followNow(notePath string) tea.Cmd {
	if notePath != model.followTarget() || notePath == model.followed {
		return nil
	}
	model.followed = notePath
	return model.follow(model.filePath(notePath))
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
	model.linkIndex = indexLinks(loaded.Tree)
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
	model.sizePad()
}

func (model Model) selected() *tree.Node {
	if len(model.rows) == 0 {
		return nil
	}
	return model.rows[model.cursor].Node
}

// selectedNotePath returns the path of the selected node's note, or "" when it has none.
func (model Model) selectedNotePath() string {
	if node := model.selected(); node != nil && node.Note != nil {
		return node.Note.Path
	}
	return ""
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

// View draws the header, the tree with the preview pane beside it, and the footer. Paired, the tree, or the plan
// view's list, takes the whole width, since nvim shows the note, and help or a picker takes its place while it
// shows.
func (model Model) View() tea.View {
	if model.width == 0 {
		return tea.NewView("")
	}
	var body string
	switch {
	case !model.paired:
		pane := model.previewPane(model.previewWidth())
		if model.pad.active() {
			pane = lipgloss.NewStyle().Height(model.paneHeight()).Render(model.pad.area.View())
		}
		body = lipgloss.JoinHorizontal(lipgloss.Top, model.treePane(model.treeWidth()),
			model.styles.preview.Render(pane))
	case model.help || model.picker.kind != pickNone:
		body = model.previewPane(model.width)
	default:
		body = model.treePane(model.width)
	}
	view := tea.NewView(strings.Join([]string{model.header(), body, model.links(), model.footer()}, "\n"))
	view.AltScreen = true
	view.ReportFocus = true
	return view
}

// treeWidth is how many columns the tree pane takes beside the preview pane.
func (model Model) treeWidth() int {
	return min(model.width, max(minTreeWidth, model.width/treeShare))
}

// previewWidth is how many columns of text the preview pane holds, inside its border and gap.
func (model Model) previewWidth() int {
	return model.width - model.treeWidth() - 1 - previewGap
}

// treePane draws the rows in view, or the plan view's file list, width columns wide.
func (model Model) treePane(width int) string {
	height := model.paneHeight()
	var lines []string
	for index := model.offset; index < len(model.rows) && index < model.offset+height; index++ {
		lines = append(lines, model.renderRow(index, width))
	}
	if model.plan.active {
		lines = model.planRows(height, width)
	}
	return lipgloss.NewStyle().Width(width).Height(height).Render(strings.Join(lines, "\n"))
}

// previewPane draws the help, the picker, the plan view's notes or the selected node's preview, cut to width
// columns.
func (model Model) previewPane(width int) string {
	height := model.paneHeight()
	lines := model.preview
	switch {
	case model.help:
		lines = helpLines()
	case model.picker.kind != pickNone:
		lines = model.picker.lines(height)
	case model.plan.active:
		lines = model.planLines(height)
	}
	var cut []string
	for _, line := range lines {
		cut = append(cut, ansi.Truncate(line, max(0, width), ""))
	}
	return lipgloss.NewStyle().Height(height).Render(strings.Join(cut, "\n"))
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

// links shows the selected node's links, to and from it, or nothing when it has none. In the plan view, the node
// is the selected note's.
func (model Model) links() string {
	node := model.selected()
	if model.plan.active {
		node = model.plan.selected()
	}
	if node == nil {
		return ""
	}
	return ansi.Truncate(linksLine(model.linkIndex, node), model.width, "")
}

// footer shows the prompt and the status message when there is one, and otherwise names the most used keys.
func (model Model) footer() string {
	line := strings.TrimSpace(model.prompt.line() + "  " + model.status)
	if model.pad.discarding {
		line = discardPrompt
	}
	if line != "" {
		return ansi.Truncate(line, model.width, "")
	}
	hints := footerHints
	switch {
	case model.pad.active():
		hints = padFooterHints
	case model.plan.active:
		hints = planFooterHints
	}
	return model.styles.footer.Render(ansi.Truncate(strings.Join(hints, hintSeparator), model.width, ""))
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
	mark, name = model.styledMarkAndName(node, mark, name)
	return ansi.Truncate(lead+mark+name+model.styles.question.Render(questions), width, "")
}

// styledMarkAndName colors node's mark as new, changed or unreadable, and styles its name as draft, planned, a
// ghost or a folder.
func (model Model) styledMarkAndName(node *tree.Node, mark, name string) (string, string) {
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
	return mark, name
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
