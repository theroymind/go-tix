package rankeval

import (
	"slices"
	"strings"
	"time"

	"github.com/theroymind/go-tix/contracts/card"
)

// rankRequest is what a ranker sees of a case: the task's text and the candidate cards, never the cut or the
// relevant set.
type rankRequest struct {
	Title    string
	Summary  string
	Criteria []string
	Cards    []card.Card
}

// ranking is a ranker's order of the request's card ids, best first, how long the model took, and whether its
// answer failed to parse, in which case every card scored 0 and the order fell back to the structural one.
type ranking struct {
	Order       []string
	Latency     time.Duration
	ParseFailed bool
}

type ranker interface {
	rank(request rankRequest) (ranking, error)
}

// completer answers one prompt with the model's text.
type completer interface {
	complete(prompt string) (string, error)
}

type structuralRanker struct{}

func (structuralRanker) rank(request rankRequest) (ranking, error) {
	return ranking{Order: cardIDs(structuralOrder(request.Cards))}, nil
}

// modelRanker asks a model to score each card, then orders the cards by score, highest first, breaking ties by
// the structural order.
type modelRanker struct {
	model completer
}

func (ranker modelRanker) rank(request rankRequest) (ranking, error) {
	prompt := rankPrompt(request)
	started := time.Now()
	answer, err := ranker.model.complete(prompt)
	latency := time.Since(started)
	if err != nil {
		return ranking{}, err
	}
	scores, parseErr := parseScores(answer, cardIDs(request.Cards))
	ordered := orderByScore(structuralOrder(request.Cards), scores)
	return ranking{Order: cardIDs(ordered), Latency: latency, ParseFailed: parseErr != nil}, nil
}

func orderByScore(structural []card.Card, scores map[string]int) []card.Card {
	ordered := slices.Clone(structural)
	slices.SortStableFunc(ordered, func(left, right card.Card) int {
		return scores[right.ID] - scores[left.ID]
	})
	return ordered
}

// newRanker picks the ranker a model name asks for: none for the structural order, a claude alias for claude
// -p, anything else for the endpoint.
func newRanker(name, endpoint string, thinking bool) ranker {
	if name == ModelNone {
		return structuralRanker{}
	}
	if slices.Contains(claudeModels, name) {
		return modelRanker{model: claudeCompleter{model: name}}
	}
	return modelRanker{model: newEndpointCompleter(strings.TrimRight(endpoint, "/"), name, thinking)}
}
