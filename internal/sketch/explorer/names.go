package explorer

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"os"
	"slices"
)

// askName opens the picker on the names in the selected note's target, when the note is attached to a file that
// exists.
func (model *Model) askName() {
	node := model.selected()
	if node == nil || node.Note == nil || node.Ghost {
		model.status = refNeedsAttachedNote
		return
	}
	names, err := fileNames(model.filePath(node.Path))
	if err != nil {
		model.status = err.Error()
		return
	}
	if len(names) == 0 {
		model.status = fmt.Sprintf(noNames, node.Path)
		return
	}
	model.picker = picker{kind: pickName, title: fmt.Sprintf(namesTitle, node.Path), choices: names}
}

// addName adds a line naming name to the end of the selected node's note, and shows the note again.
func (model *Model) addName(name string) {
	node := model.selected()
	if err := appendLine(model.filePath(node.Note.Path), fmt.Sprintf(usesLine, name)); err != nil {
		model.status = err.Error()
		return
	}
	model.reload(node.Path)
}

// fileNames returns the identifiers in the first previewMaxLen bytes of the file at filePath, sorted, each once.
func fileNames(filePath string) ([]string, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return nil, err
	}
	defer func() { _ = file.Close() }()
	head, err := io.ReadAll(io.LimitReader(file, previewMaxLen))
	if err != nil {
		return nil, err
	}
	names := identifierPattern.FindAllString(string(head), -1)
	slices.Sort(names)
	return slices.Compact(names), nil
}

// appendLine adds line to the end of the file at filePath, first ending the file's last line when it isn't.
func appendLine(filePath, line string) error {
	text, err := os.ReadFile(filePath)
	if err != nil {
		return err
	}
	if len(text) > 0 && !bytes.HasSuffix(text, []byte("\n")) {
		line = "\n" + line
	}
	file, err := os.OpenFile(filePath, os.O_WRONLY|os.O_APPEND, fileMode)
	if err != nil {
		return err
	}
	_, err = file.WriteString(line + "\n")
	return errors.Join(err, file.Close())
}
