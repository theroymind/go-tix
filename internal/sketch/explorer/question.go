package explorer

import (
	"os"
	"slices"
	"strings"

	tea "charm.land/bubbletea/v2"

	"github.com/theroymind/go-tix/contracts/sketch"
	"github.com/theroymind/go-tix/internal/sketch/tree"
)

// jumpToQuestion moves the cursor to the next node in the tree whose note has an open question, or the previous
// one when direction is -1, opening the folders above it, and wraps around at the ends.
func (model *Model) jumpToQuestion(direction int) {
	all := nodes(model.tree)
	selectedPath := model.selectedPath()
	current := slices.IndexFunc(all, func(node *tree.Node) bool { return node.Path == selectedPath })
	if current < 0 && direction < 0 {
		current = len(all)
	}
	for step := 1; step <= len(all); step++ {
		index := (current + direction*step + len(all)) % len(all)
		node := all[index]
		if node.OpenQuestions > 0 {
			model.openFoldersAbove(node.Path)
			model.showRows(node.Path)
			return
		}
	}
	model.status = noOpenQuestions
}

// answer adds a "> " line right under the first open question in the selected node's note, unless the line there
// starts with > already, as an empty answer does, and opens the note in the editor on that line.
func (model *Model) answer() tea.Cmd {
	node := model.selected()
	if node == nil || node.Note == nil {
		model.status = noneToAnswer
		return nil
	}
	open := slices.IndexFunc(node.Note.Questions, func(question sketch.Question) bool { return question.Answer == "" })
	if open < 0 {
		model.status = noneToAnswer
		return nil
	}
	questionLine := node.Note.Questions[open].Line
	if err := addAnswerLine(model.filePath(node.Note.Path), questionLine); err != nil {
		model.status = err.Error()
		return nil
	}
	model.reload(node.Path)
	return model.openNote(node.Note.Path, questionLine+1)
}

// addAnswerLine puts an answerLine into the file at filePath right after the question on line questionLine,
// counting from 1, unless the line after it starts with > already.
func addAnswerLine(filePath string, questionLine int) error {
	text, err := os.ReadFile(filePath)
	if err != nil {
		return err
	}
	lines := strings.Split(string(text), "\n")
	if questionLine < len(lines) && strings.HasPrefix(lines[questionLine], answerMarker) {
		return nil
	}
	lines = slices.Insert(lines, questionLine, answerLine)
	return os.WriteFile(filePath, []byte(strings.Join(lines, "\n")), fileMode)
}
