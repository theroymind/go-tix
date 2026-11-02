package vault

import (
	"path/filepath"
	"slices"
	"testing"
)

func TestWikilinks(t *testing.T) {
	tests := []struct {
		name string
		text string
		want []string
	}{
		{name: "plain", text: "see [[mana-pool]].", want: []string{"mana-pool"}},
		{name: "alias", text: "see [[mana-pool|the pool]]", want: []string{"mana-pool"}},
		{name: "heading", text: "see [[enemy-goals#Shape]]", want: []string{"enemy-goals"}},
		{name: "heading and alias", text: "[[enemy-goals#Shape|shape]]", want: []string{"enemy-goals"}},
		{name: "escaped pipe in a table", text: `| [[mana-pool\|pool]] | x |`, want: []string{"mana-pool"}},
		{name: "path", text: "[[tasks/goal-scoring.md]]", want: []string{"goal-scoring"}},
		{name: "embed", text: "![[diagram]]", want: []string{"diagram"}},
		{name: "same-note heading", text: "[[#Shape]]", want: nil},
		{name: "deduplicated in order", text: "[[b]] [[a]] [[b|again]]", want: []string{"b", "a"}},
		{name: "code span", text: "`[[skip]]` and [[keep]]", want: []string{"keep"}},
		{name: "double backtick span", text: "``a ` [[skip]]`` [[keep]]", want: []string{"keep"}},
		{name: "unmatched backtick", text: "a ` [[keep]]", want: []string{"keep"}},
		{name: "backtick fence", text: "```go\n[[skip]]\n```\n[[keep]]", want: []string{"keep"}},
		{name: "tilde fence", text: "~~~\n[[skip]]\n```\n[[still-skip]]\n~~~\n[[keep]]", want: []string{"keep"}},
		{name: "longer closing fence", text: "```\n[[skip]]\n`````\n[[keep]]", want: []string{"keep"}},
		{name: "unclosed fence", text: "[[keep]]\n```\n[[skip]]", want: []string{"keep"}},
		{name: "none", text: "no links [here]", want: nil},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if got := Wikilinks(test.text); !slices.Equal(got, test.want) {
				t.Errorf("Wikilinks = %q, want %q", got, test.want)
			}
		})
	}
}

func TestWikilinksInFixtures(t *testing.T) {
	tests := []struct {
		fixture string
		want    []string
	}{
		{fixture: "enemy-puzzle.md", want: []string{"enemy-goals", "pillars", "sim-queries", "goal-scoring", "weighted-step-choice", "golem-guards", "harpy-hastes"}},
		{fixture: "goal-scoring.md", want: []string{"enemy-goals"}},
		{fixture: "enemy-choice-is-a-weighted-argmax.md", want: []string{"mana-pool"}},
		{fixture: "enemy-goals.md", want: []string{"mana-pool", "enemy-ai", "enemy-choice-is-a-weighted-argmax"}},
	}
	for _, test := range tests {
		t.Run(test.fixture, func(t *testing.T) {
			if got := Wikilinks(string(readFixture(t, test.fixture))); !slices.Equal(got, test.want) {
				t.Errorf("Wikilinks = %q, want %q", got, test.want)
			}
		})
	}
}

func TestBacklinks(t *testing.T) {
	tests := []struct {
		slug string
		want []string
	}{
		{slug: "enemy-goals", want: []string{"enemy-puzzle.md", "goal-scoring.md"}},
		{slug: "mana-pool", want: []string{"enemy-choice-is-a-weighted-argmax.md", "enemy-goals.md"}},
		{slug: "nobody-links-here", want: nil},
	}
	for _, test := range tests {
		t.Run(test.slug, func(t *testing.T) {
			got, err := Backlinks("testdata", test.slug)
			if err != nil {
				t.Fatal(err)
			}
			var want []string
			for _, name := range test.want {
				want = append(want, filepath.Join("testdata", name))
			}
			if !slices.Equal(got, want) {
				t.Errorf("Backlinks = %q, want %q", got, want)
			}
		})
	}
}

func TestBacklinksMissingRoot(t *testing.T) {
	tests := []struct {
		root string
	}{
		{root: filepath.Join("testdata", "missing")},
	}
	for _, test := range tests {
		if _, err := Backlinks(test.root, "x"); err == nil {
			t.Errorf("Backlinks(%s) error = nil, want one", test.root)
		}
	}
}
