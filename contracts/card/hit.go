package card

import (
	"strings"

	"github.com/theroymind/railyard/internal/contracts/validation"
)

// Hit is one note tix search ranked against a query: its id, note type, title and one-line summary, and its score,
// higher being closer. HasSummary is false for a note without a summary, whose Summary is then empty.
type Hit struct {
	ID         string  `json:"id"`
	Type       string  `json:"type"`
	Title      string  `json:"title"`
	Summary    string  `json:"summary"`
	HasSummary bool    `json:"has_summary"`
	Score      float64 `json:"score"`
}

func (value Hit) Validate() error {
	var problems validation.Errors
	if strings.TrimSpace(value.ID) == "" {
		problems.Add(FieldID, problemRequired)
	}
	if strings.TrimSpace(value.Type) == "" {
		problems.Add(FieldType, problemRequired)
	}
	if strings.TrimSpace(value.Title) == "" {
		problems.Add(FieldTitle, problemRequired)
	}
	if value.HasSummary != (strings.TrimSpace(value.Summary) != "") {
		problems.Add(FieldHasSummary, problemSummaryMismatch)
	}
	if value.Score < minimumScore || value.Score > maximumScore {
		problems.Add(FieldScore, problemScoreRange)
	}
	return problems.Err()
}
