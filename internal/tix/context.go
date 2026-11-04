package tix

import (
	"fmt"
	"io"
	"strconv"
	"strings"
	"text/tabwriter"

	"github.com/theroymind/railyard/internal/cli"
	"github.com/theroymind/railyard/internal/contracts/card"
	"github.com/theroymind/railyard/internal/tix/backend"
	"github.com/theroymind/railyard/internal/tix/graph"
)

var contextSpecification = cli.Specification{
	Options: []cli.Option{
		{Name: optionDepth, Type: cli.OptionTypeString},
		{Name: optionJSON, Type: cli.OptionTypeBoolean},
	},
	AllowsPositionals: true,
}

// runContext walks out from the ticket or doc the reference names, over parents, dependencies, wikilinks and
// backlinks, and prints a card for each note within the depth, at its shortest distance.
func runContext(arguments []string, _ io.Reader, stdout io.Writer) error {
	parsed, err := cli.Parse(arguments, contextSpecification)
	if err != nil {
		return err
	}
	if len(parsed.Positionals) != 1 {
		return cli.NewUsageError(ErrExpectedOneReference)
	}
	depth, err := parseDepth(parsed)
	if err != nil {
		return err
	}
	store, err := openBackend()
	if err != nil {
		return err
	}
	start, _, err := resolveEither(store, parsed.Positionals[0])
	if err != nil {
		return err
	}
	notes, err := contextNotes(store)
	if err != nil {
		return err
	}
	cards, err := walkCards(notes, start, depth)
	if err != nil {
		return err
	}
	if parsed.IsEnabled(optionJSON) {
		return cli.WriteJSON(stdout, cards)
	}
	return writeCards(stdout, cards)
}

// parseDepth reads --depth as a whole number of hops, DefaultContextDepth when it is absent.
func parseDepth(parsed cli.Arguments) (int, error) {
	text, exists := parsed.String(optionDepth)
	if !exists {
		return DefaultContextDepth, nil
	}
	depth, err := strconv.Atoi(text)
	if err != nil || depth < 0 {
		return 0, cli.NewUsageError(fmt.Errorf("--%s %s %w", optionDepth, strconv.Quote(text), ErrInvalidDepth))
	}
	return depth, nil
}

// contextNotes returns the backend's notes for the walk. A backend that isn't a ContextReader gives its tickets,
// linked through their parents and dependencies only.
func contextNotes(store backend.Backend) ([]backend.ContextNote, error) {
	if reader, isContextReader := store.(backend.ContextReader); isContextReader {
		return reader.ContextNotes()
	}
	tickets, err := store.List()
	if err != nil {
		return nil, err
	}
	notes := make([]backend.ContextNote, 0, len(tickets))
	for _, listed := range tickets {
		notes = append(notes, backend.ContextNote{
			ID: listed.ID, Type: string(listed.Type), Title: listed.Title,
			Parent: listed.Parent, Dependencies: listed.Dependencies,
		})
	}
	return notes, nil
}

// walkCards walks from start over each note's parent, dependencies, wikilinks and backlinks, in that order, to the
// notes the backend holds, and returns their cards in walk order. A link to a note the backend doesn't hold is left
// out, and a note that stops the walk is reached but not walked through, unless it is the start.
func walkCards(notes []backend.ContextNote, start string, depth int) ([]card.Card, error) {
	byID := make(map[string]backend.ContextNote, len(notes))
	for _, note := range notes {
		byID[note.ID] = note
	}
	neighbours := func(id string) []graph.Neighbour {
		note := byID[id]
		if note.StopsWalk && id != start {
			return nil
		}
		var held []graph.Neighbour
		for _, neighbour := range noteNeighbours(note) {
			if _, isHeld := byID[neighbour.ID]; isHeld {
				held = append(held, neighbour)
			}
		}
		return held
	}
	cards := []card.Card{}
	for _, reached := range graph.Walk(start, depth, neighbours) {
		value := toCard(byID[reached.ID], reached.Distance, reached.Relation)
		if err := value.Validate(); err != nil {
			return nil, fmt.Errorf("%s: %w", reached.ID, err)
		}
		cards = append(cards, value)
	}
	return cards, nil
}

// noteNeighbours lists a note's hops strongest first: its parent, its dependencies, its wikilinks, its backlinks.
func noteNeighbours(note backend.ContextNote) []graph.Neighbour {
	var neighbours []graph.Neighbour
	if note.Parent != "" {
		neighbours = append(neighbours, graph.Neighbour{ID: note.Parent, Relation: card.RelationParent})
	}
	for _, dependency := range note.Dependencies {
		neighbours = append(neighbours, graph.Neighbour{ID: dependency, Relation: card.RelationDependency})
	}
	for _, wikilink := range note.Wikilinks {
		neighbours = append(neighbours, graph.Neighbour{ID: wikilink, Relation: card.RelationWikilink})
	}
	for _, backlink := range note.Backlinks {
		neighbours = append(neighbours, graph.Neighbour{ID: backlink, Relation: card.RelationBacklink})
	}
	return neighbours
}

func toCard(note backend.ContextNote, distance int, relation string) card.Card {
	return card.Card{
		ID:         note.ID,
		Type:       note.Type,
		Title:      note.Title,
		Summary:    note.Summary,
		HasSummary: note.Summary != "",
		Distance:   distance,
		Relation:   relation,
		Commits:    nonNil(note.Commits),
		Files:      nonNil(note.Files),
	}
}

// nonNil returns values, or an empty slice for none, so that JSON prints [] rather than null.
func nonNil(values []string) []string {
	if values == nil {
		return []string{}
	}
	return values
}

// writeCards prints the cards as a DISTANCE RELATION ID TYPE COMMITS SUMMARY table, each commit by its short hash.
func writeCards(stdout io.Writer, cards []card.Card) error {
	table := tabwriter.NewWriter(stdout, tableMinimumWidth, tableTabWidth, tablePadding, tablePaddingCharacter, 0)
	if _, err := fmt.Fprintln(table, cardTableHeader); err != nil {
		return err
	}
	for _, row := range cards {
		summary := row.Summary
		if !row.HasSummary {
			summary = missingSummary
		}
		if _, err := fmt.Fprintf(table, cardTableRow, row.Distance, row.Relation, row.ID, row.Type, commitsCell(row.Commits), summary); err != nil {
			return err
		}
	}
	return table.Flush()
}

// commitsCell is the commits by their short hashes, or a dash when there are none.
func commitsCell(commits []string) string {
	if len(commits) == 0 {
		return tableAbsent
	}
	short := make([]string, len(commits))
	for index, commit := range commits {
		short[index] = commit[:min(len(commit), shortHashLength)]
	}
	return strings.Join(short, commitSeparator)
}
