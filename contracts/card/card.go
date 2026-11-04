// Package card provides the Card contract that tix context prints, one per note a walk from a ticket reaches.
package card

import (
	"fmt"
	"slices"
	"strings"

	"github.com/theroymind/railyard/internal/contracts/validation"
)

// Card is one note a walk reached: its id, note type and title, its one-line summary, the hops it took to reach,
// the relation of the hop that reached it, the commits that changed its code, newest first, and the files it
// expects to touch. HasSummary is false for a note without a summary, whose Summary is then empty.
type Card struct {
	ID         string   `json:"id"`
	Type       string   `json:"type"`
	Title      string   `json:"title"`
	Summary    string   `json:"summary"`
	HasSummary bool     `json:"has_summary"`
	Distance   int      `json:"distance"`
	Relation   string   `json:"relation"`
	Commits    []string `json:"commits"`
	Files      []string `json:"files"`
}

// Relations are the hops a walk follows, strongest first: the start itself, then a note's parent, the notes it
// depends on, the notes its text links to, and the notes whose text links to it.
var Relations = []string{RelationStart, RelationParent, RelationDependency, RelationWikilink, RelationBacklink}

func (value Card) Validate() error {
	var problems validation.Errors
	if strings.TrimSpace(value.ID) == "" {
		problems.Add(FieldID, problemRequired)
	}
	if strings.TrimSpace(value.Type) == "" {
		problems.Add(FieldType, problemRequired)
	}
	if strings.TrimSpace(value.Title) == "" {
		problems.Add(FieldTitle, problemRequired)
	}
	if value.HasSummary != (strings.TrimSpace(value.Summary) != "") {
		problems.Add(FieldHasSummary, problemSummaryMismatch)
	}
	if value.Distance < minimumDistance {
		problems.Add(FieldDistance, validation.TooSmall(minimumDistance))
	}
	if !slices.Contains(Relations, value.Relation) {
		problems.Add(FieldRelation, problemRelationUnknown)
	}
	if (value.Relation == RelationStart) != (value.Distance == minimumDistance) {
		problems.Add(FieldRelation, problemStartMismatch)
	}
	addEmptyItems(&problems, FieldCommits, value.Commits)
	addEmptyItems(&problems, FieldFiles, value.Files)
	return problems.Err()
}

func addEmptyItems(problems *validation.Errors, field string, items []string) {
	for index, item := range items {
		if strings.TrimSpace(item) == "" {
			problems.Add(fmt.Sprintf("%s.%d", field, index), problemEmptyItem)
		}
	}
}
