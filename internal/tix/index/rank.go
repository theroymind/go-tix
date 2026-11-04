package index

import (
	"fmt"
	"math"
	"slices"
	"strings"
)

// Hit is one entry ranked against a query, with its score, higher being closer.
type Hit struct {
	Note  Note
	Score float64
}

// Rank scores every entry whose type is in types, or every entry when types is empty, by cosine similarity to
// query, and returns the top k, best first, with ties broken by id.
func Rank(value Index, query []float32, types []string, k int) ([]Hit, error) {
	var hits []Hit
	for _, entry := range value.Entries {
		if len(types) > 0 && !slices.Contains(types, entry.Type) {
			continue
		}
		score, err := Cosine(query, entry.Vector)
		if err != nil {
			return nil, fmt.Errorf("%s: %w", entry.ID, err)
		}
		hits = append(hits, Hit{Note: entry.note(), Score: score})
	}
	return top(hits, k), nil
}

// Cosine is the cosine similarity of two vectors of the same length: 1 for the same direction, 0 for orthogonal
// ones, -1 for opposite ones. A zero vector scores 0 against everything.
func Cosine(left, right []float32) (float64, error) {
	if len(left) == 0 || len(right) == 0 {
		return 0, ErrEmptyVector
	}
	if len(left) != len(right) {
		return 0, fmt.Errorf("%w: %d and %d", ErrDimensionMismatch, len(left), len(right))
	}
	var dot, leftSquares, rightSquares float64
	for position := range left {
		dot += float64(left[position]) * float64(right[position])
		leftSquares += float64(left[position]) * float64(left[position])
		rightSquares += float64(right[position]) * float64(right[position])
	}
	if leftSquares == 0 || rightSquares == 0 {
		return 0, nil
	}
	return dot / (math.Sqrt(leftSquares) * math.Sqrt(rightSquares)), nil
}

// top sorts hits best first, ties by id, and keeps the first k.
func top(hits []Hit, k int) []Hit {
	slices.SortStableFunc(hits, func(left, right Hit) int {
		if left.Score != right.Score {
			if left.Score > right.Score {
				return -1
			}
			return 1
		}
		return strings.Compare(left.Note.ID, right.Note.ID)
	})
	if k < minimumTopK {
		k = minimumTopK
	}
	if len(hits) > k {
		hits = hits[:k]
	}
	if hits == nil {
		return []Hit{}
	}
	return hits
}
