package ticket

import (
	"errors"
	"slices"
	"strings"
	"testing"
)

const (
	testPrefix = "ry"
	byteRange  = 256
)

type sequenceReader struct {
	batches [][]byte
	calls   []int
}

func (reader *sequenceReader) Read(buffer []byte) (int, error) {
	reader.calls = append(reader.calls, len(buffer))
	batch := reader.batches[min(len(reader.calls), len(reader.batches))-1]
	return copy(buffer, batch), nil
}

func suffixOf(indexes []byte) string {
	var suffix strings.Builder
	for _, index := range indexes {
		suffix.WriteByte(IDCharacters[index])
	}
	return suffix.String()
}

func indexesOf(index func(position int) int) []byte {
	indexes := make([]byte, IDSuffixLength)
	for position := range indexes {
		indexes[position] = byte(index(position))
	}
	return indexes
}

func TestGenerateID(t *testing.T) {
	firstIndexes := indexesOf(func(position int) int { return position })
	lastIndexes := indexesOf(func(position int) int { return len(IDCharacters) - 1 - position })
	wrapped := indexesOf(func(position int) int { return position + len(IDCharacters)*position })
	highest := indexesOf(func(int) int { return byteRange - 1 })
	lastCharacter := indexesOf(func(int) int { return len(IDCharacters) - 1 })
	firstID := testPrefix + IDSeparator + suffixOf(firstIndexes)

	tests := []struct {
		name          string
		batches       [][]byte
		isTaken       func(string) bool
		expectedID    string
		expectedError error
		expectedCalls int
	}{
		{"Maps each byte onto the ID characters modulo their count", [][]byte{wrapped}, func(string) bool { return false }, firstID, nil, 1},
		{"Maps the highest byte onto the last ID character", [][]byte{highest}, func(string) bool { return false }, testPrefix + IDSeparator + suffixOf(lastCharacter), nil, 1},
		{"Retries when the ID is taken", [][]byte{firstIndexes, lastIndexes}, func(id string) bool { return id == firstID }, testPrefix + IDSeparator + suffixOf(lastIndexes), nil, 2},
		{"Fails after the maximum number of attempts", [][]byte{firstIndexes}, func(string) bool { return true }, "", ErrIDSpaceExhausted, MaximumIDAttempts},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			reader := &sequenceReader{batches: tt.batches}

			id, err := GenerateID(testPrefix, reader, tt.isTaken)

			if !errors.Is(err, tt.expectedError) {
				t.Fatalf("GenerateID() error = %v, want %v", err, tt.expectedError)
			}
			if id != tt.expectedID {
				t.Errorf("GenerateID() = %q, want %q", id, tt.expectedID)
			}
			if tt.expectedError == nil && !IsValidID(id) {
				t.Errorf("GenerateID() = %q, which is not a valid ID", id)
			}
			wantCalls := slices.Repeat([]int{IDSuffixLength}, tt.expectedCalls)
			if !slices.Equal(reader.calls, wantCalls) {
				t.Errorf("random reads = %v, want %v", reader.calls, wantCalls)
			}
		})
	}
}

func TestIsValidID(t *testing.T) {
	tests := []struct {
		name     string
		id       string
		expected bool
	}{
		{"A prefixed ID", "ry-k3f9", true},
		{"A vault slug", "goal-scoring", true},
		{"A single character", "a", true},
		{"A leading digit", "9lives", true},
		{"The empty string", "", false},
		{"A leading hyphen", "-x", false},
		{"An uppercase letter", "Goal", false},
		{"An underscore", "a_b", false},
		{"A path segment", "../ry-k3f9", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := IsValidID(tt.id); got != tt.expected {
				t.Errorf("IsValidID(%q) = %v, want %v", tt.id, got, tt.expected)
			}
		})
	}
}
