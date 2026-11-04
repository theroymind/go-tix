package rankeval

import (
	"fmt"
	"strings"

	"github.com/theroymind/railyard-cli"
)

// repository reads commits from one git repository: which of a task's commits came first, the commit before
// it, every commit that commit descends from, and the files a commit touched.
type repository struct {
	root string
}

func (repo repository) run(arguments ...string) ([]string, error) {
	output, err := cli.RunChildIn(repo.root, gitExecutable, arguments, nil)
	if err != nil {
		return nil, err
	}
	return nonEmptyLines(string(output)), nil
}

// oldestCommit returns the one of hashes with the earliest commit time.
func (repo repository) oldestCommit(hashes []string) (string, error) {
	if len(hashes) == 0 {
		return "", ErrNoCommits
	}
	arguments := append([]string{"rev-list", "--no-walk=sorted", "--reverse"}, hashes...)
	lines, err := repo.run(arguments...)
	if err != nil {
		return "", err
	}
	if len(lines) == 0 {
		return "", ErrNoCommits
	}
	return lines[0], nil
}

// parentOf returns the first parent of hash, or ErrNoParent for a root commit.
func (repo repository) parentOf(hash string) (string, error) {
	lines, err := repo.run("rev-list", "--parents", "--no-walk", hash)
	if err != nil {
		return "", err
	}
	if len(lines) == 0 {
		return "", fmt.Errorf("%s: %w", hash, ErrNoParent)
	}
	fields := strings.Fields(lines[0])
	if len(fields) < 2 {
		return "", fmt.Errorf("%s: %w", hash, ErrNoParent)
	}
	return fields[1], nil
}

// ancestorsOf returns every commit the cut descends from, the cut included, keyed by full hash.
func (repo repository) ancestorsOf(cut string) (map[string]bool, error) {
	lines, err := repo.run("rev-list", cut)
	if err != nil {
		return nil, err
	}
	ancestors := make(map[string]bool, len(lines))
	for _, hash := range lines {
		ancestors[hash] = true
	}
	return ancestors, nil
}

// filesTouched lists the paths a commit changed against its first parent, or everything for a root commit.
func (repo repository) filesTouched(hash string) ([]string, error) {
	return repo.run("diff-tree", "--no-commit-id", "--name-only", "-r", "--root", hash)
}

func nonEmptyLines(text string) []string {
	var lines []string
	for line := range strings.SplitSeq(text, "\n") {
		trimmed := strings.TrimSpace(line)
		if trimmed != "" {
			lines = append(lines, trimmed)
		}
	}
	return lines
}
