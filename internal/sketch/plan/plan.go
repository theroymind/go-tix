// Package plan turns a set of sketch notes into a build plan: the notes in build order, a Mermaid flowchart of
// how their files link, and a markdown brief that joins both with the notes' bodies and decisions.
package plan

import (
	"cmp"
	"fmt"
	"maps"
	"path"
	"slices"
	"strings"

	"github.com/theroymind/go-tix/contracts/sketch"
	"github.com/theroymind/go-tix/contracts/ticket"
	"github.com/theroymind/go-tix/internal/sketch/questions"
	"github.com/theroymind/go-tix/internal/sketch/repo"
)

// Order returns the notes in build order: each note after every note it links to. A link counts only when it
// points at another note's target in the set. Notes free to go at the same time go in path order, and the notes of
// a cycle go together, in path order, once every note the cycle links to outside it has gone.
func Order(notes []sketch.Note) []sketch.Note {
	sorted := byPath(notes)
	links := noteLinks(sorted)
	cycles := cyclesOf(links)
	done := make([]bool, len(sorted))
	ordered := make([]sketch.Note, 0, len(sorted))
	for len(ordered) < len(sorted) {
		for index := range sorted {
			if done[index] || !isReady(cycles[index], links, done) {
				continue
			}
			for _, member := range cycles[index] {
				done[member] = true
				ordered = append(ordered, sorted[member])
			}
			break
		}
	}
	return ordered
}

// byPath returns a copy of notes sorted by path.
func byPath(notes []sketch.Note) []sketch.Note {
	sorted := slices.Clone(notes)
	slices.SortFunc(sorted, func(a, b sketch.Note) int { return cmp.Compare(a.Path, b.Path) })
	return sorted
}

// noteLinks returns, for each note, the indexes of the other notes whose targets it links to.
func noteLinks(notes []sketch.Note) [][]int {
	byTarget := map[string]int{}
	for index, value := range notes {
		byTarget[value.Target] = index
	}
	links := make([][]int, len(notes))
	for index, value := range notes {
		for _, edge := range value.Edges {
			linked, isNote := byTarget[edge.To]
			if isNote && linked != index {
				links[index] = append(links[index], linked)
			}
		}
	}
	return links
}

// cyclesOf returns, for each note, the notes on a cycle of links with it, itself included, in index order. A note
// on no cycle is alone in its own.
func cyclesOf(links [][]int) [][]int {
	reaches := make([][]bool, len(links))
	for index := range links {
		reaches[index] = reachable(index, links)
	}
	cycles := make([][]int, len(links))
	for index := range links {
		for other := range links {
			if other == index || (reaches[index][other] && reaches[other][index]) {
				cycles[index] = append(cycles[index], other)
			}
		}
	}
	return cycles
}

// reachable returns which notes a walk over links from start reaches in one step or more.
func reachable(start int, links [][]int) []bool {
	reached := make([]bool, len(links))
	pending := slices.Clone(links[start])
	for len(pending) > 0 {
		next := pending[len(pending)-1]
		pending = pending[:len(pending)-1]
		if reached[next] {
			continue
		}
		reached[next] = true
		pending = append(pending, links[next]...)
	}
	return reached
}

// isReady reports whether every note the cycle links to outside itself is done.
func isReady(cycle []int, links [][]int, done []bool) bool {
	for _, member := range cycle {
		for _, linked := range links[member] {
			if !done[linked] && !slices.Contains(cycle, linked) {
				return false
			}
		}
	}
	return true
}

// Mermaid returns a flowchart LR of the notes: one subgraph per folder holding a node for each note's target,
// styled as new or changed, an arrow for each link, labelled when the link is, and a plain node outside any
// subgraph for each linked file no note in the set plans, styled dangling when it neither exists nor is planned.
// Node ids are n0, n1, ... in path order, then the linked files in path order, so the same notes give the same
// chart.
func Mermaid(notes []sketch.Note) string {
	sorted := byPath(notes)
	ids := map[string]string{}
	folders := map[string][]sketch.Note{}
	for index, value := range sorted {
		ids[value.Target] = fmt.Sprintf("n%d", index)
		folder := path.Dir(value.Target)
		folders[folder] = append(folders[folder], value)
	}
	dangling := map[string]bool{}
	var others []string
	for _, value := range sorted {
		for _, edge := range value.Edges {
			if _, known := ids[edge.To]; known {
				continue
			}
			ids[edge.To] = ""
			others = append(others, edge.To)
			dangling[edge.To] = repo.Dangling(edge)
		}
	}
	slices.Sort(others)
	for index, other := range others {
		ids[other] = fmt.Sprintf("n%d", len(sorted)+index)
	}

	var chart strings.Builder
	chart.WriteString("flowchart LR\n")
	for index, folder := range slices.Sorted(maps.Keys(folders)) {
		label := folder
		if folder == "." {
			label = rootFolderLabel
		}
		fmt.Fprintf(&chart, "  subgraph f%d[\"%s\"]\n", index, escape(label))
		for _, value := range folders[folder] {
			fmt.Fprintf(&chart, "    %s[\"%s\"]\n", ids[value.Target], escape(value.Mark.Symbol()+path.Base(value.Target)))
		}
		chart.WriteString("  end\n")
	}
	for _, other := range others {
		label := other
		if dangling[other] {
			label += danglingSuffix
		}
		fmt.Fprintf(&chart, "  %s[\"%s\"]\n", ids[other], escape(label))
	}
	for _, value := range sorted {
		for _, edge := range value.Edges {
			if edge.Label == "" {
				fmt.Fprintf(&chart, "  %s --> %s\n", ids[value.Target], ids[edge.To])
			} else {
				fmt.Fprintf(&chart, "  %s -->|\"%s\"| %s\n", ids[value.Target], escape(edge.Label), ids[edge.To])
			}
		}
	}
	chart.WriteString(classDefs)
	classes := map[string][]string{}
	for _, value := range sorted {
		classes[string(value.Mark)] = append(classes[string(value.Mark)], ids[value.Target])
	}
	for _, other := range others {
		if dangling[other] {
			classes[danglingClass] = append(classes[danglingClass], ids[other])
		}
	}
	for _, class := range []string{string(sketch.MarkNew), string(sketch.MarkChanged), danglingClass} {
		if len(classes[class]) > 0 {
			fmt.Fprintf(&chart, "  class %s %s\n", strings.Join(classes[class], ","), class)
		}
	}
	return chart.String()
}

// escape makes text safe inside a quoted Mermaid label.
func escape(text string) string {
	return labelEscapes.Replace(text)
}

// Brief is a build brief's parts, which Markdown and the explorer's plan view both render.
type Brief struct {
	Criteria  []string      // the feature's acceptance criteria, as N. [x] TEXT lines
	Notes     []sketch.Note // the notes in build order
	Decisions []string      // each answered question as - PATH: TEXT, then each line of its answer after   >
	Open      []string      // each open question as - PATH:LINE: TEXT
}

// Parts returns the brief's parts for the notes and, when tix has it, their feature's ticket.
func Parts(tixFeature *ticket.Ticket, notes []sketch.Note) Brief {
	brief := Brief{Notes: Order(notes)}
	if tixFeature != nil {
		for index, criterion := range tixFeature.Acceptance {
			box := " "
			if criterion.Done {
				box = "x"
			}
			brief.Criteria = append(brief.Criteria, fmt.Sprintf(criterionLine, index+1, box, criterion.Text))
		}
	}
	for _, question := range questions.All(notes) {
		if question.Answer == "" {
			brief.Open = append(brief.Open, fmt.Sprintf(openLine, question.Path, question.Line, question.Text))
			continue
		}
		brief.Decisions = append(brief.Decisions, fmt.Sprintf(decisionLine, question.Path, question.Text))
		for _, line := range strings.Split(question.Answer, "\n") {
			brief.Decisions = append(brief.Decisions, answerPrefix+line)
		}
	}
	return brief
}

// NoteBody returns the note's body without its leading and trailing blank lines, and with its headings moved
// down under the brief's heading for the note: see demoteHeadings.
func NoteBody(value sketch.Note) string {
	return demoteHeadings(strings.Trim(value.Body, "\n"))
}

// demoteHeadings returns text with each ATX heading outside a fenced code block three levels deeper, so # becomes
// ####, capped at ######.
func demoteHeadings(text string) string {
	lines := strings.Split(text, "\n")
	fence := ""
	for index, line := range lines {
		trimmed := strings.TrimLeft(line, " ")
		if len(line)-len(trimmed) > maxHeadingIndent {
			continue
		}
		if marker := fenceMarker(trimmed); marker != "" {
			switch fence {
			case "":
				fence = marker
			case marker:
				fence = ""
			}
			continue
		}
		if fence != "" {
			continue
		}
		level := len(trimmed) - len(strings.TrimLeft(trimmed, "#"))
		rest := trimmed[level:]
		if level == 0 || level > maxHeadingLevel || (rest != "" && rest[0] != ' ' && rest[0] != '\t') {
			continue
		}
		lines[index] = strings.Repeat("#", min(level+headingDemotion, maxHeadingLevel)) + rest
	}
	return strings.Join(lines, "\n")
}

// fenceMarker returns ``` or ~~~ when line opens or closes a fenced code block, and "" otherwise.
func fenceMarker(line string) string {
	for _, marker := range fenceMarkers {
		if strings.HasPrefix(line, marker) {
			return marker
		}
	}
	return ""
}

// Markdown returns the build brief for the notes, which belong to feature, or to no one feature when it's empty: a
// title; the feature's title, description and acceptance criteria when tix has them; the Mermaid chart; each note
// in build order under its marked target, with its body as NoteBody gives it; the answered questions as
// decisions; and the open questions, if any.
func Markdown(feature string, tixFeature *ticket.Ticket, notes []sketch.Note) string {
	parts := Parts(tixFeature, notes)
	var brief strings.Builder
	if feature == "" {
		fmt.Fprintf(&brief, "# %s\n", planTitle)
	} else {
		fmt.Fprintf(&brief, "# %s: %s\n", planTitle, feature)
	}
	if tixFeature != nil {
		fmt.Fprintf(&brief, "\n## %s\n", tixFeature.Title)
		if description := strings.Trim(tixFeature.Description, "\n"); description != "" {
			fmt.Fprintf(&brief, "\n%s\n", description)
		}
		writeSection(&brief, "### "+criteriaTitle, parts.Criteria)
	}
	fmt.Fprintf(&brief, "\n## Graph\n\n```mermaid\n%s```\n", Mermaid(notes))
	brief.WriteString("\n## Build order\n")
	for _, value := range parts.Notes {
		fmt.Fprintf(&brief, "\n### %s%s\n", value.Mark.Symbol(), value.Target)
		if body := NoteBody(value); body != "" {
			fmt.Fprintf(&brief, "\n%s\n", body)
		}
	}
	writeSection(&brief, "## "+DecisionsTitle, parts.Decisions)
	writeSection(&brief, "## "+OpenTitle, parts.Open)
	return brief.String()
}

// writeSection writes lines under the heading, or nothing when there are none.
func writeSection(brief *strings.Builder, heading string, lines []string) {
	if len(lines) == 0 {
		return
	}
	fmt.Fprintf(brief, "\n%s\n\n%s\n", heading, strings.Join(lines, "\n"))
}
