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

// answer adds a "> " line right under the first open question in the selected node's note and opens the note in
// the editor on that line.
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
	if err := insertLine(model.filePath(node.Note.Path), questionLine, answerLine); err != nil {
		model.status = err.Error()
		return nil
	}
	model.reload(node.Path)
	return model.openNote(node.Note.Path, questionLine+1)
}

// insertLine puts line into the file at filePath right after its line number after, counting from 1.
func insertLine(filePath string, after int, line string) error {
	text, err := os.ReadFile(filePath)
	if err != nil {
		return err
	}
	lines := strings.Split(string(text), "\n")
	lines = slices.Insert(lines, after, line)
	return os.WriteFile(filePath, []byte(strings.Join(lines, "\n")), fileMode)
}
