// Package lint provides the structural checks tix lint runs on a vault's notes: the sections a note's type requires,
// a summary, and links that resolve.
package lint

import (
	"slices"
	"strconv"

	"github.com/theroymind/railyard/internal/tix/backend"
)

// Rule names, in the order Check applies them.
const (
	RuleSection = "section"
	RuleSummary = "summary"
	RuleLink    = "link"
)

const (
	messageMissingSection = "missing the section "
	messageEmptySummary   = "has no summary"
	messageLinkPrefix     = "[["
	messageLinkSuffix     = "]] names no note or file in the vault"
)

// Finding is one rule a note breaks.
type Finding struct {
	ID      string
	Path    string
	Rule    string
	Message string
}

// Check returns every rule the note breaks: each required section it lacks, an empty summary, and each link that
// names none of targets, in that order.
func Check(note backend.LintNote, targets map[string]bool) []Finding {
	var findings []Finding
	for _, heading := range note.RequiredSections {
		if !slices.Contains(note.Headings, heading) {
			findings = append(findings, finding(note, RuleSection, messageMissingSection+strconv.Quote(heading)))
		}
	}
	if note.Summary == "" {
		findings = append(findings, finding(note, RuleSummary, messageEmptySummary))
	}
	for _, link := range note.Links {
		if !targets[link] {
			findings = append(findings, finding(note, RuleLink, messageLinkPrefix+link+messageLinkSuffix))
		}
	}
	return findings
}

func finding(note backend.LintNote, rule, message string) Finding {
	return Finding{ID: note.ID, Path: note.Path, Rule: rule, Message: message}
}
