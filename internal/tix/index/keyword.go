package index

import (
	"slices"
	"strings"
	"unicode"
)

// RankKeywords scores every note whose type is in types, or every note when types is empty, by the share of the
// query's words found in its title and summary, and returns the top k that match at least one word, best first,
// ties by id. Matching ignores case and punctuation and needs no index.
func RankKeywords(notes []Note, query string, types []string, k int) []Hit {
	words := Words(query)
	var hits []Hit
	for _, note := range notes {
		if len(types) > 0 && !slices.Contains(types, note.Type) {
			continue
		}
		score := keywordScore(words, Words(note.Text()))
		if score > 0 {
			hits = append(hits, Hit{Note: note, Score: score})
		}
	}
	return top(hits, k)
}

// keywordScore is the share of the distinct query words that appear among the text's words.
func keywordScore(queryWords, textWords []string) float64 {
	if len(queryWords) == 0 {
		return 0
	}
	distinct := map[string]bool{}
	for _, word := range queryWords {
		distinct[word] = true
	}
	found := 0
	for word := range distinct {
		if slices.Contains(textWords, word) {
			found++
		}
	}
	return float64(found) / float64(len(distinct))
}

// Words splits text into lower-case words at anything that is not a letter or a digit.
func Words(text string) []string {
	isSeparator := func(character rune) bool { return !unicode.IsLetter(character) && !unicode.IsDigit(character) }
	return strings.FieldsFunc(strings.ToLower(text), isSeparator)
}
