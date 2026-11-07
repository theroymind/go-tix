// Package questions lists the open questions across a set of notes, each with the note and line that asks it.
package questions

import (
	"cmp"
	"slices"

	"github.com/theroymind/go-tix/contracts/sketch"
)

// Question is one open question: the path of the note that asks it, its 1-based line in that note, and its text.
type Question struct {
	Path string `json:"path"`
	Line int    `json:"line"`
	Text string `json:"text"`
}

// Open returns every question in notes that has no answer yet, ordered by note path and then by line.
func Open(notes []sketch.Note) []Question {
	open := []Question{}
	for _, value := range notes {
		for _, question := range value.Questions {
			if question.Answer == "" {
				open = append(open, Question{Path: value.Path, Line: question.Line, Text: question.Text})
			}
		}
	}
	slices.SortFunc(open, func(a, b Question) int {
		return cmp.Or(cmp.Compare(a.Path, b.Path), cmp.Compare(a.Line, b.Line))
	})
	return open
}
