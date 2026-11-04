package rankeval

import (
	"math"
	"testing"
	"time"

	"github.com/theroymind/go-tix/contracts/card"
)

const floatTolerance = 1e-9

func TestPrecisionAtTop(t *testing.T) {
	relevant := map[string]bool{"a": true, "c": true, "e": true}
	tests := []struct {
		name              string
		order             []string
		top               int
		expectedPack      int
		expectedHits      int
		expectedPrecision float64
	}{
		{"More cards than the pack holds", []string{"a", "b", "c", "d", "e", "f"}, 5, 5, 3, 0.6},
		{"Fewer cards than the pack holds", []string{"b", "a"}, 5, 2, 1, 0.5},
		{"The order decides what the pack holds", []string{"e", "c", "a", "b", "d"}, 2, 2, 2, 1},
		{"No cards", nil, 5, 0, 0, 0},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			pack, hits, precision := precisionAtTop(tt.order, relevant, tt.top)

			if pack != tt.expectedPack || hits != tt.expectedHits || math.Abs(precision-tt.expectedPrecision) > floatTolerance {
				t.Errorf("precisionAtTop() = (%d, %d, %v), want (%d, %d, %v)", pack, hits, precision, tt.expectedPack, tt.expectedHits, tt.expectedPrecision)
			}
		})
	}
}

type timedRanker struct {
	latency     time.Duration
	parseFailed bool
}

func (ranker timedRanker) rank(request rankRequest) (ranking, error) {
	return ranking{Order: cardIDs(request.Cards), Latency: ranker.latency, ParseFailed: ranker.parseFailed}, nil
}

func TestRunCases(t *testing.T) {
	cases := []evalCase{
		{Task: "one", Candidates: []card.Card{fixtureCard("a", card.RelationParent, 1), fixtureCard("b", card.RelationParent, 1)}, Relevant: map[string]bool{"a": true}},
		{Task: "two", Candidates: []card.Card{fixtureCard("c", card.RelationParent, 1), fixtureCard("d", card.RelationParent, 1), fixtureCard("e", card.RelationParent, 1)}, Relevant: map[string]bool{"c": true, "d": true, "e": true}},
	}
	tests := []struct {
		name                  string
		ranker                timedRanker
		top                   int
		expectedPrecision     float64
		expectedLatencyMs     float64
		expectedParseFailures int
	}{
		{"Mean precision over cases, latency over cards", timedRanker{latency: 500 * time.Millisecond}, 5, 0.75, 200, 0},
		{"A smaller pack cuts each case's order", timedRanker{latency: 0}, 1, 1, 0, 0},
		{"Parse failures are counted per case", timedRanker{parseFailed: true}, 5, 0.75, 0, 2},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ran, err := runCases(tt.ranker, cases, tt.top)

			if err != nil {
				t.Fatal(err)
			}
			if math.Abs(ran.PrecisionAtTop-tt.expectedPrecision) > floatTolerance {
				t.Errorf("runCases() precision = %v, want %v", ran.PrecisionAtTop, tt.expectedPrecision)
			}
			if math.Abs(ran.LatencyPerCardMs-tt.expectedLatencyMs) > floatTolerance {
				t.Errorf("runCases() latency per card = %v, want %v", ran.LatencyPerCardMs, tt.expectedLatencyMs)
			}
			if ran.ParseFailures != tt.expectedParseFailures {
				t.Errorf("runCases() parse failures = %d, want %d", ran.ParseFailures, tt.expectedParseFailures)
			}
			if len(ran.Cases) != len(cases) {
				t.Errorf("runCases() rows = %d, want %d", len(ran.Cases), len(cases))
			}
		})
	}
}

func TestPrecisionSpread(t *testing.T) {
	tests := []struct {
		name     string
		runs     []runReport
		expected float64
	}{
		{"No runs", nil, 0},
		{"One run", []runReport{{PrecisionAtTop: 0.5}}, 0},
		{"The gap between the best and the worst", []runReport{{PrecisionAtTop: 0.5}, {PrecisionAtTop: 0.7}, {PrecisionAtTop: 0.6}}, 0.2},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if spread := precisionSpread(tt.runs); math.Abs(spread-tt.expected) > floatTolerance {
				t.Errorf("precisionSpread() = %v, want %v", spread, tt.expected)
			}
		})
	}
}

func TestDecisiveCases(t *testing.T) {
	cases := []evalCase{
		{Candidates: []card.Card{fixtureCard("a", card.RelationParent, 1)}},
		{Candidates: []card.Card{fixtureCard("b", card.RelationParent, 1), fixtureCard("c", card.RelationParent, 1)}},
		{Candidates: []card.Card{fixtureCard("d", card.RelationParent, 1), fixtureCard("e", card.RelationParent, 1), fixtureCard("f", card.RelationParent, 1)}},
	}
	tests := []struct {
		name     string
		top      int
		expected int
	}{
		{"A pack of one leaves two cases decisive", 1, 2},
		{"A pack of two leaves one", 2, 1},
		{"A pack bigger than every case leaves none", 3, 0},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if count := decisiveCases(cases, tt.top); count != tt.expected {
				t.Errorf("decisiveCases() = %d, want %d", count, tt.expected)
			}
		})
	}
}
