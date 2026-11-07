package explorer

import (
	"errors"
	"fmt"
	"io/fs"
	"maps"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strings"

	"github.com/theroymind/go-tix/contracts/sketch"
	"github.com/theroymind/go-tix/internal/sketch/note"
	"github.com/theroymind/go-tix/internal/sketch/tree"
)

// askFeature opens the picker on the features to filter the tree to: an entry that clears the filter, then the
// features.
func (model *Model) askFeature() {
	features, err := model.features()
	if err != nil {
		model.status = err.Error()
		return
	}
	model.picker = picker{kind: pickFeature, title: featureTitle, choices: append([]string{allFeatures}, features...)}
}

// askLink opens the picker on the features to link the selected node's note to.
func (model *Model) askLink() {
	node := model.selected()
	if node == nil || node.Note == nil {
		model.status = linkNeedsNote
		return
	}
	features, err := model.features()
	if err != nil {
		model.status = err.Error()
		return
	}
	model.picker = picker{kind: pickLink, title: fmt.Sprintf(linkTitle, node.Path), choices: features}
}

// link rewrites the selected node's note to name feature in its front matter, and shows the tree again.
func (model *Model) link(feature string) {
	node := model.selected()
	err := model.setField(node.Note.Path, note.FeatureKey, feature)
	if err != nil {
		model.status = err.Error()
		return
	}
	model.reload(node.Path)
	if model.status == "" {
		model.status = fmt.Sprintf(linkedNote, node.Path, feature)
	}
}

// features returns tix's features and every feature a note names, sorted, each once.
func (model Model) features() ([]string, error) {
	features, err := model.listFeatures()
	if err != nil {
		return nil, err
	}
	for _, node := range nodes(model.loaded) {
		if node.Note != nil && node.Note.Feature != "" {
			features = append(features, node.Note.Feature)
		}
	}
	slices.Sort(features)
	return slices.Compact(features), nil
}

// filterFeature shows only the notes of the feature picked, or every note for allFeatures, with the folders above
// them open. New notes link to the feature.
func (model *Model) filterFeature(choice string) {
	model.feature = choice
	if choice == allFeatures {
		model.feature = ""
	}
	if err := saveFeature(model.featureFile, model.feature); err != nil {
		model.status = err.Error()
	}
	keep := model.selectedPath()
	model.applyFeature()
	maps.Copy(model.expanded, tree.NoteFolders(model.tree))
	model.showRows(keep)
}

// featureFile returns the file that keeps the feature filter for the branch checked out in the repo at root,
// under its git folder, or "" outside git or with no branch checked out.
func featureFile(root string) string {
	gitFolder, err := gitOutput(root, "rev-parse", "--absolute-git-dir")
	if err != nil {
		return ""
	}
	branch, err := gitOutput(root, "symbolic-ref", "--short", "HEAD")
	if err != nil {
		return ""
	}
	return filepath.Join(gitFolder, featureFolder, filepath.FromSlash(branch))
}

// gitOutput runs git with arguments in root and returns what it prints, trimmed.
func gitOutput(root string, arguments ...string) (string, error) {
	command := exec.Command("git", arguments...)
	command.Dir = root
	output, err := command.Output()
	return strings.TrimSpace(string(output)), err
}

// readFeature returns the feature kept in filePath, or "" when there's none.
func readFeature(filePath string) string {
	if filePath == "" {
		return ""
	}
	text, err := os.ReadFile(filePath)
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(text))
}

// saveFeature keeps feature in filePath, or removes the file when feature is "". With no file it does nothing.
func saveFeature(filePath, feature string) error {
	switch {
	case filePath == "":
		return nil
	case feature == "":
		err := os.Remove(filePath)
		if errors.Is(err, fs.ErrNotExist) {
			return nil
		}
		return err
	}
	if err := os.MkdirAll(filepath.Dir(filePath), folderMode); err != nil {
		return err
	}
	return os.WriteFile(filePath, []byte(feature+"\n"), fileMode)
}

// applyFeature shows the loaded tree with only the feature's notes, and counts their readiness.
func (model *Model) applyFeature() {
	model.tree = onlyFeature(model.loaded, model.feature)
	model.readiness = countReadiness(model.tree)
}

// onlyFeature returns top rebuilt with only the notes of feature, so a file planned by another feature's note
// shows plain and that note's ghosts go. With no feature it returns top.
func onlyFeature(top *tree.Node, feature string) *tree.Node {
	if feature == "" {
		return top
	}
	var files []string
	var notes []sketch.Note
	for _, node := range nodes(top) {
		if !node.IsDir && !node.Ghost {
			files = append(files, node.Path)
		}
		if node.Note != nil && node.Note.Feature == feature {
			notes = append(notes, *node.Note)
		}
	}
	return tree.Build(files, notes, nil)
}

// nodes returns every node below top, each folder before the nodes inside it, in the tree's order.
func nodes(top *tree.Node) []*tree.Node {
	var all []*tree.Node
	for _, child := range top.Children {
		all = append(all, child)
		all = append(all, nodes(child)...)
	}
	return all
}
