// Command searchcompare measures tix search against keyword matching on a vault's done tasks. For each done task
// with commits, the notes the task's commits touched are the other notes whose listed files overlap the files those
// commits changed; the task's title and summary are the query; and each ranker is scored on how often those notes
// land in its top hits. Run it as
//
//	go run ./tools/searchcompare --repository PATH --endpoint URL --model NAME [--top N]
//
// It prints a Markdown table to paste into docs/SPEC.md.
package main

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"slices"
	"strings"

	"github.com/theroymind/railyard/internal/cli"
	"github.com/theroymind/railyard/internal/contracts/ticket"
	"github.com/theroymind/railyard/internal/tix/backend"
	"github.com/theroymind/railyard/internal/tix/backend/vaultstore"
	"github.com/theroymind/railyard/internal/tix/backend/yamlstore"
	"github.com/theroymind/railyard/internal/tix/index"
)

const (
	optionRepository = "repository"
	optionEndpoint   = "endpoint"
	optionModel      = "model"
	optionTop        = "top"
	defaultTop       = 5
	ticketTrailer    = "Ticket: "
	gitCommand       = "git"
	lineSeparator    = "\n"
	textSeparator    = "\n"
)

var (
	ErrNoContext = errors.New("the backend has no context notes")
	ErrNoTasks   = errors.New("no done task has both commits and touched notes")
)

var specification = cli.Specification{
	Options: []cli.Option{
		{Name: optionRepository, Type: cli.OptionTypeString},
		{Name: optionEndpoint, Type: cli.OptionTypeString},
		{Name: optionModel, Type: cli.OptionTypeString},
		{Name: optionTop, Type: cli.OptionTypeString},
	},
}

// trial is one done task: its query and the notes its commits touched.
type trial struct {
	id      string
	query   string
	targets []string
}

// outcome counts, over the trials, how each ranker did.
type outcome struct {
	hits       int
	recall     float64
	reciprocal float64
}

func main() {
	if err := run(os.Args[1:]); err != nil {
		fmt.Fprintln(os.Stderr, "searchcompare:", err)
		os.Exit(int(cli.ExitCodeFailure))
	}
}

func run(arguments []string) error {
	parsed, err := cli.Parse(arguments, specification)
	if err != nil {
		return err
	}
	repository, err := parsed.RequiredString(optionRepository)
	if err != nil {
		return err
	}
	endpoint, err := parsed.RequiredString(optionEndpoint)
	if err != nil {
		return err
	}
	model, err := parsed.RequiredString(optionModel)
	if err != nil {
		return err
	}
	top := defaultTop
	if text, exists := parsed.String(optionTop); exists {
		if _, err := fmt.Sscan(text, &top); err != nil {
			return err
		}
	}
	backend.Register(backend.KindYAML, yamlstore.Open)
	backend.Register(backend.KindVault, vaultstore.Open)
	store, err := backend.Open(repository)
	if err != nil {
		return err
	}
	reader, isReader := store.(backend.ContextReader)
	if !isReader {
		return ErrNoContext
	}
	notes, err := reader.ContextNotes()
	if err != nil {
		return err
	}
	trials, err := buildTrials(repository, store, notes)
	if err != nil {
		return err
	}
	if len(trials) == 0 {
		return ErrNoTasks
	}
	embedder := index.OpenAIEmbedder{Endpoint: endpoint, Model: model}
	built, _, err := index.Update(index.Index{}, indexNotes(notes), embedder, model)
	if err != nil {
		return err
	}
	embeddingOutcome, keywordOutcome := outcome{}, outcome{}
	fmt.Printf("| task | targets | embedding rank | keyword rank |\n|---|---|---|---|\n")
	for _, candidate := range trials {
		embeddingRanks, err := embeddingRanks(built, embedder, candidate)
		if err != nil {
			return err
		}
		keywordRanks := keywordRanks(indexNotes(notes), candidate)
		embeddingOutcome.add(embeddingRanks, len(candidate.targets), top)
		keywordOutcome.add(keywordRanks, len(candidate.targets), top)
		fmt.Printf("| %s | %d | %s | %s |\n", candidate.id, len(candidate.targets), describe(embeddingRanks), describe(keywordRanks))
	}
	fmt.Printf("\n%d trials, top %d\n\n| ranker | hit@%d | recall@%d | MRR |\n|---|---|---|---|\n", len(trials), top, top, top)
	fmt.Printf("| embedding (%s) | %d/%d | %.2f | %.2f |\n", model, embeddingOutcome.hits, len(trials), embeddingOutcome.recall/float64(len(trials)), embeddingOutcome.reciprocal/float64(len(trials)))
	fmt.Printf("| keyword | %d/%d | %.2f | %.2f |\n", keywordOutcome.hits, len(trials), keywordOutcome.recall/float64(len(trials)), keywordOutcome.reciprocal/float64(len(trials)))
	return nil
}

// buildTrials pairs each done task with the notes its commits touched, and leaves out tasks with no commits or no
// touched notes.
func buildTrials(repository string, store backend.Backend, notes []backend.ContextNote) ([]trial, error) {
	tickets, err := store.List()
	if err != nil {
		return nil, err
	}
	byID := map[string]backend.ContextNote{}
	for _, note := range notes {
		byID[note.ID] = note
	}
	var trials []trial
	for _, listed := range tickets {
		if listed.Type != ticket.TypeTask || listed.Status != ticket.StatusCompleted {
			continue
		}
		note := byID[listed.ID]
		commits, err := taskCommits(repository, note)
		if err != nil {
			return nil, err
		}
		if len(commits) == 0 {
			continue
		}
		touched, err := touchedFiles(repository, commits)
		if err != nil {
			return nil, err
		}
		targets := touchedNotes(notes, note.ID, touched)
		if len(targets) == 0 {
			continue
		}
		trials = append(trials, trial{id: note.ID, query: queryText(note), targets: targets})
	}
	return trials, nil
}

// taskCommits is the note's commits property joined with every commit whose message carries its Ticket trailer.
func taskCommits(repository string, note backend.ContextNote) ([]string, error) {
	output, err := git(repository, "log", "--all", "--format=%H", "--grep", "^"+ticketTrailer+note.ID+"$")
	if err != nil {
		return nil, err
	}
	commits := slices.Clone(note.Commits)
	for _, hash := range strings.Split(strings.TrimSpace(output), lineSeparator) {
		if hash != "" && !slices.Contains(commits, hash) {
			commits = append(commits, hash)
		}
	}
	return commits, nil
}

func touchedFiles(repository string, commits []string) ([]string, error) {
	var files []string
	for _, hash := range commits {
		output, err := git(repository, "show", "--name-only", "--format=", hash)
		if err != nil {
			return nil, err
		}
		for _, file := range strings.Split(strings.TrimSpace(output), lineSeparator) {
			if file != "" && !slices.Contains(files, file) {
				files = append(files, file)
			}
		}
	}
	return files, nil
}

// touchedNotes are the notes other than self whose listed files include one the commits touched.
func touchedNotes(notes []backend.ContextNote, self string, touched []string) []string {
	var targets []string
	for _, note := range notes {
		if note.ID == self {
			continue
		}
		for _, file := range note.Files {
			if slices.Contains(touched, file) {
				targets = append(targets, note.ID)
				break
			}
		}
	}
	return targets
}

func queryText(note backend.ContextNote) string {
	if note.Summary == "" {
		return note.Title
	}
	return note.Title + textSeparator + note.Summary
}

func indexNotes(notes []backend.ContextNote) []index.Note {
	indexed := make([]index.Note, 0, len(notes))
	for _, note := range notes {
		indexed = append(indexed, index.Note{ID: note.ID, Type: note.Type, Title: note.Title, Summary: note.Summary})
	}
	return indexed
}

// embeddingRanks is the 1-based rank of each target among every other note, by cosine similarity to the query.
func embeddingRanks(built index.Index, embedder index.Embedder, candidate trial) ([]int, error) {
	vectors, err := embedder.Embed([]string{candidate.query})
	if err != nil {
		return nil, err
	}
	hits, err := index.Rank(built, vectors[0], nil, len(built.Entries))
	if err != nil {
		return nil, err
	}
	return ranksOf(hits, candidate), nil
}

func keywordRanks(notes []index.Note, candidate trial) []int {
	return ranksOf(index.RankKeywords(notes, candidate.query, nil, len(notes)), candidate)
}

// ranksOf is each target's 1-based rank among the hits with the task itself skipped, or 0 when it isn't ranked.
func ranksOf(hits []index.Hit, candidate trial) []int {
	ranks := make([]int, len(candidate.targets))
	rank := 0
	for _, hit := range hits {
		if hit.Note.ID == candidate.id {
			continue
		}
		rank++
		if position := slices.Index(candidate.targets, hit.Note.ID); position >= 0 {
			ranks[position] = rank
		}
	}
	return ranks
}

func (total *outcome) add(ranks []int, targets, top int) {
	found := 0
	best := 0
	for _, rank := range ranks {
		if rank == 0 || rank > top {
			continue
		}
		found++
		if best == 0 || rank < best {
			best = rank
		}
	}
	if found > 0 {
		total.hits++
		total.reciprocal += 1 / float64(best)
	}
	total.recall += float64(found) / float64(targets)
}

func describe(ranks []int) string {
	parts := make([]string, len(ranks))
	for position, rank := range ranks {
		parts[position] = fmt.Sprint(rank)
		if rank == 0 {
			parts[position] = "-"
		}
	}
	return strings.Join(parts, " ")
}

func git(repository string, arguments ...string) (string, error) {
	command := exec.Command(gitCommand, arguments...)
	command.Dir = repository
	output, err := command.Output()
	if err != nil {
		return "", fmt.Errorf("git %s: %w", strings.Join(arguments, " "), err)
	}
	return string(output), nil
}
