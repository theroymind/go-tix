package explorer

import (
	"fmt"
	"strings"

	tea "charm.land/bubbletea/v2"
)

type pickKind int

const (
	pickNone    pickKind = iota
	pickName             // picks a name from the selected note's target, to add to the note
	pickFeature          // picks the feature to filter the tree to
	pickLink             // picks the feature to link the selected note to
)

// picker is a list the preview pane shows in place of the preview, filtered as you type: choices is the whole
// list, typed is the filter, and cursor indexes the choices that match it.
type picker struct {
	kind    pickKind
	title   string
	choices []string
	typed   string
	cursor  int
}

// matches returns the choices that hold the filter, ignoring case.
func (list picker) matches() []string {
	typed := strings.ToLower(list.typed)
	var matches []string
	for _, choice := range list.choices {
		if strings.Contains(strings.ToLower(choice), typed) {
			matches = append(matches, choice)
		}
	}
	return matches
}

// press handles a key: the arrows and ctrl+n/ctrl+p move, backspace takes back a letter, and any other key types
// into the filter, so j and k type rather than move. On enter it returns the choice under the cursor and true.
func (list *picker) press(message tea.KeyPressMsg) (string, bool) {
	switch message.String() {
	case "down", "ctrl+n":
		list.cursor = max(0, min(list.cursor+1, len(list.matches())-1))
	case "up", "ctrl+p":
		list.cursor = max(0, list.cursor-1)
	case "backspace":
		typed := []rune(list.typed)
		list.typed = string(typed[:max(0, len(typed)-1)])
		list.cursor = 0
	case "enter":
		matches := list.matches()
		if len(matches) == 0 {
			return "", false
		}
		return matches[list.cursor], true
	default:
		list.typeText(message.Text)
	}
	return "", false
}

// typeText adds text to the filter and puts the cursor back on the first match.
func (list *picker) typeText(text string) {
	list.typed += text
	list.cursor = 0
}

// lines returns up to height lines drawing the picker: its title, the filter, and the matches around the cursor,
// the one under it marked.
func (list picker) lines(height int) []string {
	lines := []string{list.title, fmt.Sprintf(pickerFilter, list.typed)}
	matches := list.matches()
	if len(matches) == 0 {
		return append(lines, noMatches)
	}
	shown := max(1, height-len(lines))
	first := max(0, list.cursor-shown+1)
	for index := first; index < len(matches) && index < first+shown; index++ {
		marker := noPick
		if index == list.cursor {
			marker = pickMarker
		}
		lines = append(lines, marker+matches[index])
	}
	return lines
}

// answerPicker passes a key to the picker: esc closes it, and a pick adds the name to the note, filters the tree
// to the feature, or links the note to it.
func (model *Model) answerPicker(message tea.KeyPressMsg) tea.Cmd {
	switch message.String() {
	case "esc":
		model.picker = picker{}
		return nil
	case "ctrl+c":
		return tea.Quit
	}
	choice, picked := model.picker.press(message)
	if !picked {
		return nil
	}
	kind := model.picker.kind
	model.picker = picker{}
	switch kind {
	case pickName:
		model.addName(choice)
	case pickFeature:
		model.filterFeature(choice)
	case pickLink:
		model.link(choice)
	}
	return nil
}
