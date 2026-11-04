package rankeval

import (
	"maps"
	"slices"
	"testing"

	"github.com/theroymind/railyard/internal/contracts/card"
	"github.com/theroymind/railyard/internal/contracts/ticket"
)

func fixtureCard(id, relation string, distance int, commits ...string) card.Card {
	return card.Card{ID: id, Type: "task", Title: id, Summary: id, HasSummary: true, Distance: distance, Relation: relation, Commits: commits, Files: []string{}}
}

func TestBuildCaseOnFixtureHistory(t *testing.T) {
	history := buildFixtureHistory(t)
	commit := func(position int) string { return history.commits[position] }
	task := ticket.Ticket{ID: "task", Title: "The task", Acceptance: []ticket.Criterion{{Text: "a.txt changes"}, {Text: "c.txt appears"}}}
	start := fixtureCard("task", card.RelationStart, 0, commit(fixtureTaskSecond), commit(fixtureTaskFirst))
	tests := []struct {
		name               string
		cards              []card.Card
		expectedReason     string
		expectedCandidates []string
		expectedRelevant   []string
		expectedCommits    map[string][]string
	}{
		{
			"A card before the cut that shares a file is relevant, one that doesn't isn't, one after the cut is out",
			[]card.Card{
				start,
				fixtureCard("shares-a", card.RelationDependency, 1, commit(fixtureCardBeforeCut)),
				fixtureCard("other-file", card.RelationWikilink, 1, commit(fixtureOtherFile)),
				fixtureCard("after-cut", card.RelationBacklink, 1, commit(fixtureCardAfterCut)),
				fixtureCard("no-commits", card.RelationParent, 1),
			},
			"",
			[]string{"shares-a", "other-file"},
			[]string{"shares-a"},
			map[string][]string{"shares-a": {commit(fixtureCardBeforeCut)}, "other-file": {commit(fixtureOtherFile)}},
		},
		{
			"A card keeps only its commits before the cut",
			[]card.Card{
				start,
				fixtureCard("mixed", card.RelationDependency, 1, commit(fixtureCardAfterCut), commit(fixtureCardBeforeCut)),
			},
			"",
			[]string{"mixed"},
			[]string{"mixed"},
			map[string][]string{"mixed": {commit(fixtureCardBeforeCut)}},
		},
		{
			"The task's own commits on another card are after the cut",
			[]card.Card{start, fixtureCard("twin", card.RelationDependency, 1, commit(fixtureTaskFirst))},
			skipReasonNoCandidates, nil, nil, nil,
		},
		{
			"No candidate shares a file",
			[]card.Card{start, fixtureCard("other-file", card.RelationWikilink, 1, commit(fixtureOtherFile))},
			skipReasonNoRelevant, nil, nil, nil,
		},
		{
			"A task without commits",
			[]card.Card{fixtureCard("task", card.RelationStart, 0), fixtureCard("shares-a", card.RelationDependency, 1, commit(fixtureCardBeforeCut))},
			skipReasonNoCommits, nil, nil, nil,
		},
		{
			"A task whose first commit is the root",
			[]card.Card{fixtureCard("task", card.RelationStart, 0, commit(fixtureCardBeforeCut)), fixtureCard("later", card.RelationDependency, 1, commit(fixtureOtherFile))},
			skipReasonRootCommit, nil, nil, nil,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			built, reason, err := buildCase(task, tt.cards, history.repo, DefaultNotesDirectory)

			if err != nil {
				t.Fatal(err)
			}
			if reason != tt.expectedReason {
				t.Fatalf("buildCase() reason = %q, want %q", reason, tt.expectedReason)
			}
			if reason != "" {
				return
			}
			if built.Cut != commit(fixtureOtherFile) {
				t.Errorf("buildCase() cut = %s, want %s", built.Cut, commit(fixtureOtherFile))
			}
			if !slices.Equal(cardIDs(built.Candidates), tt.expectedCandidates) {
				t.Errorf("buildCase() candidates = %q, want %q", cardIDs(built.Candidates), tt.expectedCandidates)
			}
			if !slices.Equal(slices.Sorted(maps.Keys(built.Relevant)), tt.expectedRelevant) {
				t.Errorf("buildCase() relevant = %q, want %q", slices.Sorted(maps.Keys(built.Relevant)), tt.expectedRelevant)
			}
			for _, candidate := range built.Candidates {
				if !slices.Equal(candidate.Commits, tt.expectedCommits[candidate.ID]) {
					t.Errorf("buildCase() %s commits = %q, want %q", candidate.ID, candidate.Commits, tt.expectedCommits[candidate.ID])
				}
			}
			if !slices.Equal(built.Criteria, []string{"a.txt changes", "c.txt appears"}) {
				t.Errorf("buildCase() criteria = %q", built.Criteria)
			}
		})
	}
}

func TestCodeFiles(t *testing.T) {
	tests := []struct {
		name     string
		paths    []string
		notes    string
		expected []string
	}{
		{"Notes are left out", []string{"docs/tasks/a.md", "sim/a.odin", "docs/README.md"}, "docs", []string{"sim/a.odin"}},
		{"A file whose name starts like the notes directory stays", []string{"docs.txt", "documents/a.md"}, "docs", []string{"docs.txt", "documents/a.md"}},
		{"The notes directory is cleaned", []string{"docs/a.md", "b.txt"}, "./docs/", []string{"b.txt"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if kept := codeFiles(tt.paths, tt.notes); !slices.Equal(kept, tt.expected) {
				t.Errorf("codeFiles() = %q, want %q", kept, tt.expected)
			}
		})
	}
}
