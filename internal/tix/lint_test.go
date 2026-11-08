package tix_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/theroymind/railyard-cli"
)

const gatesConfig = `prefix: odin
backend: vault
vault:
  tickets:
    task:
      ticket_type: task
      statuses: {todo: open, done: completed}
      fields: {phase: parent}
  docs:
    decision:
      statuses: [proposed, accepted, superseded]
  required_sections:
    task: [Acceptance criteria]
    decision: [Rejected]
`

const (
	cleanDecision = "---\ntype: decision\nstatus: accepted\nsummary: Turn order is a wait counter.\n---\n\n# Wait counter\n\nSee [[mana]] and [[tasks.base]].\n\n## Rejected\n\n- initiative\n"
	cleanTask     = "---\ntype: task\nstatus: todo\nsummary: Add mana.\n---\n\n# Mana\n\nFollows [[wait-counter]].\n\n## Acceptance criteria\n\n- [ ] mana\n"
)

// writeVault writes a repository holding .tix/config.yml and each file under docs/, and returns its root.
func writeVault(t *testing.T, config string, files map[string]string) string {
	t.Helper()
	root := t.TempDir()
	written := map[string]string{".tix/config.yml": config, "docs/tasks.base": ""}
	for path, text := range files {
		written[filepath.Join("docs", path)] = text
	}
	for path, text := range written {
		full := filepath.Join(root, path)
		if err := os.MkdirAll(filepath.Dir(full), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(full, []byte(text), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	return root
}

func TestLint(t *testing.T) {
	tests := []struct {
		name             string
		files            map[string]string
		arguments        []string
		expectedExitCode cli.ExitCode
		expectedLines    []string
	}{
		{"A clean vault passes with no output",
			map[string]string{"decisions/wait-counter.md": cleanDecision, "tasks/mana.md": cleanTask},
			[]string{"lint", "--all"}, cli.ExitCodeSuccess, nil},
		{"A note without a required section fails naming the note and rule",
			map[string]string{"decisions/wait-counter.md": strings.Replace(cleanDecision, "## Rejected", "## Alternatives", 1), "tasks/mana.md": cleanTask},
			[]string{"lint", "--all"}, cli.ExitCodeFailure,
			[]string{`docs/decisions/wait-counter.md: section: missing the section "Rejected"`, "tix lint: lint failed: 1 problems in 1 notes"}},
		{"A note without a summary fails naming the note and rule",
			map[string]string{"decisions/wait-counter.md": cleanDecision, "tasks/mana.md": strings.Replace(cleanTask, "summary: Add mana.\n", "summary:\n", 1)},
			[]string{"lint", "--all"}, cli.ExitCodeFailure, []string{"docs/tasks/mana.md: summary: has no summary"}},
		{"A link to no note fails naming the note and rule",
			map[string]string{"decisions/wait-counter.md": cleanDecision, "tasks/mana.md": strings.Replace(cleanTask, "[[wait-counter]]", "[[turn-order]]", 1)},
			[]string{"lint", "--all"}, cli.ExitCodeFailure, []string{"docs/tasks/mana.md: link: [[turn-order]] names no note or file in the vault"}},
		{"A parent property naming no note is a broken link",
			map[string]string{"decisions/wait-counter.md": cleanDecision, "tasks/mana.md": strings.Replace(cleanTask, "status: todo\n", "status: todo\nphase: first-fight\n", 1)},
			[]string{"lint", "--all"}, cli.ExitCodeFailure, []string{"docs/tasks/mana.md: link: [[first-fight]]"}},
		{"Every failing note is named",
			map[string]string{"decisions/wait-counter.md": strings.Replace(cleanDecision, "summary: Turn order is a wait counter.\n", "", 1), "tasks/mana.md": strings.Replace(cleanTask, "## Acceptance criteria", "## Criteria", 1)},
			[]string{"lint", "--all"}, cli.ExitCodeFailure,
			[]string{"docs/decisions/wait-counter.md: summary", "docs/tasks/mana.md: section", "2 problems in 2 notes"}},
		{"A reference lints only its note",
			map[string]string{"decisions/wait-counter.md": cleanDecision, "tasks/mana.md": strings.Replace(cleanTask, "summary: Add mana.\n", "", 1)},
			[]string{"lint", "wait-counter"}, cli.ExitCodeSuccess, nil},
		{"A note of a type outside the profile isn't linted",
			map[string]string{"decisions/wait-counter.md": cleanDecision, "tasks/mana.md": cleanTask, "notes/scratch.md": "---\ntype: scratch\n---\n\n[[nowhere]]\n"},
			[]string{"lint", "--all"}, cli.ExitCodeSuccess, nil},
		{"No reference and no --all is a usage error",
			map[string]string{"decisions/wait-counter.md": cleanDecision, "tasks/mana.md": cleanTask},
			[]string{"lint"}, cli.ExitCodeUsage, []string{"expected note references or --all"}},
		{"A reference and --all together are a usage error",
			map[string]string{"decisions/wait-counter.md": cleanDecision, "tasks/mana.md": cleanTask},
			[]string{"lint", "--all", "mana"}, cli.ExitCodeUsage, []string{"expected note references or --all"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			root := writeVault(t, gatesConfig, tt.files)

			exitCode, output := run(t, root, tt.arguments...)

			if exitCode != tt.expectedExitCode {
				t.Fatalf("exit code = %d, expected %d; output %q", exitCode, tt.expectedExitCode, output)
			}
			if tt.expectedLines == nil && output != "" {
				t.Errorf("output = %q, expected none", output)
			}
			for _, line := range tt.expectedLines {
				if !strings.Contains(output, line) {
					t.Errorf("output = %q, expected it to contain %q", output, line)
				}
			}
		})
	}
}

func TestLintNamesRequiredSectionsStillHoldingTheTemplate(t *testing.T) {
	tests := []struct {
		name          string
		arguments     []string
		expectedLines []string
	}{
		{"A task added bare names each required section the template filled",
			nil, []string{
				`docs/features/uncategorized/tasks/draw-the-map.md: template: "Description" still holds the template's text`,
				`docs/features/uncategorized/tasks/draw-the-map.md: template: "Files" still holds the template's text`,
				`docs/features/uncategorized/tasks/draw-the-map.md: template: "Acceptance criteria" still holds the template's text`,
			}},
		{"A task added whole is clean",
			[]string{"--description", "Draw it.", "--file", "map.go", "--acceptance", "The map shows.", "--set", "summary=Draw the map."}, nil},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			root := t.TempDir()
			if exitCode, output := run(t, root, "init", "--prefix", "tx"); exitCode != cli.ExitCodeSuccess {
				t.Fatalf("init exit code = %d, output %q", exitCode, output)
			}
			arguments := append(append([]string{"add", "--type", "task"}, tt.arguments...), "Draw the map")
			if exitCode, output := run(t, root, arguments...); exitCode != cli.ExitCodeSuccess {
				t.Fatalf("add exit code = %d, output %q", exitCode, output)
			}

			_, output := run(t, root, "lint", "draw-the-map")

			var templateLines []string
			for line := range strings.Lines(output) {
				if strings.Contains(line, ": template: ") {
					templateLines = append(templateLines, strings.TrimSuffix(line, "\n"))
				}
			}
			if strings.Join(templateLines, "\n") != strings.Join(tt.expectedLines, "\n") {
				t.Errorf("template findings = %q, expected %q", templateLines, tt.expectedLines)
			}
		})
	}
}
