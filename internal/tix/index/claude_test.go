package index_test

import (
	"errors"
	"reflect"
	"strings"
	"testing"

	"github.com/theroymind/go-tix/internal/tix/index"
)

type fakeAsker struct {
	answer string
	err    error
	asked  *string
}

func (asker fakeAsker) Ask(prompt string) (string, error) {
	if asker.asked != nil {
		*asker.asked = prompt
	}
	return asker.answer, asker.err
}

func TestRankByModel(t *testing.T) {
	notes := []index.Note{
		{ID: "BS-1", Type: "task", Title: "Add login", Summary: "Users sign in with email"},
		{ID: "BS-2", Type: "task", Title: "Fix logout"},
		{ID: "ADR-1", Type: "decision", Title: "Use sessions"},
	}
	failure := errors.New("claude is down")
	tests := []struct {
		name        string
		answer      string
		askErr      error
		types       []string
		k           int
		expectedIDs []string
		expectedErr error
	}{
		{"Matches come back best first, ties by id",
			`{"matches":[{"id":"BS-2","score":2},{"id":"ADR-1","score":3},{"id":"BS-1","score":2}]}`,
			nil, nil, 10, []string{"ADR-1", "BS-1", "BS-2"}, nil},
		{"Only the top k are kept", `{"matches":[{"id":"BS-2","score":1},{"id":"BS-1","score":3}]}`,
			nil, nil, 1, []string{"BS-1"}, nil},
		{"Made-up ids are dropped", `{"matches":[{"id":"BS-9","score":3},{"id":"BS-1","score":1}]}`,
			nil, nil, 10, []string{"BS-1"}, nil},
		{"Notes outside the types are dropped even when scored", `{"matches":[{"id":"ADR-1","score":3},{"id":"BS-1","score":1}]}`,
			nil, []string{"task"}, 10, []string{"BS-1"}, nil},
		{"Text around the object is tolerated", "Here you go:\n```json\n{\"matches\":[{\"id\":\"BS-2\",\"score\":2}]}\n```",
			nil, nil, 10, []string{"BS-2"}, nil},
		{"No matches is an empty result", `{"matches":[]}`, nil, nil, 10, []string{}, nil},
		{"An answer with no object is invalid", "nothing matches", nil, nil, 10, nil, index.ErrInvalidAnswer},
		{"A score outside 1 to 3 is invalid", `{"matches":[{"id":"BS-1","score":5}]}`, nil, nil, 10, nil, index.ErrInvalidAnswer},
		{"An unknown key is invalid", `{"matches":[],"extra":1}`, nil, nil, 10, nil, index.ErrInvalidAnswer},
		{"The asker's error is returned", "", failure, nil, 10, nil, failure},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			asker := fakeAsker{answer: tt.answer, err: tt.askErr}

			hits, err := index.RankByModel(asker, notes, "login", tt.types, tt.k)

			if !errors.Is(err, tt.expectedErr) {
				t.Fatalf("error = %v, expected %v", err, tt.expectedErr)
			}
			if tt.expectedErr != nil {
				return
			}
			ids := []string{}
			for _, hit := range hits {
				ids = append(ids, hit.Note.ID)
			}
			if !reflect.DeepEqual(ids, tt.expectedIDs) {
				t.Errorf("ids = %v, expected %v", ids, tt.expectedIDs)
			}
		})
	}
}

func TestRankByModelPrompt(t *testing.T) {
	notes := []index.Note{
		{ID: "BS-1", Type: "task", Title: "Add login", Summary: "Users sign in with email"},
		{ID: "ADR-1", Type: "decision", Title: "Use sessions"},
	}
	var prompt string
	asker := fakeAsker{answer: `{"matches":[{"id":"BS-1","score":3}]}`, asked: &prompt}

	hits, err := index.RankByModel(asker, notes, "sign in", []string{"task"}, 10)
	if err != nil {
		t.Fatal(err)
	}

	for _, mention := range []string{"Query: sign in", "id: BS-1 | type: task | title: Add login | summary: Users sign in with email"} {
		if !strings.Contains(prompt, mention) {
			t.Errorf("prompt = %q, want it to mention %q", prompt, mention)
		}
	}
	if strings.Contains(prompt, "ADR-1") {
		t.Errorf("prompt = %q, expected it to leave out notes outside the types", prompt)
	}
	if len(hits) != 1 || hits[0].Score != 1 {
		t.Errorf("hits = %v, expected BS-1 scoring 1", hits)
	}
}
