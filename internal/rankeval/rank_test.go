package rankeval

import (
	"errors"
	"slices"
	"strings"
	"testing"

	"github.com/theroymind/go-tix/contracts/card"
)

type fixedCompleter struct {
	answer string
	err    error
}

func (completer fixedCompleter) complete(string) (string, error) {
	return completer.answer, completer.err
}

var errModelDown = errors.New("model down")

func rankCards() []card.Card {
	return []card.Card{
		fixtureCard("far-parent", card.RelationParent, 2, "c1"),
		fixtureCard("near-backlink", card.RelationBacklink, 1, "c2"),
		fixtureCard("near-wikilink", card.RelationWikilink, 1),
		fixtureCard("near-dependency", card.RelationDependency, 1, "c3"),
		fixtureCard("near-parent", card.RelationParent, 1, "c4"),
	}
}

func TestStructuralOrder(t *testing.T) {
	expected := []string{"near-dependency", "near-parent", "near-wikilink", "near-backlink", "far-parent"}

	ordered := cardIDs(structuralOrder(rankCards()))

	if !slices.Equal(ordered, expected) {
		t.Errorf("structuralOrder() = %q, want %q", ordered, expected)
	}
}

func TestStructuralOrderPutsCommitsFirstWithinARelation(t *testing.T) {
	cards := []card.Card{
		fixtureCard("without", card.RelationWikilink, 1),
		fixtureCard("with", card.RelationWikilink, 1, "c1"),
		fixtureCard("also-without", card.RelationWikilink, 1),
	}
	expected := []string{"with", "without", "also-without"}

	ordered := cardIDs(structuralOrder(cards))

	if !slices.Equal(ordered, expected) {
		t.Errorf("structuralOrder() = %q, want %q", ordered, expected)
	}
}

func TestModelRankerRank(t *testing.T) {
	tests := []struct {
		name                string
		model               fixedCompleter
		expectedOrder       []string
		expectedParseFailed bool
		expectedError       error
	}{
		{
			"Scores order the cards, ties keep the structural order",
			fixedCompleter{answer: `{"scores":[{"id":"far-parent","score":3},{"id":"near-backlink","score":2},{"id":"near-wikilink","score":0},{"id":"near-dependency","score":2},{"id":"near-parent","score":0}]}`},
			[]string{"far-parent", "near-dependency", "near-backlink", "near-parent", "near-wikilink"},
			false, nil,
		},
		{
			"An unparsable answer scores every card 0 and keeps the structural order",
			fixedCompleter{answer: "no idea"},
			[]string{"near-dependency", "near-parent", "near-wikilink", "near-backlink", "far-parent"},
			true, nil,
		},
		{
			"A model that fails to answer is an error",
			fixedCompleter{err: errModelDown},
			nil, false, errModelDown,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ranker := modelRanker{model: tt.model}

			ranked, err := ranker.rank(rankRequest{Title: "A task", Cards: rankCards()})

			if !errors.Is(err, tt.expectedError) {
				t.Fatalf("rank() error = %v, want %v", err, tt.expectedError)
			}
			if !slices.Equal(ranked.Order, tt.expectedOrder) {
				t.Errorf("rank() order = %q, want %q", ranked.Order, tt.expectedOrder)
			}
			if ranked.ParseFailed != tt.expectedParseFailed {
				t.Errorf("rank() parse failed = %v, want %v", ranked.ParseFailed, tt.expectedParseFailed)
			}
		})
	}
}

func TestNewRanker(t *testing.T) {
	tests := []struct {
		name     string
		model    string
		expected string
	}{
		{"none is the structural order", ModelNone, "rankeval.structuralRanker"},
		{"haiku goes through claude", "haiku", "rankeval.claudeCompleter"},
		{"Anything else goes to the endpoint", "mlx-community/some-model", "rankeval.endpointCompleter"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ranker := newRanker(tt.model, DefaultEndpoint, false)

			if name := rankerName(ranker); name != tt.expected {
				t.Errorf("newRanker(%q) = %s, want %s", tt.model, name, tt.expected)
			}
		})
	}
}

func rankerName(ranker ranker) string {
	switch typed := ranker.(type) {
	case structuralRanker:
		return "rankeval.structuralRanker"
	case modelRanker:
		switch typed.model.(type) {
		case claudeCompleter:
			return "rankeval.claudeCompleter"
		case endpointCompleter:
			return "rankeval.endpointCompleter"
		}
	}
	return "unknown"
}

func TestRankPrompt(t *testing.T) {
	request := rankRequest{
		Title:    "Mana",
		Summary:  "Gives every combatant an MP pool.",
		Criteria: []string{"Attack costs nothing", "Spells cost MP"},
		Cards: []card.Card{
			{ID: "balance", Type: "task", Title: "Balance", Summary: "Holds every tuning value.", HasSummary: true, Distance: 1, Relation: card.RelationWikilink, Files: []string{"sim/balance.odin", "sim/world.odin"}},
			{ID: "bare", Type: "idea", Title: "Bare", Distance: 1, Relation: card.RelationBacklink},
		},
	}
	expected := []string{
		promptTitleLabel + "Mana",
		promptSummaryLabel + "Gives every combatant an MP pool.",
		promptCriteriaHeading + promptListItemPrefix + "Attack costs nothing\n" + promptListItemPrefix + "Spells cost MP",
		promptListItemPrefix + "id: balance | type: task | title: Balance | summary: Holds every tuning value. | files: sim/balance.odin, sim/world.odin",
		promptListItemPrefix + "id: bare | type: idea | title: Bare | summary: " + promptMissingSummary + "\n",
		promptAnswerInstruction,
	}

	prompt := rankPrompt(request)

	for _, part := range expected {
		if !strings.Contains(prompt, part) {
			t.Errorf("rankPrompt() lacks %q:\n%s", part, prompt)
		}
	}
	if strings.Contains(prompt, card.RelationWikilink) || strings.Contains(prompt, "distance") {
		t.Errorf("rankPrompt() leaks the walk's structure:\n%s", prompt)
	}
}
