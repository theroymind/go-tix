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
	ErrFolderName   = errors.New("that's a folder; type a file name")
)

type promptKind int

const (
	promptNone   promptKind = iota
	promptName              // asks for the name of a file to plan in a folder
	promptDelete            // asks y/n before deleting a note, or every note under a folder
)

// prompt is the question the footer asks, if any: path is the folder a name prompt creates in, or the note or
// folder a delete prompt deletes, notes are the notes a delete prompt deletes, and typed is the name typed so far.
type prompt struct {
	kind  promptKind
	path  string
	notes []string
	typed string
}

// line returns the prompt as the footer shows it, or "" when there is none.
func (prompt prompt) line() string {
	switch prompt.kind {
	case promptName:
		return fmt.Sprintf(namePrompt, prompt.path, prompt.typed)
	case promptDelete:
		if len(prompt.notes) == 1 && prompt.notes[0] == prompt.path {
			return fmt.Sprintf(deletePrompt, prompt.path)
		}
		return fmt.Sprintf(deleteFolderPrompt, count(len(prompt.notes), "note"), prompt.path)
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
	case "tab":
		model.prompt.typed = completeFolder(model.loaded, model.prompt.path, model.prompt.typed)
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

// answerDelete deletes the prompt's notes on y, along with the folders that leaves empty, and cancels on any other
// key.
func (model *Model) answerDelete(message tea.KeyPressMsg) {
	notes := model.prompt.notes
	model.prompt = prompt{}
	if message.String() != "y" {
		return
	}
	var deleted int
	var failed error
	for _, notePath := range notes {
		if err := os.Remove(model.filePath(notePath)); err != nil {
			failed = err
			continue
		}
		deleted++
		model.removeEmptyFolders(path.Dir(notePath))
	}
	model.reload(model.selectedPath())
	switch {
	case failed != nil:
		model.status = failed.Error()
	case len(notes) > 1:
		model.status = fmt.Sprintf(deletedNotes, count(deleted, "note"))
	}
}

// removeEmptyFolders removes the folder at folder, under the repo root, when it's empty, and then each folder above
// it that this leaves empty, up to the root. A folder holding anything stays, since os.Remove won't take it.
func (model Model) removeEmptyFolders(folder string) {
	for ; folder != rootFolder; folder = path.Dir(folder) {
		if os.Remove(model.filePath(folder)) != nil {
			return
		}
	}
}

// newTarget returns the slash-separated path from the repo root of the file typed names in folder, with a stray
// .sketch.md or .sketch taken out. The name may hold folders, as in hooks/useX.ts, but not empty, . or .. ones,
// and must not be absolute or name a folder, whether it's on disk under root or a ghost in top.
func newTarget(root string, top *tree.Node, folder, typed string) (string, error) {
	name := strings.TrimSpace(typed)
	if strings.HasPrefix(name, "/") || filepath.IsAbs(name) {
		return "", ErrAbsoluteName
	}
	name = withoutSketch(name)
	if name == "" {
		return "", ErrEmptyName
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

// withoutSketch returns name with a trailing .sketch.md taken off, and then a .sketch at the end or before the
// last extension, so useOrbit.ts.sketch.md and useOrbit.sketch.ts both give useOrbit.ts.
func withoutSketch(name string) string {
	name = strings.TrimSuffix(name, sketch.NoteSuffix)
	folder, base := path.Split(name)
	extension := path.Ext(base)
	stem := strings.TrimSuffix(base, extension)
	switch {
	case extension == sketchExt:
		base = stem
	case strings.HasSuffix(stem, sketchExt):
		base = strings.TrimSuffix(stem, sketchExt) + extension
	}
	return folder + base
}

// completeFolder returns typed with its last segment completed to the folders under folder in top that it starts:
// to the one folder and a slash when only one matches, or as far as the matches agree when several do. It returns
// typed as it is when none matches.
func completeFolder(top *tree.Node, folder, typed string) string {
	typedFolder, partial := path.Split(typed)
	parent := find(top, path.Join(folder, typedFolder))
	if parent == nil {
		return typed
	}
	var matches []string
	for _, child := range parent.Children {
		if child.IsDir && strings.HasPrefix(child.Name, partial) {
			matches = append(matches, child.Name)
		}
	}
	switch len(matches) {
	case 0:
		return typed
	case 1:
		return typedFolder + matches[0] + "/"
	}
	common := matches[0]
	for _, match := range matches[1:] {
		for !strings.HasPrefix(match, common) {
			common = common[:len(common)-1]
		}
	}
	return typedFolder + common
}

// find returns the node at nodePath under top, top itself for the root folder, or nil when there's none.
func find(top *tree.Node, nodePath string) *tree.Node {
	if nodePath == rootFolder {
		return top
	}
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

// noteText is a new note: front matter linking it to feature when there is one and marking it draft. It has no
// heading, since sketch plan heads each note with the file it plans.
func noteText(feature string) string {
	frontMatter := noteFence
	if feature != "" {
		frontMatter += fmt.Sprintf(noteFeature, feature)
	}
	return frontMatter + noteDraft + noteFence
}

// writeNew writes text to a new file at filePath, making the folders above it, and reports whether it made the
// file. A file already there is left as it is.
func writeNew(filePath, text string) (bool, error) {
	if err := os.MkdirAll(filepath.Dir(filePath), folderMode); err != nil {
		return false, err
	}
	file, err := os.OpenFile(filePath, os.O_WRONLY|os.O_CREATE|os.O_EXCL, fileMode)
	if errors.Is(err, fs.ErrExist) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	_, err = file.WriteString(text)
	return true, errors.Join(err, file.Close())
}
