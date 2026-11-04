package rankeval

import (
	"strings"

	"github.com/theroymind/go-tix/contracts/card"
)

// rankPrompt asks the model to score every card against the task's title, summary and criteria.
func rankPrompt(request rankRequest) string {
	sections := []string{
		promptInstruction,
		promptTitleLabel + request.Title,
	}
	if strings.TrimSpace(request.Summary) != "" {
		sections = append(sections, promptSummaryLabel+request.Summary)
	}
	if len(request.Criteria) > 0 {
		sections = append(sections, promptCriteriaHeading+listItems(request.Criteria))
	}
	sections = append(sections, promptCardsHeading+listItems(cardLines(request.Cards)), promptAnswerInstruction)
	return strings.Join(sections, promptSectionSeparator) + "\n"
}

func cardLines(cards []card.Card) []string {
	lines := make([]string, 0, len(cards))
	for _, item := range cards {
		summary := item.Summary
		if !item.HasSummary {
			summary = promptMissingSummary
		}
		fields := []string{
			promptCardIDLabel + item.ID,
			promptCardTypeLabel + item.Type,
			promptCardTitleLabel + item.Title,
			promptCardSummaryLabel + summary,
		}
		if len(item.Files) > 0 {
			fields = append(fields, promptCardFilesLabel+strings.Join(item.Files, promptFilesJoiner))
		}
		lines = append(lines, strings.Join(fields, promptCardFieldJoiner))
	}
	return lines
}

func listItems(items []string) string {
	lines := make([]string, len(items))
	for index, item := range items {
		lines[index] = promptListItemPrefix + item
	}
	return strings.Join(lines, "\n")
}
