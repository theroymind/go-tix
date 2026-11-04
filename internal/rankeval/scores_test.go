package rankeval

import (
	"errors"
	"maps"
	"testing"
)

func TestParseScores(t *testing.T) {
	ids := []string{"alpha", "beta"}
	zero := map[string]int{"alpha": 0, "beta": 0}
	tests := []struct {
		name          string
		answer        string
		expected      map[string]int
		expectedError error
	}{
		{"The object asked for", `{"scores":[{"id":"alpha","score":3},{"id":"beta","score":1}]}`, map[string]int{"alpha": 3, "beta": 1}, nil},
		{"The object inside a code fence with prose around it", "Here you go:\n```json\n{\"scores\":[{\"id\":\"alpha\",\"score\":0},{\"id\":\"beta\",\"score\":2}]}\n```\nDone.", map[string]int{"alpha": 0, "beta": 2}, nil},
		{"No object at all", "I cannot rank these.", zero, ErrInvalidAnswer},
		{"Broken JSON", `{"scores":[{"id":"alpha","score":3}`, zero, ErrInvalidAnswer},
		{"An unknown field", `{"scores":[{"id":"alpha","score":3,"reason":"x"},{"id":"beta","score":1}]}`, zero, ErrInvalidAnswer},
		{"A score above 3", `{"scores":[{"id":"alpha","score":4},{"id":"beta","score":1}]}`, zero, ErrInvalidAnswer},
		{"A score below 0", `{"scores":[{"id":"alpha","score":-1},{"id":"beta","score":1}]}`, zero, ErrInvalidAnswer},
		{"A card that was not asked about", `{"scores":[{"id":"alpha","score":3},{"id":"beta","score":1},{"id":"gamma","score":2}]}`, zero, ErrUnknownCard},
		{"A card scored twice", `{"scores":[{"id":"alpha","score":3},{"id":"alpha","score":1},{"id":"beta","score":1}]}`, zero, ErrDuplicateCard},
		{"A card left unscored", `{"scores":[{"id":"alpha","score":3}]}`, zero, ErrMissingCard},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			scores, err := parseScores(tt.answer, ids)

			if !errors.Is(err, tt.expectedError) {
				t.Fatalf("parseScores() error = %v, want %v", err, tt.expectedError)
			}
			if !maps.Equal(scores, tt.expected) {
				t.Errorf("parseScores() = %v, want %v", scores, tt.expected)
			}
		})
	}
}
