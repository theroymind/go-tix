package tix

import (
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"text/tabwriter"

	"github.com/theroymind/railyard/internal/cli"
	"github.com/theroymind/railyard/internal/contracts/card"
	"github.com/theroymind/railyard/internal/tix/backend"
	"github.com/theroymind/railyard/internal/tix/index"
)

var indexSpecification = cli.Specification{}

var searchSpecification = cli.Specification{
	Options: []cli.Option{
		{Name: optionType, Type: cli.OptionTypeString},
		{Name: optionTop, Type: cli.OptionTypeString},
		{Name: optionJSON, Type: cli.OptionTypeBoolean},
	},
	AllowsPositionals: true,
}

// repository is an open backend with the directory holding its .tix and the embeddings its config names.
type repository struct {
	store      backend.Backend
	root       string
	embeddings backend.Embeddings
}

// runIndex embeds every note's title and summary through the configured endpoint and writes the repository's index
// under the state directory, embedding only the notes that changed since the last run. When the endpoint fails
// the previous index stays as it was.
func runIndex(arguments []string, _ io.Reader, stdout io.Writer) error {
	if _, err := cli.Parse(arguments, indexSpecification); err != nil {
		return err
	}
	opened, err := openRepository()
	if err != nil {
		return err
	}
	notes, err := indexNotes(opened.store)
	if err != nil {
		return err
	}
	path, err := indexPath(opened.root)
	if err != nil {
		return err
	}
	previous, err := index.Read(path)
	if err != nil && !errors.Is(err, index.ErrNoIndex) {
		return err
	}
	embedder := index.OpenAIEmbedder{Endpoint: opened.embeddings.Endpoint, Model: opened.embeddings.Model}
	updated, embedded, err := index.Update(previous, notes, embedder, opened.embeddings.Model)
	if err != nil {
		return err
	}
	if err := index.Write(path, updated); err != nil {
		return err
	}
	_, err = fmt.Fprintf(stdout, indexedMessage, len(updated.Entries), embedded, path)
	return err
}

// runSearch embeds the query through the configured endpoint and prints the notes closest to it, best first.
func runSearch(arguments []string, _ io.Reader, stdout io.Writer) error {
	parsed, err := cli.Parse(arguments, searchSpecification)
	if err != nil {
		return err
	}
	if len(parsed.Positionals) != 1 {
		return cli.NewUsageError(ErrExpectedOneQuery)
	}
	top, err := parseTop(parsed)
	if err != nil {
		return err
	}
	opened, err := openRepository()
	if err != nil {
		return err
	}
	types, err := searchTypes(parsed, opened.store)
	if err != nil {
		return err
	}
	path, err := indexPath(opened.root)
	if err != nil {
		return err
	}
	loaded, err := index.Read(path)
	if err != nil {
		return err
	}
	if err := loaded.CheckModel(opened.embeddings.Model); err != nil {
		return err
	}
	embedder := index.OpenAIEmbedder{Endpoint: opened.embeddings.Endpoint, Model: opened.embeddings.Model}
	queries, err := embedder.Embed([]string{parsed.Positionals[0]})
	if err != nil {
		return err
	}
	ranked, err := index.Rank(loaded, queries[0], types, top)
	if err != nil {
		return err
	}
	hits, err := toHits(ranked)
	if err != nil {
		return err
	}
	if parsed.IsEnabled(optionJSON) {
		return cli.WriteJSON(stdout, hits)
	}
	return writeHits(stdout, hits)
}

// openRepository opens the backend the working directory is in, and fails when its config names no embeddings.
func openRepository() (repository, error) {
	workingDirectory, err := os.Getwd()
	if err != nil {
		return repository{}, err
	}
	tixDirectory, err := backend.FindDirectory(workingDirectory)
	if err != nil {
		return repository{}, err
	}
	config, err := backend.ReadConfig(tixDirectory)
	if err != nil {
		return repository{}, err
	}
	if config.Embeddings == nil {
		return repository{}, fmt.Errorf("%s: %w", backend.ConfigPath(tixDirectory), ErrNoEmbeddings)
	}
	store, err := backend.Open(workingDirectory)
	if err != nil {
		return repository{}, err
	}
	return repository{store: store, root: filepath.Dir(tixDirectory), embeddings: *config.Embeddings}, nil
}

func indexPath(root string) (string, error) {
	stateDirectory, err := cli.StateDirectory()
	if err != nil {
		return "", err
	}
	return index.Path(stateDirectory, root), nil
}

// indexNotes is every note tix context walks, as the index embeds it.
func indexNotes(store backend.Backend) ([]index.Note, error) {
	notes, err := contextNotes(store)
	if err != nil {
		return nil, err
	}
	indexed := make([]index.Note, len(notes))
	for position, note := range notes {
		indexed[position] = index.Note{ID: note.ID, Type: note.Type, Title: note.Title, Summary: note.Summary}
	}
	return indexed, nil
}

// parseTop reads --top as how many hits to print, DefaultSearchTop when it is absent.
func parseTop(parsed cli.Arguments) (int, error) {
	text, exists := parsed.String(optionTop)
	if !exists {
		return DefaultSearchTop, nil
	}
	top, err := strconv.Atoi(text)
	if err != nil || top < 1 {
		return 0, cli.NewUsageError(fmt.Errorf("--%s %s %w", optionTop, strconv.Quote(text), ErrInvalidTop))
	}
	return top, nil
}

// searchTypes returns the note types --type keeps: the value itself, and the project's own names for it when it is a
// ticket type, such as phase for epic. Without --type every type is kept.
func searchTypes(parsed cli.Arguments, store backend.Backend) ([]string, error) {
	value, exists := parsed.String(optionType)
	if !exists {
		return nil, nil
	}
	types := []string{value}
	for _, name := range typeNames(store) {
		if string(name.Type) == value && name.Name != value {
			types = append(types, name.Name)
		}
	}
	return types, nil
}

func toHits(ranked []index.Hit) ([]card.Hit, error) {
	hits := make([]card.Hit, 0, len(ranked))
	for _, hit := range ranked {
		value := card.Hit{
			ID: hit.Note.ID, Type: hit.Note.Type, Title: hit.Note.Title,
			Summary: hit.Note.Summary, HasSummary: hit.Note.Summary != "", Score: hit.Score,
		}
		if err := value.Validate(); err != nil {
			return nil, fmt.Errorf("%s: %w", hit.Note.ID, err)
		}
		hits = append(hits, value)
	}
	return hits, nil
}

// writeHits prints the hits as a SCORE ID TYPE SUMMARY table.
func writeHits(stdout io.Writer, hits []card.Hit) error {
	table := tabwriter.NewWriter(stdout, tableMinimumWidth, tableTabWidth, tablePadding, tablePaddingCharacter, 0)
	if _, err := fmt.Fprintln(table, hitTableHeader); err != nil {
		return err
	}
	for _, row := range hits {
		summary := row.Summary
		if !row.HasSummary {
			summary = missingSummary
		}
		if _, err := fmt.Fprintf(table, hitTableRow, row.Score, row.ID, row.Type, summary); err != nil {
			return err
		}
	}
	return table.Flush()
}
