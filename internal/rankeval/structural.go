package rankeval

import (
	"slices"

	"github.com/theroymind/go-tix/contracts/card"
)

// structuralOrder sorts cards the way odin-tbc's warm_up.py packs them: nearest first, then parent and
// dependency before wikilink before backlink, then cards with commits before those without. Ties keep walk order.
func structuralOrder(cards []card.Card) []card.Card {
	ordered := slices.Clone(cards)
	slices.SortStableFunc(ordered, func(left, right card.Card) int {
		if left.Distance != right.Distance {
			return left.Distance - right.Distance
		}
		if relationRank(left.Relation) != relationRank(right.Relation) {
			return relationRank(left.Relation) - relationRank(right.Relation)
		}
		return commitsRank(left) - commitsRank(right)
	})
	return ordered
}

func relationRank(relation string) int {
	switch relation {
	case card.RelationStart, card.RelationParent, card.RelationDependency:
		return relationRankStrong
	case card.RelationWikilink:
		return relationRankWikilink
	case card.RelationBacklink:
		return relationRankBacklink
	}
	return relationRankUnknown
}

func commitsRank(item card.Card) int {
	if len(item.Commits) > 0 {
		return 0
	}
	return 1
}

func cardIDs(cards []card.Card) []string {
	ids := make([]string, len(cards))
	for index, item := range cards {
		ids[index] = item.ID
	}
	return ids
}
