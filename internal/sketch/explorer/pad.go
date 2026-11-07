package explorer

import (
	"os"

	"charm.land/bubbles/v2/textarea"
	tea "charm.land/bubbletea/v2"
)

// padMsg asks the explorer to edit the note at filePath in its preview pane, with the cursor on line when it's
// above 0.
type padMsg struct {
	filePath string
	line     int
}

// paneOpen is the Open the explorer uses outside nvim: it edits the note in the preview pane, beside the tree,
// instead of handing the terminal to an editor. The pad reloads the tree itself when it closes, so done goes
// unused.
func paneOpen(filePath string, line int, _ func(error) tea.Msg) tea.Cmd {
	return func() tea.Msg { return padMsg{filePath: filePath, line: line} }
}

// pad is the note being edited in the preview pane, if any: the file it saves to, the text last read or saved,
// the text area that edits it, and whether it's asking before throwing away unsaved changes.
type pad struct {
	filePath   string
	saved      string
	area       textarea.Model
	discarding bool
}

func (pad pad) active() bool {
	return pad.filePath != ""
}

// startPad loads the note message names into the pad, with the cursor on its line, and focuses it.
func (model *Model) startPad(message padMsg) tea.Cmd {
	text, err := os.ReadFile(message.filePath)
	if err != nil {
		model.status = err.Error()
		return nil
	}
	area := textarea.New()
	area.Prompt = ""
	area.ShowLineNumbers = false
	area.MaxHeight = 0 // no cap on the note's length
	area.SetStyles(textarea.DefaultStyles(model.isDark))
	area.SetValue(string(text))
	area.MoveToBegin()
	for area.Line() < message.line-1 {
		before := area.Line()
		area.CursorDown()
		if area.Line() == before {
			break
		}
	}
	if message.line > 0 {
		area.CursorEnd()
	}
	// saved is the text as the area holds it, which turns tabs into spaces, so closing an untouched note leaves
	// the file as it was.
	model.pad = pad{filePath: message.filePath, saved: area.Value(), area: area}
	model.sizePad()
	return model.pad.area.Focus()
}

// sizePad fits the pad to the preview pane.
func (model *Model) sizePad() {
	if !model.pad.active() {
		return
	}
	model.pad.area.SetWidth(max(1, model.previewWidth()))
	model.pad.area.SetHeight(model.paneHeight())
}

// updatePad types a key into the pad: esc saves the note and goes back to the tree, ctrl+s saves it, ctrl+c
// saves it and quits, and ctrl+x goes back to the tree without saving, asking first when there are changes to
// lose.
func (model *Model) updatePad(message tea.Msg) tea.Cmd {
	if press, isKey := message.(tea.KeyPressMsg); isKey {
		switch {
		case model.pad.discarding:
			model.pad.discarding = false
			if press.String() == "y" {
				model.discardPad()
			}
			return nil
		case press.String() == "ctrl+x":
			if model.pad.area.Value() == model.pad.saved {
				model.discardPad()
				return nil
			}
			model.pad.discarding = true
			return nil
		case press.String() == "esc":
			model.closePad()
			return nil
		case press.String() == "ctrl+s":
			model.savePad()
			return nil
		case press.String() == "ctrl+c":
			if model.savePad() {
				return tea.Quit
			}
			return nil
		}
	}
	var command tea.Cmd
	model.pad.area, command = model.pad.area.Update(message)
	return command
}

// savePad writes the pad's text to its note when it has changed since it was read or last saved, and reports
// whether the note is saved, putting the error in the status line when it isn't.
func (model *Model) savePad() bool {
	text := model.pad.area.Value()
	if text == model.pad.saved {
		return true
	}
	if err := os.WriteFile(model.pad.filePath, []byte(text), fileMode); err != nil {
		model.status = err.Error()
		return false
	}
	model.pad.saved = text
	model.status = savedNote
	return true
}

// discardPad goes back to the tree, leaving the note as it was last read or saved.
func (model *Model) discardPad() {
	model.pad = pad{}
	model.reload(model.selectedPath())
}

// closePad saves the note and, once it's saved, puts the preview back and reads the tree again, keeping the cursor
// where it is. A note that fails to save keeps the pad open.
func (model *Model) closePad() {
	if !model.savePad() {
		return
	}
	model.pad = pad{}
	model.reload(model.selectedPath())
}
