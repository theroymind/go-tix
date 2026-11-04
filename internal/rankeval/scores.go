package rankeval

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/theroymind/railyard/internal/contracts/strictjson"
	"github.com/theroymind/railyard/internal/contracts/validation"
)

const problemIDRequired = "is required"

type scoreAnswer struct {
	Scores []cardScore `json:"scores"`
}

type cardScore struct {
	ID    string `json:"id"`
	Score int    `json:"score"`
}

func (answer scoreAnswer) Validate() error {
	var problems validation.Errors
	for index, scored := range answer.Scores {
		field := fmt.Sprintf("scores.%d", index)
		if strings.TrimSpace(scored.ID) == "" {
			problems.Add(field+".id", problemIDRequired)
		}
		if scored.Score < MinimumScore {
			problems.Add(field+".score", validation.TooSmall(MinimumScore))
		}
		if scored.Score > MaximumScore {
			problems.Add(field+".score", validation.TooBig(MaximumScore))
		}
	}
	return problems.Err()
}

// parseScores reads a model's answer as the JSON object it was asked for, tolerating text around it, and
// returns a score per card id. An answer that isn't that object, scores a card not asked about, scores one
// twice, leaves one unscored or steps outside 0 to 3 is an error, and every card then scores 0.
func parseScores(answer string, ids []string) (map[string]int, error) {
	zero := make(map[string]int, len(ids))
	for _, id := range ids {
		zero[id] = MinimumScore
	}
	object, found := jsonObject(answer)
	if !found {
		return zero, fmt.Errorf("%w: no JSON object in %s", ErrInvalidAnswer, strconv.Quote(answer))
	}
	var decoded scoreAnswer
	if err := strictjson.Unmarshal([]byte(object), &decoded); err != nil {
		return zero, fmt.Errorf("%w: %w", ErrInvalidAnswer, err)
	}
	if err := decoded.Validate(); err != nil {
		return zero, fmt.Errorf("%w: %w", ErrInvalidAnswer, err)
	}
	scores := make(map[string]int, len(ids))
	for _, scored := range decoded.Scores {
		if _, asked := zero[scored.ID]; !asked {
			return zero, fmt.Errorf("%w: %w %s", ErrInvalidAnswer, ErrUnknownCard, strconv.Quote(scored.ID))
		}
		if _, seen := scores[scored.ID]; seen {
			return zero, fmt.Errorf("%w: %w %s", ErrInvalidAnswer, ErrDuplicateCard, strconv.Quote(scored.ID))
		}
		scores[scored.ID] = scored.Score
	}
	for _, id := range ids {
		if _, seen := scores[id]; !seen {
			return zero, fmt.Errorf("%w: %w %s", ErrInvalidAnswer, ErrMissingCard, strconv.Quote(id))
		}
	}
	return scores, nil
}

// jsonObject cuts the text from its first opening brace to its last closing brace, so a fenced or prefaced
// answer still parses.
func jsonObject(text string) (string, bool) {
	start := strings.Index(text, jsonObjectStart)
	end := strings.LastIndex(text, jsonObjectEnd)
	if start < 0 || end < start {
		return "", false
	}
	return text[start : end+len(jsonObjectEnd)], true
}
