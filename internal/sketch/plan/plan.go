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
			dangling[edge.To] = edge.Mark == ""
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

// Markdown returns the build brief for the notes, which belong to feature, or to no one feature when it's empty: a
// title; the feature's title, description and acceptance criteria when tix has them; the Mermaid chart; each note
// in build order under its marked target, with its body as written; the answered questions as decisions; and the
// open questions, if any.
func Markdown(feature string, tixFeature *ticket.Ticket, notes []sketch.Note) string {
	var brief strings.Builder
	if feature == "" {
		fmt.Fprintf(&brief, "# %s\n", planTitle)
	} else {
		fmt.Fprintf(&brief, "# %s: %s\n", planTitle, feature)
	}
	if tixFeature != nil {
		writeFeature(&brief, *tixFeature)
	}
	fmt.Fprintf(&brief, "\n## Graph\n\n```mermaid\n%s```\n", Mermaid(notes))
	brief.WriteString("\n## Build order\n")
	for _, value := range Order(notes) {
		fmt.Fprintf(&brief, "\n### %s%s\n", value.Mark.Symbol(), value.Target)
		if body := strings.Trim(value.Body, "\n"); body != "" {
			fmt.Fprintf(&brief, "\n%s\n", body)
		}
	}
	var decisions, open []questions.Question
	for _, question := range questions.All(notes) {
		if question.Answer == "" {
			open = append(open, question)
		} else {
			decisions = append(decisions, question)
		}
	}
	if len(decisions) > 0 {
		brief.WriteString("\n## Decisions\n\n")
		for _, question := range decisions {
			fmt.Fprintf(&brief, "- %s: %s\n", question.Path, question.Text)
			for _, line := range strings.Split(question.Answer, "\n") {
				fmt.Fprintf(&brief, "  > %s\n", line)
			}
		}
	}
	if len(open) > 0 {
		brief.WriteString("\n## Open questions\n\n")
		for _, question := range open {
			fmt.Fprintf(&brief, "- %s:%d: %s\n", question.Path, question.Line, question.Text)
		}
	}
	return brief.String()
}

// writeFeature writes the feature's title as a heading, then its description and acceptance criteria when it has
// them.
func writeFeature(brief *strings.Builder, value ticket.Ticket) {
	fmt.Fprintf(brief, "\n## %s\n", value.Title)
	if description := strings.Trim(value.Description, "\n"); description != "" {
		fmt.Fprintf(brief, "\n%s\n", description)
	}
	if len(value.Acceptance) == 0 {
		return
	}
	brief.WriteString("\n### Acceptance criteria\n\n")
	for index, criterion := range value.Acceptance {
		box := " "
		if criterion.Done {
			box = "x"
		}
		fmt.Fprintf(brief, "%d. [%s] %s\n", index+1, box, criterion.Text)
	}
}
