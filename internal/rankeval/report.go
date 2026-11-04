package rankeval

import "time"

// report is what rankeval run prints: the dataset it built and, per model, one run report per run.
type report struct {
	Vault    string        `json:"vault"`
	Depth    int           `json:"depth"`
	Top      int           `json:"top"`
	Cases    int           `json:"cases"`
	Decisive int           `json:"decisive"`
	Skipped  []skippedCase `json:"skipped"`
	Models   []modelReport `json:"models"`
}

// modelReport holds a model's runs, the gap between its best and worst run's precision, and, when a run failed,
// the error that stopped it; its earlier runs stay.
type modelReport struct {
	Model           string      `json:"model"`
	PrecisionSpread float64     `json:"precision_spread"`
	Error           string      `json:"error,omitempty"`
	Runs            []runReport `json:"runs"`
}

// runReport is one pass over every case: the mean precision at the pack's card count, the model's latency per
// card scored, how many answers failed to parse, and one row per case.
type runReport struct {
	PrecisionAtTop   float64      `json:"precision_at_top"`
	LatencyPerCardMs float64      `json:"latency_per_card_ms"`
	ParseFailures    int          `json:"parse_failures"`
	Cases            []caseResult `json:"cases"`
}

type caseResult struct {
	Task        string   `json:"task"`
	Candidates  int      `json:"candidates"`
	Relevant    int      `json:"relevant"`
	Pack        int      `json:"pack"`
	Hits        int      `json:"hits"`
	Precision   float64  `json:"precision"`
	LatencyMs   float64  `json:"latency_ms"`
	ParseFailed bool     `json:"parse_failed"`
	Order       []string `json:"order"`
}

// precisionAtTop takes the first top ids of the order as the pack, fewer when there are fewer cards, and
// returns the pack size, the relevant cards in it and their share of the pack.
func precisionAtTop(order []string, relevant map[string]bool, top int) (int, int, float64) {
	pack := min(top, len(order))
	if pack == 0 {
		return 0, 0, 0
	}
	hits := 0
	for _, id := range order[:pack] {
		if relevant[id] {
			hits++
		}
	}
	return pack, hits, float64(hits) / float64(pack)
}

func runCases(ranker ranker, cases []evalCase, top int) (runReport, error) {
	result := runReport{Cases: []caseResult{}}
	precisionSum := 0.0
	latencySum := time.Duration(0)
	cardsScored := 0
	for _, item := range cases {
		request := rankRequest{Title: item.Title, Summary: item.Summary, Criteria: item.Criteria, Cards: item.Candidates}
		ranked, err := ranker.rank(request)
		if err != nil {
			return runReport{}, err
		}
		pack, hits, precision := precisionAtTop(ranked.Order, item.Relevant, top)
		result.Cases = append(result.Cases, caseResult{
			Task:        item.Task,
			Candidates:  len(item.Candidates),
			Relevant:    len(item.Relevant),
			Pack:        pack,
			Hits:        hits,
			Precision:   precision,
			LatencyMs:   milliseconds(ranked.Latency),
			ParseFailed: ranked.ParseFailed,
			Order:       ranked.Order,
		})
		precisionSum += precision
		latencySum += ranked.Latency
		cardsScored += len(item.Candidates)
		if ranked.ParseFailed {
			result.ParseFailures++
		}
	}
	if len(cases) > 0 {
		result.PrecisionAtTop = precisionSum / float64(len(cases))
	}
	if cardsScored > 0 {
		result.LatencyPerCardMs = milliseconds(latencySum) / float64(cardsScored)
	}
	return result, nil
}

// decisiveCases counts the cases with more candidates than the pack holds, the only ones where the order
// changes the precision.
func decisiveCases(cases []evalCase, top int) int {
	count := 0
	for _, item := range cases {
		if len(item.Candidates) > top {
			count++
		}
	}
	return count
}

func milliseconds(duration time.Duration) float64 {
	return duration.Seconds() * millisecondsPerSecond
}

// precisionSpread is the gap between the best and the worst run's mean precision.
func precisionSpread(runs []runReport) float64 {
	if len(runs) == 0 {
		return 0
	}
	lowest, highest := runs[0].PrecisionAtTop, runs[0].PrecisionAtTop
	for _, run := range runs[1:] {
		lowest = min(lowest, run.PrecisionAtTop)
		highest = max(highest, run.PrecisionAtTop)
	}
	return highest - lowest
}
