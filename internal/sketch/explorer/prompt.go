package explorer

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"slices"
	"strings"

	tea "charm.land/bubbletea/v2"

	"github.com/theroymind/go-tix/contracts/sketch"
	"github.com/theroymind/go-tix/internal/sketch/tree"
)

var (
	ErrEmptyName    = errors.New("type a file name")
	ErrAbsoluteName = errors.New("the name must be relative to the folder")
	ErrNameSegment  = errors.New("the name can't hold an empty, . or .. folder")
	ErrNoteName     = errors.New("type the file's name, without " + sketch.NoteSuffix)
	ErrFolderName   = errors.New("that's a folder; type a file name")
)

type promptKind int

const (
	promptNone   promptKind = iota
	promptName              // asks for the name of a file to plan in a folder
	promptDelete            // asks y/n before deleting a note
)

// prompt is the question the footer asks, if any: path is the folder a name prompt creates in, or the note a
// delete prompt deletes, and typed is the name typed so far.
type prompt struct {
	kind  promptKind
	path  string
	typed string
}

// line returns the prompt as the footer shows it, or "" when there is none.
func (prompt prompt) line() string {
	switch prompt.kind {
	case promptName:
		return fmt.Sprintf(namePrompt, prompt.path, prompt.typed)
	case promptDelete:
		return fmt.Sprintf(deletePrompt, prompt.path)
	}
	return ""
}

// answerName types into the name prompt. enter creates the note for the typed name, or keeps the prompt open
// saying what's wrong with it; esc cancels.
func (model *Model) answerName(message tea.KeyPressMsg) tea.Cmd {
	switch message.String() {
	case "esc":
		model.prompt = prompt{}
	case "ctrl+c":
		return tea.Quit
	case "backspace":
		typed := []rune(model.prompt.typed)
		model.prompt.typed = string(typed[:max(0, len(typed)-1)])
	case "enter":
		target, err := newTarget(model.root, model.tree, model.prompt.path, model.prompt.typed)
		if err != nil {
			model.status = err.Error()
			return nil
		}
		model.prompt = prompt{}
		return model.createNote(target)
	default:
		model.prompt.typed += message.Text
	}
	return nil
}

// answerDelete deletes the note on y, and cancels on any other key.
func (model *Model) answerDelete(message tea.KeyPressMsg) {
	notePath := model.prompt.path
	model.prompt = prompt{}
	if message.String() != "y" {
		return
	}
	if err := os.Remove(model.filePath(notePath)); err != nil {
		model.status = err.Error()
		return
	}
	model.reload(model.selectedPath())
}

// newTarget returns the slash-separated path from the repo root of the file typed names in folder. The name may
// hold folders, as in hooks/useX.ts, but not empty, . or .. ones, and must not be absolute, name a note, or name a
// folder, whether it's on disk under root or a ghost in top.
func newTarget(root string, top *tree.Node, folder, typed string) (string, error) {
	name := strings.TrimSpace(typed)
	switch {
	case name == "":
		return "", ErrEmptyName
	case strings.HasPrefix(name, "/") || filepath.IsAbs(name):
		return "", ErrAbsoluteName
	case strings.HasSuffix(name, sketch.NoteSuffix):
		return "", ErrNoteName
	}
	for _, segment := range strings.Split(name, "/") {
		if segment == "" || segment == "." || segment == ".." {
			return "", ErrNameSegment
		}
	}
	target := path.Join(folder, name)
	if node := find(top, target); node != nil && node.IsDir {
		return "", ErrFolderName
	}
	if info, err := os.Stat(filepath.Join(root, filepath.FromSlash(target))); err == nil && info.IsDir() {
		return "", ErrFolderName
	}
	return target, nil
}

// find returns the node at nodePath under top, or nil when there's none.
func find(top *tree.Node, nodePath string) *tree.Node {
	node := top
	for _, name := range strings.Split(nodePath, "/") {
		index := slices.IndexFunc(node.Children, func(child *tree.Node) bool { return child.Name == name })
		if index < 0 {
			return nil
		}
		node = node.Children[index]
	}
	return node
}

// noteText is a new note planning target: front matter linking it to feature when there is one and marking it
// draft, then a heading naming the file.
func noteText(feature, target string) string {
	frontMatter := noteFence
	if feature != "" {
		frontMatter += fmt.Sprintf(noteFeature, feature)
	}
	frontMatter += noteDraft + noteFence
	return frontMatter + fmt.Sprintf(noteHeading, path.Base(target))
}

// writeNew writes text to a new file at filePath, making the folders above it. A file already there is left as it
// is.
func writeNew(filePath, text string) error {
	if err := os.MkdirAll(filepath.Dir(filePath), folderMode); err != nil {
		return err
	}
	file, err := os.OpenFile(filePath, os.O_WRONLY|os.O_CREATE|os.O_EXCL, fileMode)
	if errors.Is(err, fs.ErrExist) {
		return nil
	}
	if err != nil {
		return err
	}
	_, err = file.WriteString(text)
	return errors.Join(err, file.Close())
}
