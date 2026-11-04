package rankeval

import (
	"errors"
	"path/filepath"
	"strings"

	"github.com/theroymind/railyard/internal/contracts/card"
	"github.com/theroymind/railyard/internal/contracts/ticket"
)

// evalCase is one done task as the ranker would see it before its own commits: the task's text, the cut, the
// cards left with commits before the cut, and which of those cards a history reader calls relevant.
type evalCase struct {
	Task       string
	Title      string
	Summary    string
	Criteria   []string
	Cut        string
	Candidates []card.Card
	Relevant   map[string]bool
}

type skippedCase struct {
	Task   string `json:"task"`
	Reason string `json:"reason"`
}

type history interface {
	oldestCommit(hashes []string) (string, error)
	parentOf(hash string) (string, error)
	ancestorsOf(cut string) (map[string]bool, error)
	filesTouched(hash string) ([]string, error)
}

func buildCases(store vault, repo history, depth int, notesDirectory string) ([]evalCase, []skippedCase, error) {
	tasks, err := store.doneTasks()
	if err != nil {
		return nil, nil, err
	}
	var cases []evalCase
	skipped := []skippedCase{}
	for _, task := range tasks {
		cards, err := store.context(task.ID, depth)
		if err != nil {
			return nil, nil, err
		}
		shown, err := store.show(task.ID)
		if err != nil {
			return nil, nil, err
		}
		built, reason, err := buildCase(shown, cards, repo, notesDirectory)
		if err != nil {
			return nil, nil, err
		}
		if reason != "" {
			skipped = append(skipped, skippedCase{Task: task.ID, Reason: reason})
			continue
		}
		cases = append(cases, built)
	}
	return cases, skipped, nil
}

// buildCase turns one task's walk into a case, or names the reason it yields none: the task has no commits or a
// root commit, its commits touched only notes, no card has commits before the cut, or none of those is relevant.
func buildCase(task ticket.Ticket, cards []card.Card, repo history, notesDirectory string) (evalCase, string, error) {
	start, err := startCard(cards)
	if err != nil {
		return evalCase{}, "", err
	}
	if len(start.Commits) == 0 {
		return evalCase{}, skipReasonNoCommits, nil
	}
	cut, reason, err := cutBefore(start.Commits, repo)
	if err != nil || reason != "" {
		return evalCase{}, reason, err
	}
	taskFiles, err := codeFilesTouched(start.Commits, repo, notesDirectory)
	if err != nil {
		return evalCase{}, "", err
	}
	if len(taskFiles) == 0 {
		return evalCase{}, skipReasonNoCodeFiles, nil
	}
	ancestors, err := repo.ancestorsOf(cut)
	if err != nil {
		return evalCase{}, "", err
	}
	candidates := asOfCandidates(cards, ancestors)
	if len(candidates) == 0 {
		return evalCase{}, skipReasonNoCandidates, nil
	}
	relevant, err := relevantCards(candidates, taskFiles, repo, notesDirectory)
	if err != nil {
		return evalCase{}, "", err
	}
	if len(relevant) == 0 {
		return evalCase{}, skipReasonNoRelevant, nil
	}
	built := evalCase{
		Task:       start.ID,
		Title:      task.Title,
		Summary:    start.Summary,
		Criteria:   criteriaText(task.Acceptance),
		Cut:        cut,
		Candidates: candidates,
		Relevant:   relevant,
	}
	return built, "", nil
}

func startCard(cards []card.Card) (card.Card, error) {
	for _, item := range cards {
		if item.Relation == card.RelationStart {
			return item, nil
		}
	}
	return card.Card{}, ErrNoStartCard
}

// cutBefore is the parent of the oldest of the task's commits, or a skip reason when that commit is a root.
func cutBefore(commits []string, repo history) (string, string, error) {
	oldest, err := repo.oldestCommit(commits)
	if err != nil {
		return "", "", err
	}
	cut, err := repo.parentOf(oldest)
	if err == nil {
		return cut, "", nil
	}
	if errors.Is(err, ErrNoParent) {
		return "", skipReasonRootCommit, nil
	}
	return "", "", err
}

// codeFilesTouched is the set of files the commits changed, leaving out everything under the notes directory.
func codeFilesTouched(commits []string, repo history, notesDirectory string) (map[string]bool, error) {
	files := map[string]bool{}
	for _, commit := range commits {
		touched, err := repo.filesTouched(commit)
		if err != nil {
			return nil, err
		}
		for _, path := range codeFiles(touched, notesDirectory) {
			files[path] = true
		}
	}
	return files, nil
}

func codeFiles(paths []string, notesDirectory string) []string {
	notesPrefix := filepath.ToSlash(filepath.Clean(notesDirectory)) + "/"
	var kept []string
	for _, path := range paths {
		if !strings.HasPrefix(path, notesPrefix) {
			kept = append(kept, path)
		}
	}
	return kept
}

// asOfCandidates drops the start card, keeps on every other card only the commits the cut descends from, and
// drops the cards left with none, in walk order.
func asOfCandidates(cards []card.Card, ancestors map[string]bool) []card.Card {
	var candidates []card.Card
	for _, item := range cards {
		if item.Relation == card.RelationStart {
			continue
		}
		var before []string
		for _, commit := range item.Commits {
			if ancestors[commit] {
				before = append(before, commit)
			}
		}
		if len(before) == 0 {
			continue
		}
		kept := item
		kept.Commits = before
		candidates = append(candidates, kept)
	}
	return candidates
}

// relevantCards names each candidate whose commits touched a file the task touched, by id.
func relevantCards(candidates []card.Card, taskFiles map[string]bool, repo history, notesDirectory string) (map[string]bool, error) {
	relevant := map[string]bool{}
	for _, candidate := range candidates {
		cardFiles, err := codeFilesTouched(candidate.Commits, repo, notesDirectory)
		if err != nil {
			return nil, err
		}
		if sharesFile(cardFiles, taskFiles) {
			relevant[candidate.ID] = true
		}
	}
	return relevant, nil
}

func sharesFile(left, right map[string]bool) bool {
	for path := range left {
		if right[path] {
			return true
		}
	}
	return false
}

func criteriaText(criteria []ticket.Criterion) []string {
	texts := make([]string, 0, len(criteria))
	for _, criterion := range criteria {
		texts = append(texts, criterion.Text)
	}
	return texts
}
