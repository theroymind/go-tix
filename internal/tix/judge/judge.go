// Package judge provides the judge tix judge asks about a draft vault note: the prompt that sets the draft beside its
// neighbours, the Judge that answers it through claude -p, and the check that a verdict names only those neighbours.
package judge

import (
	"errors"
	"fmt"
	"slices"
	"strconv"
	"strings"

	"github.com/theroymind/go-tix/contracts/verdict"
)

var (
	ErrUnknownNote       = errors.New("names a note that isn't among the draft's neighbours")
	ErrNamesDraft        = errors.New("names the draft itself")
	ErrClaudeFailed      = errors.New("claude reported an error")
	ErrUnknownClaudeType = errors.New("unknown claude output type")
)

// Judge answers a prompt about one draft with a verdict that has validated.
type Judge interface {
	Judge(prompt string) (verdict.Verdict, error)
}

// Note is a draft or one of its neighbours: its id, note type, status ("" for none), and its full text.
type Note struct {
	ID     string
	Type   string
	Status string
	Text   string
}

// CheckNames fails when the verdict names the draft itself or a note other than one of its neighbours, the only
// notes the judge was shown.
func CheckNames(value verdict.Verdict, draft string, neighbours []Note) error {
	for _, named := range []string{value.DuplicateOf, value.ConflictsWith} {
		if named == "" {
			continue
		}
		if named == draft {
			return fmt.Errorf("verdict %s %w", strconv.Quote(named), ErrNamesDraft)
		}
		isNeighbour := slices.ContainsFunc(neighbours, func(neighbour Note) bool { return neighbour.ID == named })
		if !isNeighbour {
			return fmt.Errorf("verdict %s %w", strconv.Quote(named), ErrUnknownNote)
		}
	}
	return nil
}

// Prompt asks the judge about draft beside its neighbours, each under a heading naming its id, type and status.
func Prompt(draft Note, neighbours []Note) string {
	var builder strings.Builder
	builder.WriteString(instructions)
	builder.WriteString(draftHeading)
	writeNote(&builder, draft)
	builder.WriteString(neighboursHeading)
	if len(neighbours) == 0 {
		builder.WriteString(noNeighbours)
	}
	for _, neighbour := range neighbours {
		writeNote(&builder, neighbour)
	}
	return builder.String()
}

func writeNote(builder *strings.Builder, note Note) {
	status := note.Status
	if status == "" {
		status = noStatus
	}
	fmt.Fprintf(builder, noteHeading, note.ID, note.Type, status)
	builder.WriteString(strings.TrimSpace(note.Text))
	builder.WriteString(noteEnd)
}
