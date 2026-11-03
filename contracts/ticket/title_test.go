package ticket

import (
	"errors"
	"strings"
	"testing"
)

func TestNormalizeTitle(t *testing.T) {
	tests := []struct {
		name     string
		text     string
		expected string
	}{
		{"Joins wrapped lines with one space", "Golem guards\n  twice", "Golem guards twice"},
		{"Replaces carriage returns and tabs", "Golem\r\n\tguards\ttwice", "Golem guards twice"},
		{"Collapses runs of spaces", "Golem   guards    twice", "Golem guards twice"},
		{"Trims the ends", "  Golem guards twice \n", "Golem guards twice"},
		{"Keeps a title of exactly the maximum length", strings.Repeat("a", TitleMaximumLength), strings.Repeat("a", TitleMaximumLength)},
		{"Counts characters, not bytes", strings.Repeat("é", TitleMaximumLength), strings.Repeat("é", TitleMaximumLength)},
		{"Leaves an empty title empty", "  \n\t ", ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := NormalizeTitle(tt.text)
			if err != nil {
				t.Fatalf("NormalizeTitle(%q) error = %v", tt.text, err)
			}
			if got != tt.expected {
				t.Errorf("NormalizeTitle(%q) = %q, expected %q", tt.text, got, tt.expected)
			}
		})
	}
}

func TestNormalizeTitleRejectsLongTitles(t *testing.T) {
	tests := []struct {
		name string
		text string
	}{
		{"One character over the maximum", strings.Repeat("a", TitleMaximumLength+1)},
		{"Over the maximum after flattening", strings.Repeat("ab\n", 25)},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := NormalizeTitle(tt.text)
			if !errors.Is(err, ErrTitleTooLong) {
				t.Fatalf("NormalizeTitle error = %v, expected ErrTitleTooLong", err)
			}
			if !strings.Contains(err.Error(), "72") {
				t.Errorf("error %q does not name the limit", err)
			}
		})
	}
}
