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
	model.picker = picker{kind: pickFile, title: fmt.Sprintf(fileTitle, node.Path),
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

// addLink adds a -> line linking the selected node's note to target, unless it links there already, and shows the
// note again.
func (model *Model) addLink(target string) {
	node := model.selected()
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

// link is one end of a link the links line shows: the path at the other end and its mark.
type link struct {
	path string
	mark sketch.Mark
}

// uses returns the links from node's note, each marked as its file is.
func uses(node *tree.Node) []link {
	if node.Note == nil {
		return nil
	}
	var links []link
	for _, edge := range node.Note.Edges {
		links = append(links, link{path: edge.To, mark: edge.Mark})
	}
	return links
}

// usedBy returns the links to node from the notes in top, each marked as the linking note's target is.
func usedBy(top *tree.Node, node *tree.Node) []link {
	var links []link
	for _, other := range nodes(top) {
		if other.Note == nil {
			continue
		}
		for _, edge := range other.Note.Edges {
			if edge.To == node.Path {
				links = append(links, link{path: other.Path, mark: other.Note.Mark})
				break
			}
		}
	}
	return links
}

// linksLine describes node's links, as in uses → + a.ts, ! c.ts   used by ← ~ d.ts, or returns "" when it has
// none. Each path shows by its base name, or in full when another file in top, or another path in the line, has
// the same base name. A link to a path that is neither a file nor planned shows !.
func linksLine(top *tree.Node, node *tree.Node) string {
	from, to := uses(node), usedBy(top, node)
	if len(from) == 0 && len(to) == 0 {
		return ""
	}
	paths := map[string]bool{}
	for _, other := range nodes(top) {
		if !other.IsDir {
			paths[other.Path] = true
		}
	}
	for _, each := range slices.Concat(from, to) {
		paths[each.path] = true
	}
	baseCount := map[string]int{}
	for each := range paths {
		baseCount[path.Base(each)]++
	}
	describe := func(links []link) string {
		var names []string
		for _, each := range links {
			symbol := each.mark.Symbol()
			if symbol == "" {
				symbol = danglingMark
			}
			name := path.Base(each.path)
			if baseCount[name] > 1 {
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
