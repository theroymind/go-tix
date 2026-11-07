// Package questions lists the questions across a set of notes, each with the note and line that asks it.
package questions

import (
	"cmp"
	"slices"

	"github.com/theroymind/go-tix/contracts/sketch"
)

// Question is one question: the path of the note that asks it, its 1-based line in that note, its text, and its
// answer, empty while it's open.
type Question struct {
	Path   string `json:"path"`
	Line   int    `json:"line"`
	Text   string `json:"text"`
	Answer string `json:"answer,omitempty"`
}

// All returns every question in notes, answered or not, ordered by note path and then by line.
func All(notes []sketch.Note) []Question {
	all := []Question{}
	for _, value := range notes {
		for _, question := range value.Questions {
			all = append(all, Question{Path: value.Path, Line: question.Line, Text: question.Text, Answer: question.Answer})
		}
	}
	slices.SortFunc(all, func(a, b Question) int {
		return cmp.Or(cmp.Compare(a.Path, b.Path), cmp.Compare(a.Line, b.Line))
	})
	return all
}

// Open returns every question in notes that has no answer yet, ordered as All orders them.
func Open(notes []sketch.Note) []Question {
	open := []Question{}
	for _, question := range All(notes) {
		if question.Answer == "" {
			open = append(open, question)
		}
	}
	return open
}
