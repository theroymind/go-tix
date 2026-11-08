// Package lint provides the structural checks tix lint runs on a vault's notes: the sections a note's type requires,
// sections rewritten from their template, a summary, and links that resolve.
package lint

import (
	"slices"
	"strconv"

	"github.com/theroymind/go-tix/internal/tix/backend"
)

// Rule names, in the order Check applies them.
const (
	RuleSection  = "section"
	RuleTemplate = "template"
	RuleSummary  = "summary"
	RuleLink     = "link"
)

const (
	messageMissingSection = "missing the section "
	messageTemplateText   = " still holds the template's text"
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

// Check returns every rule the note breaks: each required section it lacks, each section still holding its
// template's text, an empty summary, and each link that names none of targets, in that order.
func Check(note backend.LintNote, targets map[string]bool) []Finding {
	var findings []Finding
	for _, heading := range note.RequiredSections {
		if !slices.Contains(note.Headings, heading) {
			findings = append(findings, finding(note, RuleSection, messageMissingSection+strconv.Quote(heading)))
		}
	}
	for _, heading := range note.TemplateSections {
		findings = append(findings, finding(note, RuleTemplate, strconv.Quote(heading)+messageTemplateText))
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
