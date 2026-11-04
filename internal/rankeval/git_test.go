package rankeval

import (
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strings"
	"testing"
)

// fixtureHistory is a repository built by the test: five commits on one line, each touching one file.
type fixtureHistory struct {
	repo    repository
	commits []string
}

const (
	fixtureCardBeforeCut = 0
	fixtureOtherFile     = 1
	fixtureTaskFirst     = 2
	fixtureTaskSecond    = 3
	fixtureCardAfterCut  = 4
)

var fixtureFiles = []string{"a.txt", "b.txt", "a.txt", "c.txt", "a.txt"}

func buildFixtureHistory(t *testing.T) fixtureHistory {
	t.Helper()
	root := t.TempDir()
	runGit(t, root, "init", "--quiet", "--initial-branch=main")
	var commits []string
	for index, name := range fixtureFiles {
		path := filepath.Join(root, name)
		if err := os.WriteFile(path, []byte(strings.Repeat("x", index+1)), 0o600); err != nil {
			t.Fatal(err)
		}
		runGit(t, root, "add", name)
		runGit(t, root, "commit", "--quiet", "--message", "commit "+name)
		commits = append(commits, strings.TrimSpace(runGit(t, root, "rev-parse", "HEAD")))
	}
	return fixtureHistory{repo: repository{root: root}, commits: commits}
}

func runGit(t *testing.T, root string, arguments ...string) string {
	t.Helper()
	command := exec.Command("git", arguments...)
	command.Dir = root
	command.Env = append(os.Environ(),
		"GIT_CONFIG_GLOBAL=/dev/null",
		"GIT_CONFIG_NOSYSTEM=1",
		"GIT_AUTHOR_NAME=test", "GIT_AUTHOR_EMAIL=test@example.com",
		"GIT_COMMITTER_NAME=test", "GIT_COMMITTER_EMAIL=test@example.com",
	)
	output, err := command.CombinedOutput()
	if err != nil {
		t.Fatalf("git %s: %v\n%s", strings.Join(arguments, " "), err, output)
	}
	return string(output)
}

func TestRepositoryOldestCommit(t *testing.T) {
	history := buildFixtureHistory(t)
	tests := []struct {
		name          string
		hashes        []int
		expected      int
		expectedError error
	}{
		{"The earlier of two commits given newest first", []int{fixtureTaskSecond, fixtureTaskFirst}, fixtureTaskFirst, nil},
		{"One commit is its own oldest", []int{fixtureTaskSecond}, fixtureTaskSecond, nil},
		{"No commits", nil, -1, ErrNoCommits},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			hashes := make([]string, len(tt.hashes))
			for index, position := range tt.hashes {
				hashes[index] = history.commits[position]
			}

			oldest, err := history.repo.oldestCommit(hashes)

			if !errors.Is(err, tt.expectedError) {
				t.Fatalf("oldestCommit() error = %v, want %v", err, tt.expectedError)
			}
			if tt.expected >= 0 && oldest != history.commits[tt.expected] {
				t.Errorf("oldestCommit() = %s, want %s", oldest, history.commits[tt.expected])
			}
		})
	}
}

func TestRepositoryParentOf(t *testing.T) {
	history := buildFixtureHistory(t)
	tests := []struct {
		name          string
		hash          int
		expected      int
		expectedError error
	}{
		{"The commit before the task's first", fixtureTaskFirst, fixtureOtherFile, nil},
		{"The root commit has no parent", fixtureCardBeforeCut, -1, ErrNoParent},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			parent, err := history.repo.parentOf(history.commits[tt.hash])

			if !errors.Is(err, tt.expectedError) {
				t.Fatalf("parentOf() error = %v, want %v", err, tt.expectedError)
			}
			if tt.expected >= 0 && parent != history.commits[tt.expected] {
				t.Errorf("parentOf() = %s, want %s", parent, history.commits[tt.expected])
			}
		})
	}
}

func TestRepositoryAncestorsOf(t *testing.T) {
	history := buildFixtureHistory(t)

	ancestors, err := history.repo.ancestorsOf(history.commits[fixtureOtherFile])
	if err != nil {
		t.Fatal(err)
	}

	tests := []struct {
		name     string
		hash     int
		expected bool
	}{
		{"The cut itself", fixtureOtherFile, true},
		{"A commit before the cut", fixtureCardBeforeCut, true},
		{"The task's first commit", fixtureTaskFirst, false},
		{"A commit after the task", fixtureCardAfterCut, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if ancestors[history.commits[tt.hash]] != tt.expected {
				t.Errorf("ancestorsOf()[%s] = %v, want %v", history.commits[tt.hash], !tt.expected, tt.expected)
			}
		})
	}
}

func TestRepositoryFilesTouched(t *testing.T) {
	history := buildFixtureHistory(t)
	tests := []struct {
		name     string
		hash     int
		expected []string
	}{
		{"A root commit lists its files", fixtureCardBeforeCut, []string{"a.txt"}},
		{"A later commit lists only what it changed", fixtureTaskSecond, []string{"c.txt"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			files, err := history.repo.filesTouched(history.commits[tt.hash])

			if err != nil {
				t.Fatal(err)
			}
			if !slices.Equal(files, tt.expected) {
				t.Errorf("filesTouched() = %q, want %q", files, tt.expected)
			}
		})
	}
}
