package store

import (
	"errors"
	"strconv"
	"strings"
	"testing"

	"github.com/theroymind/railyard/internal/contracts/ticket"
)

const testPrefix = "bs"

func qualified(suffix string) string {
	return testPrefix + ticket.IDSeparator + suffix
}

func TestResolveReference(t *testing.T) {
	ids := []string{qualified("k3f9"), qualified("k3aa"), qualified("2m7q"), "ab-k3f9"}

	tests := []struct {
		name      string
		reference string
		ids       []string
		expected  string
	}{
		{"Resolves an exact ID", qualified("2m7q"), ids, qualified("2m7q")},
		{"Resolves an exact ID under another prefix", "ab-k3f9", ids, "ab-k3f9"},
		{"Resolves a suffix with the prefix prepended", "k3f9", ids, qualified("k3f9")},
		{"Resolves a unique prefix of a suffix", "2m", ids, qualified("2m7q")},
		{"Resolves a unique prefix of a full ID", qualified("k3f"), ids, qualified("k3f9")},
		{"Prefers an exact match over a longer prefix match", "k3f9", []string{qualified("k3f9"), qualified("k3f9") + "x"}, qualified("k3f9")},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ResolveReference(tt.reference, testPrefix, tt.ids)
			if err != nil {
				t.Fatalf("ResolveReference(%q) error = %v", tt.reference, err)
			}
			if got != tt.expected {
				t.Errorf("ResolveReference(%q) = %q, want %q", tt.reference, got, tt.expected)
			}
		})
	}
}

func TestResolveReferenceFails(t *testing.T) {
	ids := []string{qualified("k3f9"), qualified("k3aa"), qualified("2m7q"), "ab-k3f9"}

	tests := []struct {
		name            string
		reference       string
		expectedError   error
		expectedMessage string
	}{
		{"Fails on an ambiguous reference and lists the matches", "k3", ErrAmbiguous, strconv.Quote("k3") + " is ambiguous: " + qualified("k3aa") + ", " + qualified("k3f9")},
		{"Fails when no suffix matches", "zz", ErrNoMatch, "no ticket matches " + strconv.Quote("zz")},
		{"Fails when no full ID matches", qualified("zz"), ErrNoMatch, "no ticket matches " + strconv.Quote(qualified("zz"))},
		{"Fails on an empty reference", "", ErrNoMatch, "no ticket matches " + strconv.Quote("")},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := ResolveReference(tt.reference, testPrefix, ids)

			if !errors.Is(err, tt.expectedError) {
				t.Fatalf("ResolveReference(%q) error = %v, want %v", tt.reference, err, tt.expectedError)
			}
			if !strings.Contains(err.Error(), tt.expectedMessage) {
				t.Errorf("ResolveReference(%q) error = %q, want it to contain %q", tt.reference, err, tt.expectedMessage)
			}
		})
	}
}
