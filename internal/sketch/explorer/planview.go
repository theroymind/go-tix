package explorer

import (
	"fmt"
	"slices"
	"strings"

	"charm.land/bubbles/v2/key"
	tea "charm.land/bubbletea/v2"
	"github.com/charmbracelet/x/ansi"

	"github.com/theroymind/go-tix/contracts/sketch"
	"github.com/theroymind/go-tix/contracts/ticket"
	"github.com/theroymind/go-tix/internal/sketch/plan"
	"github.com/theroymind/go-tix/internal/sketch/refs"
	"github.com/theroymind/go-tix/internal/sketch/tree"
)

// LookUpFeature returns the feature's ticket from tix, or nil when tix doesn't track the repo.
type LookUpFeature func(feature string) (*ticket.Ticket, error)

// planFeatureMsg brings the feature's ticket, looked up for the plan view.
type planFeatureMsg struct {
	feature string
	ticket  *ticket.Ticket
	err     error
}

// planView is the plan view's state: the notes that show in build order, the one selected, and the text beside
// them, which is the feature's title and criteria followed by each note's heading and body.
type planView struct {
	active   bool
	notes    []*tree.Node // the nodes of the notes, in build order
	cursor   int
	feature  *ticket.Ticket
	lines    []string // the text of the right pane
	headings []int    // the line in lines of each note's heading
	scroll   int      // the first line of lines that shows
}

// tixFeature returns the LookUpFeature that asks the tix resolver, and finds nothing outside a tix repo.
func tixFeature(resolver refs.Resolver) LookUpFeature {
	return func(feature string) (*ticket.Ticket, error) {
		if !resolver.IsTixRepo() {
			return nil, nil
		}
		value, err := resolver.Ticket(feature)
		if err != nil {
			return nil, err
		}
		return &value, nil
	}
}

// enterPlan swaps the tree for the plan view on the notes that show, selecting the tree's selected note when it
// is one of them, and returns the command that looks up the feature filter's ticket, if there is a filter.
func (model *Model) enterPlan() tea.Cmd {
	notes := planNotes(model.tree)
	if len(notes) == 0 {
		model.status = noNotesToPlan
		return nil
	}
	model.plan = planView{active: true, notes: notes}
	model.plan.layOut()
	model.plan.selectTarget(model.selectedPath())
	if model.feature == "" {
		return nil
	}
	feature, lookUp := model.feature, model.lookUpFeature
	return func() tea.Msg {
		value, err := lookUp(feature)
		return planFeatureMsg{feature: feature, ticket: value, err: err}
	}
}

// refreshPlan shows the notes that show now in the plan view, keeping the selected note when it's still there.
// It leaves the plan view when no notes are left, and does nothing outside it.
func (model *Model) refreshPlan() {
	if !model.plan.active {
		return
	}
	keep := model.plan.selected().Path
	model.plan.notes = planNotes(model.tree)
	if len(model.plan.notes) == 0 {
		model.plan.active = false
		return
	}
	model.plan.layOut()
	model.plan.selectTarget(keep)
}

// showPlanFeature puts the feature's ticket above the notes, when it's for the feature filter the plan view shows.
func (model *Model) showPlanFeature(message planFeatureMsg) {
	if !model.plan.active || message.feature != model.feature {
		return
	}
	if message.err != nil {
		model.status = message.err.Error()
		return
	}
	model.plan.feature = message.ticket
	model.plan.layOut()
	model.plan.moveTo(model.plan.cursor)
}

// updatePlan handles a key in the plan view: v or esc goes back to the tree, on the node it was on before.
func (model *Model) updatePlan(message tea.KeyPressMsg) tea.Cmd {
	half := max(1, model.paneHeight()/2)
	switch {
	case key.Matches(message, keys.Quit):
		return tea.Quit
	case key.Matches(message, keys.Plan), key.Matches(message, keys.Back):
		model.plan.active = false
	case key.Matches(message, keys.Down):
		model.plan.moveTo(model.plan.cursor + 1)
	case key.Matches(message, keys.Up):
		model.plan.moveTo(model.plan.cursor - 1)
	case key.Matches(message, keys.ScrollDown):
		model.plan.scrollTo(model.plan.scroll + half)
	case key.Matches(message, keys.ScrollUp):
		model.plan.scrollTo(model.plan.scroll - half)
	case key.Matches(message, keys.Help):
		model.help = true
	}
	return nil
}

// planNotes returns the nodes of the notes in top that read, in plan.Order's build order.
func planNotes(top *tree.Node) []*tree.Node {
	byTarget := map[string]*tree.Node{}
	var notes []sketch.Note
	for _, node := range nodes(top) {
		if node.Note != nil && node.Problem == nil {
			byTarget[node.Note.Target] = node
			notes = append(notes, *node.Note)
		}
	}
	ordered := make([]*tree.Node, 0, len(notes))
	for _, value := range plan.Order(notes) {
		ordered = append(ordered, byTarget[value.Target])
	}
	return ordered
}

func (view planView) selected() *tree.Node {
	return view.notes[view.cursor]
}

// selectTarget selects the note planning target, or the first note when none does.
func (view *planView) selectTarget(target string) {
	index := 0
	for each, node := range view.notes {
		if node.Path == target {
			index = each
		}
	}
	view.moveTo(index)
}

// moveTo selects the note at index, kept in the list, and scrolls its heading to the top of the right pane.
func (view *planView) moveTo(index int) {
	view.cursor = max(0, min(index, len(view.notes)-1))
	view.scroll = view.headings[view.cursor]
}

// scrollTo scrolls the right pane to line, kept in the text.
func (view *planView) scrollTo(line int) {
	view.scroll = max(0, min(line, len(view.lines)-1))
}

// layOut writes the right pane's text from plan.Parts, as sketch plan writes its brief: the feature's title and
// criteria when tix gave them, then each note's heading, its mark and path, and its body, then the decisions and
// open questions, with a rule between each part.
func (view *planView) layOut() {
	view.lines, view.headings = nil, nil
	var notes []sketch.Note
	for _, node := range view.notes {
		notes = append(notes, *node.Note)
	}
	brief := plan.Parts(view.feature, notes)
	if view.feature != nil {
		view.lines = append(view.lines, view.feature.Title, "")
		view.lines = append(view.lines, brief.Criteria...)
		view.lines = append(view.lines, "", planRule)
	}
	// view.notes is in plan.Order's build order already, as brief.Notes is.
	for index, node := range view.notes {
		if index > 0 {
			view.lines = append(view.lines, "", planRule)
		}
		view.headings = append(view.headings, len(view.lines))
		view.lines = append(view.lines, node.Note.Mark.Symbol()+" "+node.Note.Target, "")
		for _, line := range strings.Split(plan.NoteBody(*node.Note), "\n") {
			view.lines = append(view.lines, ansi.Strip(strings.ReplaceAll(line, "\t", indent+indent)))
		}
	}
	view.addSection(plan.DecisionsTitle, brief.Decisions)
	view.addSection(plan.OpenTitle, brief.Open)
}

// addSection adds lines under a rule and the title, or nothing when there are none.
func (view *planView) addSection(title string, lines []string) {
	if len(lines) == 0 {
		return
	}
	view.lines = append(view.lines, "", planRule, title, "")
	for _, line := range lines {
		view.lines = append(view.lines, ansi.Strip(line))
	}
}

// planRows draws the rows of the note list that show in height, keeping the selected one in view.
func (model Model) planRows(height, width int) []string {
	first := max(0, model.plan.cursor-height+1)
	var rows []string
	for index := first; index < len(model.plan.notes) && index < first+height; index++ {
		rows = append(rows, model.planRow(index, width))
	}
	return rows
}

// planRow draws one note of the list: its + or ~, its path, styled as the tree styles it, and how many links it
// has to other files and from other notes. The selected note is drawn plain and reversed, the full width of the
// pane.
func (model Model) planRow(index, width int) string {
	node := model.plan.notes[index]
	mark := node.Note.Mark.Symbol() + " "
	counts := fmt.Sprintf(linkCounts, len(node.Note.Edges), len(model.linkIndex.usedBy[node.Path]))
	if index == model.plan.cursor {
		plain := ansi.Truncate(mark+node.Path+counts, width, "")
		return model.styles.cursor.Render(plain + strings.Repeat(" ", max(0, width-ansi.StringWidth(plain))))
	}
	styledMark, name := model.styledMarkAndName(node, mark, node.Path)
	return ansi.Truncate(styledMark+name+model.styles.footer.Render(counts), width, "")
}

// planLines returns the lines of the right pane that show in height, from the scroll, with the notes' headings in
// bold.
func (model Model) planLines(height int) []string {
	view := model.plan
	var lines []string
	for index := view.scroll; index < len(view.lines) && index < view.scroll+height; index++ {
		line := view.lines[index]
		if slices.Contains(view.headings, index) {
			line = model.styles.header.Render(line)
		}
		lines = append(lines, line)
	}
	return lines
}
