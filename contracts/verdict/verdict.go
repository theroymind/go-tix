// Package verdict provides the Verdict contract a judge returns on a draft vault note: accept it, or refuse it as a
// duplicate of a note or a conflict with one, with the reason.
package verdict

import (
	"slices"
	"strings"

	"github.com/theroymind/railyard/internal/contracts/validation"
)

const (
	Accept = "accept"
	Refuse = "refuse"
)

const (
	FieldVerdict       = "verdict"
	FieldDuplicateOf   = "duplicate_of"
	FieldConflictsWith = "conflicts_with"
	FieldReason        = "reason"
)

const (
	problemRequired        = "is required"
	problemBlank           = "must be empty or a note id"
	problemAcceptNamesNote = "must be empty when the verdict is accept"
	problemRefuseNamesNone = "must name a note when the verdict is refuse, here or in " + FieldConflictsWith
)

// Verdicts are the values Verdict.Verdict takes.
var Verdicts = []string{Accept, Refuse}

// Verdict is a judge's answer on one draft note. DuplicateOf names the note the draft restates and ConflictsWith the
// note it contradicts, each "" for none. A refusal names at least one of them, and an acceptance neither.
type Verdict struct {
	Verdict       string `json:"verdict"`
	DuplicateOf   string `json:"duplicate_of"`
	ConflictsWith string `json:"conflicts_with"`
	Reason        string `json:"reason"`
}

// Schema is the JSON Schema of a Verdict, for a model that takes one to shape its answer.
const Schema = `{"type":"object","additionalProperties":false,` +
	`"required":["verdict","duplicate_of","conflicts_with","reason"],` +
	`"properties":{"verdict":{"type":"string","enum":["accept","refuse"]},` +
	`"duplicate_of":{"type":"string"},"conflicts_with":{"type":"string"},"reason":{"type":"string"}}}`

func (value Verdict) Validate() error {
	var problems validation.Errors
	if !slices.Contains(Verdicts, value.Verdict) {
		problems.Add(FieldVerdict, validation.InvalidOption(Verdicts))
	}
	if strings.TrimSpace(value.Reason) == "" {
		problems.Add(FieldReason, problemRequired)
	}
	addBlank(&problems, FieldDuplicateOf, value.DuplicateOf)
	addBlank(&problems, FieldConflictsWith, value.ConflictsWith)
	namesNote := value.DuplicateOf != "" || value.ConflictsWith != ""
	if value.Verdict == Accept && namesNote {
		problems.Add(FieldDuplicateOf, problemAcceptNamesNote)
	}
	if value.Verdict == Refuse && !namesNote {
		problems.Add(FieldDuplicateOf, problemRefuseNamesNone)
	}
	return problems.Err()
}

// IsRefusal reports whether the judge refused the draft.
func (value Verdict) IsRefusal() bool {
	return value.Verdict == Refuse
}

func addBlank(problems *validation.Errors, field, id string) {
	if id != "" && strings.TrimSpace(id) != id {
		problems.Add(field, problemBlank)
	}
}
