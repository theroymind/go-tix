package index_test

import (
	"errors"
	"math"
	"reflect"
	"testing"

	"github.com/theroymind/railyard/internal/tix/index"
)

const tolerance = 1e-9

func TestCosine(t *testing.T) {
	tests := []struct {
		name        string
		left, right []float32
		expected    float64
		expectedErr error
	}{
		{"The same direction scores 1", []float32{1, 2, 3}, []float32{2, 4, 6}, 1, nil},
		{"Orthogonal vectors score 0", []float32{1, 0}, []float32{0, 1}, 0, nil},
		{"Opposite vectors score -1", []float32{1, 1}, []float32{-1, -1}, -1, nil},
		{"A 45 degree angle scores the square root of a half", []float32{1, 0}, []float32{1, 1}, math.Sqrt(0.5), nil},
		{"A zero vector scores 0", []float32{0, 0}, []float32{1, 1}, 0, nil},
		{"Lengths must match", []float32{1, 0}, []float32{1, 0, 0}, 0, index.ErrDimensionMismatch},
		{"An empty vector is an error", nil, []float32{1}, 0, index.ErrEmptyVector},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			score, err := index.Cosine(tt.left, tt.right)

			if !errors.Is(err, tt.expectedErr) {
				t.Fatalf("error = %v, expected %v", err, tt.expectedErr)
			}
			if math.Abs(score-tt.expected) > tolerance {
				t.Errorf("score = %v, expected %v", score, tt.expected)
			}
		})
	}
}

// rankedIndex holds four notes along fixed directions, so that a query along one axis orders them by angle.
var rankedIndex = index.Index{SchemaVersion: index.SchemaVersion, Model: "fixed", Entries: []index.Entry{
	{ID: "diagonal", Type: "task", Title: "Diagonal", Vector: []float32{1, 1}},
	{ID: "sideways", Type: "idea", Title: "Sideways", Vector: []float32{0, 1}},
	{ID: "straight", Type: "task", Title: "Straight", Summary: "Right along.", Vector: []float32{3, 0}},
	{ID: "backward", Type: "task", Title: "Backward", Vector: []float32{-1, 0}},
}}

func TestRank(t *testing.T) {
	tests := []struct {
		name     string
		query    []float32
		types    []string
		k        int
		expected []string
	}{
		{"Orders by angle to the query", []float32{1, 0}, nil, 10, []string{"straight", "diagonal", "sideways", "backward"}},
		{"Keeps the top k", []float32{1, 0}, nil, 2, []string{"straight", "diagonal"}},
		{"Filters by type", []float32{1, 0}, []string{"idea"}, 10, []string{"sideways"}},
		{"Breaks a tie by id", []float32{1, 1}, nil, 10, []string{"diagonal", "sideways", "straight", "backward"}},
		{"A k below one keeps one", []float32{1, 0}, nil, 0, []string{"straight"}},
		{"No type matching gives an empty list", []float32{1, 0}, []string{"spec"}, 10, []string{}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			hits, err := index.Rank(rankedIndex, tt.query, tt.types, tt.k)
			if err != nil {
				t.Fatal(err)
			}

			ids := []string{}
			for _, hit := range hits {
				ids = append(ids, hit.Note.ID)
			}
			if !reflect.DeepEqual(ids, tt.expected) {
				t.Errorf("ranked %v, expected %v", ids, tt.expected)
			}
		})
	}
}

func TestRankScoresAndNotes(t *testing.T) {
	hits, err := index.Rank(rankedIndex, []float32{1, 0}, nil, 1)
	if err != nil {
		t.Fatal(err)
	}

	expected := index.Hit{Note: index.Note{ID: "straight", Type: "task", Title: "Straight", Summary: "Right along."}, Score: 1}
	if len(hits) != 1 || !reflect.DeepEqual(hits[0], expected) {
		t.Errorf("hits = %+v, expected [%+v]", hits, expected)
	}
}

func TestRankReportsAVectorOfAnotherLength(t *testing.T) {
	_, err := index.Rank(rankedIndex, []float32{1, 0, 0}, nil, 10)

	if !errors.Is(err, index.ErrDimensionMismatch) {
		t.Errorf("error = %v, expected %v", err, index.ErrDimensionMismatch)
	}
}

var keywordNotes = []index.Note{
	{ID: "enemy-choice", Type: "task", Title: "Enemy choice", Summary: "How an enemy picks its target."},
	{ID: "enemy-goals", Type: "spec", Title: "Enemy goals", Summary: "What each enemy wants from a fight."},
	{ID: "turn-order", Type: "task", Title: "Turn order", Summary: "Who acts first."},
}

func TestRankKeywords(t *testing.T) {
	tests := []struct {
		name     string
		query    string
		types    []string
		k        int
		expected []string
	}{
		{"Scores by the share of query words found, ties by id", "enemy picks who to hit", nil, 10, []string{"enemy-choice", "enemy-goals", "turn-order"}},
		{"Ignores case and punctuation", "ENEMY, target!", nil, 10, []string{"enemy-choice", "enemy-goals"}},
		{"Leaves out notes matching no word", "swords", nil, 10, []string{}},
		{"Filters by type", "enemy", []string{"spec"}, 10, []string{"enemy-goals"}},
		{"Keeps the top k", "enemy", nil, 1, []string{"enemy-choice"}},
		{"An empty query matches nothing", "", nil, 10, []string{}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			hits := index.RankKeywords(keywordNotes, tt.query, tt.types, tt.k)

			ids := []string{}
			for _, hit := range hits {
				ids = append(ids, hit.Note.ID)
			}
			if !reflect.DeepEqual(ids, tt.expected) {
				t.Errorf("ranked %v, expected %v", ids, tt.expected)
			}
		})
	}
}

func TestRankKeywordsScore(t *testing.T) {
	hits := index.RankKeywords(keywordNotes, "enemy picks who to hit", nil, 1)

	expectedScore := 2.0 / 5.0
	if len(hits) != 1 || math.Abs(hits[0].Score-expectedScore) > tolerance {
		t.Errorf("hits = %+v, expected one hit scoring %v", hits, expectedScore)
	}
}
