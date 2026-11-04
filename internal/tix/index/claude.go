package index

import (
	"encoding/json"
	"fmt"
	"slices"
	"strconv"
	"strings"

	"github.com/theroymind/railyard-cli"
	"github.com/theroymind/railyard-cli/strictjson"
)

type Asker interface {
	Ask(prompt string) (string, error)
}

type ClaudeAsker struct {
	Model string
}

type claudeResult struct {
	Type    string `json:"type"`
	IsError bool   `json:"is_error"`
	Result  string `json:"result"`
}

func (asker ClaudeAsker) Ask(prompt string) (string, error) {
	modelArgument := claudeModelOption + asker.Model
	arguments := append(slices.Clone(claudeArguments), modelArgument)
	input := strings.NewReader(prompt)
	output, runErr := cli.RunChild(claudeExecutable, arguments, input)
	var result claudeResult
	if err := json.Unmarshal(output, &result); err != nil {
		if runErr != nil {
			return "", fmt.Errorf("%w: %w", ErrClaudeFailed, runErr)
		}
		return "", fmt.Errorf("%w: %w", ErrClaudeFailed, err)
	}
	if result.Type != claudeResultType {
		quotedType := strconv.Quote(result.Type)
		return "", fmt.Errorf("%w: unknown output type %s", ErrClaudeFailed, quotedType)
	}
	if result.IsError {
		message := strings.TrimSpace(result.Result)
		return "", fmt.Errorf("%w: %s", ErrClaudeFailed, message)
	}
	return result.Result, nil
}

type matchAnswer struct {
	Matches []noteMatch `json:"matches"`
}

type noteMatch struct {
	ID    string `json:"id"`
	Score int    `json:"score"`
}

func RankByModel(asker Asker, notes []Note, query string, types []string, k int) ([]Hit, error) {
	candidates := make(map[string]Note, len(notes))
	var kept []Note
	for _, note := range notes {
		if len(types) > 0 && !slices.Contains(types, note.Type) {
			continue
		}
		candidates[note.ID] = note
		kept = append(kept, note)
	}
	if len(kept) == 0 {
		return nil, nil
	}
	prompt := matchPrompt(kept, query)
	answer, err := asker.Ask(prompt)
	if err != nil {
		return nil, err
	}
	matches, err := parseMatches(answer)
	if err != nil {
		return nil, err
	}
	scores := map[string]int{}
	for _, match := range matches {
		_, isCandidate := candidates[match.ID]
		isBetter := match.Score > scores[match.ID]
		if isCandidate && isBetter {
			scores[match.ID] = match.Score
		}
	}
	hits := make([]Hit, 0, len(scores))
	for id, score := range scores {
		ratio := float64(score) / maximumMatchScore
		hits = append(hits, Hit{Note: candidates[id], Score: ratio})
	}
	return top(hits, k), nil
}

func matchPrompt(notes []Note, query string) string {
	lines := make([]string, len(notes))
	for position, note := range notes {
		summary := note.Summary
		if summary == "" {
			summary = promptMissingSummary
		}
		fields := []string{
			promptIDLabel + note.ID,
			promptTypeLabel + note.Type,
			promptTitleLabel + note.Title,
			promptSummaryLabel + summary,
		}
		joinedFields := strings.Join(fields, promptFieldJoiner)
		lines[position] = promptListItemPrefix + joinedFields
	}
	noteList := strings.Join(lines, textSeparator)
	sections := []string{
		promptInstruction,
		promptQueryLabel + query,
		promptNotesHeading + noteList,
		promptAnswerInstruction,
	}
	return strings.Join(sections, promptSectionSeparator) + textSeparator
}

func parseMatches(answer string) ([]noteMatch, error) {
	start := strings.Index(answer, jsonObjectStart)
	end := strings.LastIndex(answer, jsonObjectEnd)
	if start < 0 || end < start {
		quotedAnswer := strconv.Quote(answer)
		return nil, fmt.Errorf("%w: no JSON object in %s", ErrInvalidAnswer, quotedAnswer)
	}
	object := answer[start : end+len(jsonObjectEnd)]
	var decoded matchAnswer
	if err := strictjson.Unmarshal([]byte(object), &decoded); err != nil {
		return nil, fmt.Errorf("%w: %w", ErrInvalidAnswer, err)
	}
	for _, match := range decoded.Matches {
		if match.Score < minimumMatchScore || match.Score > maximumMatchScore {
			return nil, fmt.Errorf("%w: %s scored %d, expected 1 to 3", ErrInvalidAnswer, match.ID, match.Score)
		}
	}
	return decoded.Matches, nil
}
