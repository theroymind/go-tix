package command

import (
	"fmt"
	"io"

	cli "github.com/theroymind/railyard-cli"

	"github.com/theroymind/go-tix/contracts/sketch"
	"github.com/theroymind/go-tix/internal/sketch/questions"
	"github.com/theroymind/go-tix/internal/sketch/refs"
	"github.com/theroymind/go-tix/internal/sketch/repo"
)

// runCheck prints each reason the notes under the repo root, or with --feature only that feature's, aren't ready
// to build from, one per line, and fails with ErrNotReady when there is any: a note that isn't ready, an open
// question, a link that leads nowhere, a feature tix has no ticket for, a note that didn't read, or a feature with
// no notes at all.
func runCheck(arguments []string, stdout, stderr io.Writer) error {
	parsed, err := cli.Parse(arguments, cli.Specification{
		Options: []cli.Option{{Name: optionFeature, Type: cli.OptionTypeString}},
	})
	if err != nil {
		return err
	}
	feature, _ := parsed.String(optionFeature)
	root, loaded, err := loadNotes(feature)
	if err != nil {
		return err
	}
	notes := loaded.Notes
	unknown := unknownFeatures(refs.New(root), notes, stderr)

	var problems []string
	if feature != "" && len(notes) == 0 {
		problems = append(problems, fmt.Sprintf(noNotesProblem, feature))
	}
	for _, value := range notes {
		problems = append(problems, noteProblems(value, unknown)...)
	}
	for _, problem := range loaded.Problems {
		problems = append(problems, problem.Error())
	}
	for _, problem := range problems {
		if _, err := fmt.Fprintln(stdout, problem); err != nil {
			return err
		}
	}
	if len(problems) > 0 {
		return ErrNotReady
	}
	return nil
}

// noteProblems returns, as PATH:LINE: problem lines, each reason the note isn't ready to build from: its status
// isn't ready, it asks an open question, it links to a file that neither exists nor is planned, or its feature is
// among unknown.
func noteProblems(value sketch.Note, unknown map[string]bool) []string {
	var problems []string
	switch value.Status {
	case sketch.StatusReady:
	case "":
		problems = append(problems, fmt.Sprintf(noStatusProblem, value.Path))
	default:
		problems = append(problems, fmt.Sprintf(notReadyProblem, value.Path, value.Status))
	}
	for _, question := range questions.Open([]sketch.Note{value}) {
		problems = append(problems, fmt.Sprintf(openQuestionProblem, question.Path, question.Line, question.Text))
	}
	for _, edge := range value.Edges {
		if repo.Dangling(edge) {
			problems = append(problems, fmt.Sprintf(danglingLinkProblem, value.Path, edge.To))
		}
	}
	if unknown[value.Feature] {
		problems = append(problems, fmt.Sprintf(unknownFeatureProblem, value.Path, value.Feature))
	}
	return problems
}

// notesOf returns the notes that belong to feature, in order.
func notesOf(notes []sketch.Note, feature string) []sketch.Note {
	var featureNotes []sketch.Note
	for _, value := range notes {
		if value.Feature == feature {
			featureNotes = append(featureNotes, value)
		}
	}
	return featureNotes
}
