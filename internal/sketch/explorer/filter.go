package explorer

import (
	"maps"
	"slices"

	"github.com/theroymind/go-tix/contracts/sketch"
	"github.com/theroymind/go-tix/internal/sketch/tree"
)

// askFeature opens the picker on the features to filter the tree to: an entry that clears the filter, then tix's
// features and every feature a note names, sorted, each once.
func (model *Model) askFeature() {
	features, err := model.listFeatures()
	if err != nil {
		model.status = err.Error()
		return
	}
	for _, node := range nodes(model.loaded) {
		if node.Note != nil && node.Note.Feature != "" {
			features = append(features, node.Note.Feature)
		}
	}
	slices.Sort(features)
	features = slices.Compact(features)
	model.picker = picker{kind: pickFeature, title: featureTitle, choices: append([]string{allFeatures}, features...)}
}

// filterFeature shows only the notes of the feature picked, or every note for allFeatures, with the folders above
// them open. New notes link to the feature.
func (model *Model) filterFeature(choice string) {
	model.feature = choice
	if choice == allFeatures {
		model.feature = ""
	}
	keep := model.selectedPath()
	model.tree = onlyFeature(model.loaded, model.feature)
	maps.Copy(model.expanded, tree.NoteFolders(model.tree))
	model.showRows(keep)
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
