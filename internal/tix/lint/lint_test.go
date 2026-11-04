package lint_test

import (
	"reflect"
	"testing"

	"github.com/theroymind/railyard/internal/tix/backend"
	"github.com/theroymind/railyard/internal/tix/lint"
)

func TestCheck(t *testing.T) {
	clean := backend.LintNote{
		ID: "wait-counter", Path: "docs/decisions/wait-counter.md", Summary: "Turn order is a wait counter.",
		Headings: []string{"Wait counter", "Rejected"}, RequiredSections: []string{"Rejected"},
		Links: []string{"combat-loop", "tasks.base"},
	}
	targets := map[string]bool{"combat-loop": true, "tasks.base": true, "wait-counter": true}
	withChange := func(change func(note *backend.LintNote)) backend.LintNote {
		note := clean
		change(&note)
		return note
	}
	tests := []struct {
		name          string
		note          backend.LintNote
		expectedRules []string
		expectedTexts []string
	}{
		{"A note with its sections, a summary and resolving links is clean", clean, nil, nil},
		{"A missing required section is named", withChange(func(note *backend.LintNote) { note.Headings = []string{"Wait counter"} }),
			[]string{lint.RuleSection}, []string{`missing the section "Rejected"`}},
		{"A section must match its heading exactly", withChange(func(note *backend.LintNote) { note.Headings = []string{"Rejected alternatives"} }),
			[]string{lint.RuleSection}, []string{`missing the section "Rejected"`}},
		{"Each missing section is its own finding", withChange(func(note *backend.LintNote) {
			note.RequiredSections = []string{"Acceptance criteria", "Verification"}
		}), []string{lint.RuleSection, lint.RuleSection}, []string{`missing the section "Acceptance criteria"`, `missing the section "Verification"`}},
		{"A type with no required sections needs none", withChange(func(note *backend.LintNote) {
			note.Headings, note.RequiredSections = nil, nil
		}), nil, nil},
		{"An empty summary is named", withChange(func(note *backend.LintNote) { note.Summary = "" }),
			[]string{lint.RuleSummary}, []string{"has no summary"}},
		{"A link to no note is named", withChange(func(note *backend.LintNote) { note.Links = []string{"combat-loop", "turn-order"} }),
			[]string{lint.RuleLink}, []string{"[[turn-order]] names no note or file in the vault"}},
		{"Findings come section, summary, link", withChange(func(note *backend.LintNote) {
			note.Headings, note.Summary, note.Links = nil, "", []string{"gone"}
		}), []string{lint.RuleSection, lint.RuleSummary, lint.RuleLink}, nil},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			findings := lint.Check(tt.note, targets)

			var rules, texts []string
			for _, found := range findings {
				if found.ID != tt.note.ID || found.Path != tt.note.Path {
					t.Errorf("finding names %s at %s, expected %s at %s", found.ID, found.Path, tt.note.ID, tt.note.Path)
				}
				rules = append(rules, found.Rule)
				texts = append(texts, found.Message)
			}
			if !reflect.DeepEqual(rules, tt.expectedRules) {
				t.Errorf("rules = %v, expected %v", rules, tt.expectedRules)
			}
			if tt.expectedTexts != nil && !reflect.DeepEqual(texts, tt.expectedTexts) {
				t.Errorf("messages = %q, expected %q", texts, tt.expectedTexts)
			}
		})
	}
}
