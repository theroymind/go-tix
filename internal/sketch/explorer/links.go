package explorer

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"path"
	"slices"
	"strings"

	"github.com/theroymind/go-tix/contracts/sketch"
	"github.com/theroymind/go-tix/internal/sketch/repo"
	"github.com/theroymind/go-tix/internal/sketch/tree"
)

// askLinkTarget opens the picker on the files the selected note can link to.
func (model *Model) askLinkTarget() {
	node := model.selected()
	switch {
	case node == nil || node.Note == nil:
		model.status = refNeedsNote
		return
	case node.Problem != nil:
		model.status = node.Problem.Error()
		return
	}
	model.picker = picker{kind: pickFile, title: fmt.Sprintf(fileTitle, node.Path), subject: node.Path,
		choices: linkChoices(model.loaded, node.Path)}
}

// linkChoices returns every file in the tree, the planned ones included, except from itself.
func linkChoices(top *tree.Node, from string) []string {
	var choices []string
	for _, node := range nodes(top) {
		if !node.IsDir && node.Path != from {
			choices = append(choices, node.Path)
		}
	}
	return choices
}

// addLink adds a -> line linking the note of the node at nodePath to target, unless it links there already, and
// shows the note again.
func (model *Model) addLink(nodePath, target string) {
	node := model.subjectNode(nodePath)
	if node == nil {
		return
	}
	if slices.ContainsFunc(node.Note.Edges, func(edge sketch.Edge) bool { return edge.To == target }) {
		model.status = fmt.Sprintf(alreadyLinked, node.Path, target)
		return
	}
	if err := appendLine(model.filePath(node.Note.Path), linkMarker+target); err != nil {
		model.status = err.Error()
		return
	}
	model.reload(node.Path)
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

// link is one end of a link the links line shows: the path at the other end, its mark, and whether it leads
// nowhere.
type link struct {
	path     string
	mark     sketch.Mark
	dangling bool
}

// linkIndex is what the links line needs from the whole tree, worked out once per load: the links to each path
// from the notes, and how many files and linked paths share each base name.
type linkIndex struct {
	usedBy    map[string][]link
	baseCount map[string]int
}

// indexLinks returns the linkIndex of top. Each link to a path is marked as the linking note's target is, in the
// tree's order, once per note.
func indexLinks(top *tree.Node) linkIndex {
	index := linkIndex{usedBy: map[string][]link{}, baseCount: map[string]int{}}
	paths := map[string]bool{}
	for _, node := range nodes(top) {
		if !node.IsDir {
			paths[node.Path] = true
		}
		if node.Note == nil {
			continue
		}
		linked := map[string]bool{}
		for _, edge := range node.Note.Edges {
			paths[edge.To] = true
			if !linked[edge.To] {
				linked[edge.To] = true
				index.usedBy[edge.To] = append(index.usedBy[edge.To], link{path: node.Path, mark: node.Note.Mark})
			}
		}
	}
	for each := range paths {
		index.baseCount[path.Base(each)]++
	}
	return index
}

// uses returns the links from node's note, each marked as its file is.
func uses(node *tree.Node) []link {
	if node.Note == nil {
		return nil
	}
	var links []link
	for _, edge := range node.Note.Edges {
		links = append(links, link{path: edge.To, mark: edge.Mark, dangling: repo.Dangling(edge)})
	}
	return links
}

// linksLine describes node's links, as in uses → + a.ts, ! c.ts   used by ← ~ d.ts, or returns "" when it has
// none. Each path shows by its base name, or in full when another file or linked path in the tree has the same
// base name. A link to a path that is neither a file nor planned shows !.
func linksLine(index linkIndex, node *tree.Node) string {
	from, to := uses(node), index.usedBy[node.Path]
	if len(from) == 0 && len(to) == 0 {
		return ""
	}
	describe := func(links []link) string {
		var names []string
		for _, each := range links {
			symbol := each.mark.Symbol()
			if each.dangling {
				symbol = danglingMark
			}
			name := path.Base(each.path)
			if index.baseCount[name] > 1 {
				name = each.path
			}
			names = append(names, symbol+" "+name)
		}
		return strings.Join(names, linkSeparator)
	}
	var parts []string
	if len(from) > 0 {
		parts = append(parts, usesLabel+describe(from))
	}
	if len(to) > 0 {
		parts = append(parts, usedByLabel+describe(to))
	}
	return strings.Join(parts, linksGap)
}
