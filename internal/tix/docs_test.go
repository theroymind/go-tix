package tix_test

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/theroymind/go-tix/internal/tix/backend"
	"github.com/theroymind/railyard-cli"
)

func TestDocsLs(t *testing.T) {
	tests := []struct {
		name             string
		arguments        []string
		expectedExitCode cli.ExitCode
		expectedOutput   string
		mentions         []string
	}{
		{"Every doc is listed by id, type, status and title",
			[]string{"docs", "ls"}, cli.ExitCodeSuccess,
			"ID                                 TYPE      STATUS     TITLE\n" +
				"enemy-ai                           idea      open       Enemy AI: targeting, behaviour and planning\n" +
				"enemy-choice-is-a-weighted-argmax  decision  accepted   enemy-choice-is-a-weighted-argmax\n" +
				"enemy-goals                        spec      confirmed  Enemy goals\n" +
				"turn-order-is-a-wait-counter       decision  proposed   Turn order is a wait counter\n", nil},
		{"Type and status filter the list, like grep -l 'status: proposed' docs/decisions/*.md",
			[]string{"docs", "ls", "--type", "decision", "--status", "proposed"}, cli.ExitCodeSuccess,
			"ID                            TYPE      STATUS    TITLE\n" +
				"turn-order-is-a-wait-counter  decision  proposed  Turn order is a wait counter\n", nil},
		{"A status no doc has lists the header alone",
			[]string{"docs", "ls", "--status", "superseded"}, cli.ExitCodeSuccess, "ID  TYPE  STATUS  TITLE\n", nil},
		{"A type the profile doesn't define names the types it does",
			[]string{"docs", "ls", "--type", "task"}, cli.ExitCodeFailure, "", []string{`"task"`, `"decision", "idea", "spec"`}},
		{"docs alone is a usage error",
			[]string{"docs"}, cli.ExitCodeUsage, "", []string{"expected docs ls", "usage:"}},
		{"A subcommand other than ls is a usage error",
			[]string{"docs", "show", "enemy-ai"}, cli.ExitCodeUsage, "", []string{"expected docs ls", `"show"`}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			root := copyVault(t)

			exitCode, output := run(t, root, tt.arguments...)

			if exitCode != tt.expectedExitCode {
				t.Fatalf("exit code = %d, expected %d; output %q", exitCode, tt.expectedExitCode, output)
			}
			if tt.expectedOutput != "" && output != tt.expectedOutput {
				t.Errorf("output =\n%s\nexpected\n%s", output, tt.expectedOutput)
			}
			for _, mention := range tt.mentions {
				if !strings.Contains(output, mention) {
					t.Errorf("output %q does not mention %q", output, mention)
				}
			}
		})
	}
}

func TestDocsLsJSON(t *testing.T) {
	root := copyVault(t)

	exitCode, output := run(t, root, "docs", "ls", "--json", "--type", "idea")

	if exitCode != cli.ExitCodeSuccess {
		t.Fatalf("exit code = %d, output %q", exitCode, output)
	}
	var docs []backend.Doc
	if err := json.Unmarshal([]byte(output), &docs); err != nil {
		t.Fatalf("output is not JSON: %v\n%s", err, output)
	}
	if len(docs) != 1 || docs[0].ID != "enemy-ai" || docs[0].Type != "idea" || docs[0].Path != filepath.Join("docs", "ideas", "enemy-ai.md") {
		t.Errorf("docs = %+v, expected the enemy-ai idea alone", docs)
	}
}

func TestDocShow(t *testing.T) {
	tests := []struct {
		name             string
		arguments        []string
		expectedExitCode cli.ExitCode
		isNote           bool
		mentions         []string
	}{
		{"The note is printed as it is", []string{"doc", "show", "enemy-ai"}, cli.ExitCodeSuccess, true, nil},
		{"A unique prefix resolves", []string{"doc", "show", "enemy-a"}, cli.ExitCodeSuccess, true, nil},
		{"JSON shows the doc", []string{"doc", "show", "--json", "enemy-ai"}, cli.ExitCodeSuccess, false,
			[]string{`"id": "enemy-ai"`, `"type": "idea"`, `"title": "Enemy AI: targeting, behaviour and planning"`, `"status": "open"`}},
		{"A ticket slug is no doc", []string{"doc", "show", "golem-guards"}, cli.ExitCodeFailure, false, []string{"no doc matches", `"golem-guards"`}},
		{"doc alone is a usage error", []string{"doc"}, cli.ExitCodeUsage, false, []string{"expected doc show or doc add"}},
		{"doc show without a reference is a usage error", []string{"doc", "show"}, cli.ExitCodeUsage, false, []string{"expected doc show REF"}},
		{"A subcommand other than show or add is a usage error", []string{"doc", "ls"}, cli.ExitCodeUsage, false, []string{"expected doc show or doc add", `"ls"`}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			root := copyVault(t)

			exitCode, output := run(t, root, tt.arguments...)

			if exitCode != tt.expectedExitCode {
				t.Fatalf("exit code = %d, expected %d; output %q", exitCode, tt.expectedExitCode, output)
			}
			if tt.isNote {
				if expected := readFile(t, filepath.Join(root, "docs", "ideas", "enemy-ai.md")); output != expected {
					t.Errorf("output =\n%s\nexpected the note\n%s", output, expected)
				}
			}
			for _, mention := range tt.mentions {
				if !strings.Contains(output, mention) {
					t.Errorf("output %q does not mention %q", output, mention)
				}
			}
		})
	}
}

func TestDocCommandsOnTheYAMLBackend(t *testing.T) {
	tests := []struct {
		name      string
		arguments []string
	}{
		{"docs ls", []string{"docs", "ls"}},
		{"doc show", []string{"doc", "show", yamlID}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			root, _ := yamlRepository(t)

			exitCode, output := run(t, root, tt.arguments...)

			if exitCode != cli.ExitCodeFailure {
				t.Fatalf("exit code = %d, expected %d; output %q", exitCode, cli.ExitCodeFailure, output)
			}
			if !strings.Contains(output, backend.ErrNoDocs.Error()) {
				t.Errorf("output %q does not say the backend has no docs", output)
			}
		})
	}
}

func TestDocAdd(t *testing.T) {
	today := time.Now().Format(backend.DateLayout)
	const created = "docs/decisions/bosses-plan-with-goap.md"
	tests := []struct {
		name             string
		arguments        []string
		expectedExitCode cli.ExitCode
		expectedOutput   string
		writes           bool
		mentions         []string
	}{
		{"A decision is created from its template with status proposed and today's date, and its id printed",
			[]string{"doc", "add", "--type", "decision", "Bosses plan with GOAP"}, cli.ExitCodeSuccess, "bosses-plan-with-goap\n", true, nil},
		{"An existing slug exits 1 and writes nothing",
			[]string{"doc", "add", "--type", "decision", "Turn order is a wait counter"}, cli.ExitCodeFailure, "", false,
			[]string{`"turn-order-is-a-wait-counter"`, "is taken by"}},
		{"A type with no template exits 1 and writes nothing",
			[]string{"doc", "add", "--type", "idea", "Bosses"}, cli.ExitCodeFailure, "", false, []string{`"idea"`, "has no template"}},
		{"A type outside the profile names the types in it",
			[]string{"doc", "add", "--type", "task", "Bosses"}, cli.ExitCodeFailure, "", false, []string{`"task"`, `"decision", "idea", "spec"`}},
		{"The type is required",
			[]string{"doc", "add", "Bosses"}, cli.ExitCodeUsage, "", false, []string{"--type"}},
		{"The title is required",
			[]string{"doc", "add", "--type", "decision"}, cli.ExitCodeUsage, "", false, []string{"expected doc add --type T TITLE"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			root := copyVault(t)
			before := snapshot(t, root)

			exitCode, output := run(t, root, tt.arguments...)

			if exitCode != tt.expectedExitCode {
				t.Fatalf("exit code = %d, expected %d; output %q", exitCode, tt.expectedExitCode, output)
			}
			if tt.expectedOutput != "" && output != tt.expectedOutput {
				t.Errorf("output = %q, expected %q", output, tt.expectedOutput)
			}
			for _, mention := range tt.mentions {
				if !strings.Contains(output, mention) {
					t.Errorf("output %q does not mention %q", output, mention)
				}
			}
			path := filepath.Join(root, created)
			if tt.writes {
				text, err := os.ReadFile(path)
				if err != nil {
					t.Fatalf("the decision was not written: %v", err)
				}
				for _, line := range []string{"type: decision\n", "status: proposed\n", "date: \"" + today + "\"\n"} {
					if !strings.Contains(string(text), line) {
						t.Errorf("note lacks %q:\n%s", line, text)
					}
				}
				before[path] = string(text)
			}
			if after := snapshot(t, root); !equalSnapshots(before, after) {
				t.Errorf("other notes changed:\n%s", snapshotDiff(before, after))
			}
		})
	}
}
