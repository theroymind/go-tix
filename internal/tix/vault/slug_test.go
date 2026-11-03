package vault

import (
	"regexp"
	"testing"

	"github.com/theroymind/railyard/internal/contracts/ticket"
)

func TestSlugify(t *testing.T) {
	tests := []struct {
		name     string
		title    string
		expected string
	}{
		{"Joins words with hyphens", "Golem guards twice", "golem-guards-twice"},
		{"Lowercases", "Golem GUARDS Twice", "golem-guards-twice"},
		{"Turns punctuation into one hyphen", "tix: flatten, then cap!", "tix-flatten-then-cap"},
		{"Collapses repeated separators", "a -- b__c", "a-b-c"},
		{"Trims leading and trailing junk", "  ...Golem guards?! ", "golem-guards"},
		{"Keeps digits", "Task 17 v2", "task-17-v2"},
		{"Drops non-ASCII letters", "Café über", "caf-ber"},
		{"Is empty for an all-punctuation title", "?!--...", ""},
	}
	idExpression := regexp.MustCompile(ticket.IDPattern)
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := Slugify(tt.title)
			if got != tt.expected {
				t.Errorf("Slugify(%q) = %q, expected %q", tt.title, got, tt.expected)
			}
			if got != "" && !idExpression.MatchString(got) {
				t.Errorf("Slugify(%q) = %q, which does not match %s", tt.title, got, ticket.IDPattern)
			}
		})
	}
}
