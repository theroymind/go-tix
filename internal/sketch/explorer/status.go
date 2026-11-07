package explorer

import (
	"fmt"
	"os"

	"github.com/theroymind/go-tix/contracts/sketch"
	"github.com/theroymind/go-tix/internal/sketch/note"
	"github.com/theroymind/go-tix/internal/sketch/repo"
	"github.com/theroymind/go-tix/internal/sketch/tree"
)

// readiness counts the notes in a tree that are ready to build from, the ones still draft, and the links in them
// that lead nowhere.
type readiness struct {
	draft, ready, dangling int
}

// countReadiness counts the readiness of the notes below top that read. A note with no status counts as draft.
func countReadiness(top *tree.Node) readiness {
	var counts readiness
	for _, node := range nodes(top) {
		if node.Note == nil || node.Problem != nil {
			continue
		}
		if node.Note.Status == sketch.StatusReady {
			counts.ready++
		} else {
			counts.draft++
		}
		for _, edge := range node.Note.Edges {
			if repo.Dangling(edge) {
				counts.dangling++
			}
		}
	}
	return counts
}

// isDraft reports whether node has a note that isn't ready.
func isDraft(node *tree.Node) bool {
	return node.Note != nil && node.Note.Status != sketch.StatusReady
}

// toggleReady marks the selected node's note ready, or back to draft when it's ready already, and shows the tree
// again.
func (model *Model) toggleReady() {
	node := model.selected()
	switch {
	case node == nil || node.Note == nil:
		model.status = readyNeedsNote
		return
	case node.Problem != nil:
		model.status = node.Problem.Error()
		return
	}
	status := sketch.StatusReady
	if node.Note.Status == sketch.StatusReady {
		status = sketch.StatusDraft
	}
	if err := model.setField(node.Note.Path, note.StatusKey, string(status)); err != nil {
		model.status = err.Error()
		return
	}
	model.reload(node.Path)
	if model.status == "" {
		model.status = fmt.Sprintf(markedNote, node.Path, status)
	}
}

// setField rewrites the front matter of the note at notePath to give key the value.
func (model Model) setField(notePath, key, value string) error {
	filePath := model.filePath(notePath)
	text, err := os.ReadFile(filePath)
	if err != nil {
		return err
	}
	updated, err := note.SetField(string(text), key, value)
	if err != nil {
		return err
	}
	return os.WriteFile(filePath, []byte(updated), fileMode)
}
