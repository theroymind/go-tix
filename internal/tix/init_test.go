package tix_test

import (
	"bytes"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/theroymind/go-tix/contracts/ticket"
	"github.com/theroymind/go-tix/internal/tix"
	"github.com/theroymind/railyard-cli"
)

const starterVaultConfig = `prefix: %s
backend: vault
vault:
  root: %s
  type_property: type
  tickets:
    feature:
      ticket_type: epic
      statuses:
        active: in_progress
        done: completed
        todo: open
    task:
      ticket_type: task
      statuses:
        active: in_progress
        done: completed
        todo: open
      fields:
        depends_on: dependencies
        feature: parent
  docs:
    idea:
      statuses:
        - open
      template: templates/idea.md
      directory: ideas
  criteria_heading: Acceptance criteria
  ticket_templates:
    feature: templates/feature.md
    task: templates/task.md
  ticket_directories:
    feature: features/{slug}
    task: '{parent}/tasks'
  uncategorized_directory: features/uncategorized
  files_heading: Files
  required_sections:
    feature:
      - Description
      - Acceptance criteria
    task:
      - Description
      - Files
      - Acceptance criteria
`

func starterConfig(prefix, root string) string {
	return fmt.Sprintf(starterVaultConfig, prefix, root)
}

func TestInit(t *testing.T) {
	tests := []struct {
		name             string
		arguments        []string
		input            string
		expectedExitCode cli.ExitCode
		expectedConfig   string
		mentions         []string
	}{
		{"--prefix writes the starter vault without asking anything", []string{"init", "--prefix", "bs"}, "",
			cli.ExitCodeSuccess, starterConfig("bs", "docs"), nil},
		{"--docs roots the starter vault elsewhere", []string{"init", "--prefix", "bs", "--docs", "notes"}, "",
			cli.ExitCodeSuccess, starterConfig("bs", "notes"), nil},
		{"--docs outside this directory is refused", []string{"init", "--prefix", "bs", "--docs", "../notes"}, "",
			cli.ExitCodeFailure, "", []string{`"../notes" is not a directory inside this one`}},
		{"--docs with the YAML backend is refused", []string{"init", "--prefix", "bs", "--backend", "yaml", "--docs", "notes"}, "",
			cli.ExitCodeFailure, "", []string{"needs --backend vault"}},
		{"--backend yaml writes the prefix alone, as tix v1 did", []string{"init", "--prefix", "bs", "--backend", "yaml"}, "",
			cli.ExitCodeSuccess, "prefix: bs\n", nil},
		{"An unknown --backend is refused", []string{"init", "--prefix", "bs", "--backend", "sql"}, "",
			cli.ExitCodeFailure, "", []string{`"sql" is invalid`}},
		{"Without --prefix, a blank backend answer takes the vault", []string{"init"}, "bs\n\n\n",
			cli.ExitCodeSuccess, starterConfig("bs", "docs"),
			[]string{"Ticket id prefix: ", "Backend (yaml or vault) [vault]: ", "Docs directory, relative to this one [docs]: ", "wrote "}},
		{"An invalid prefix is asked again", []string{"init"}, "BS\nbs\nyaml\n",
			cli.ExitCodeSuccess, "prefix: bs\n", []string{`"BS" is not a prefix`}},
		{"An unknown backend is asked again", []string{"init"}, "bs\nsql\nyaml\n",
			cli.ExitCodeSuccess, "prefix: bs\n", []string{`"sql" is invalid`}},
		{"A docs directory outside this one is asked again",
			[]string{"init"}, "odin\nvault\n../docs\nnotes\n",
			cli.ExitCodeSuccess, starterConfig("odin", "notes"), []string{`"../docs" is not a directory inside this one`}},
		{"Input that ends early writes nothing", []string{"init"}, "bs\n",
			cli.ExitCodeFailure, "", []string{"input ended before setup finished"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			root := t.TempDir()

			exitCode, output := runWithInput(t, root, tt.input, tt.arguments...)

			if exitCode != tt.expectedExitCode {
				t.Fatalf("exit code = %d, expected %d; output %q", exitCode, tt.expectedExitCode, output)
			}
			for _, mention := range tt.mentions {
				if !strings.Contains(output, mention) {
					t.Errorf("output = %q, want it to mention %q", output, mention)
				}
			}
			tixDirectory := filepath.Join(root, ticket.DirectoryName)
			if tt.expectedConfig == "" {
				if _, err := os.Stat(tixDirectory); !errors.Is(err, fs.ErrNotExist) {
					t.Errorf("stat %s error = %v, expected the directory not to exist", tixDirectory, err)
				}
				return
			}
			configPath := filepath.Join(tixDirectory, ticket.ConfigFileName)
			config := readFile(t, configPath)
			if config != tt.expectedConfig {
				t.Errorf("config =\n%s\nexpected\n%s", config, tt.expectedConfig)
			}
		})
	}
}

func runWithInput(t *testing.T, directory, input string, arguments ...string) (cli.ExitCode, string) {
	t.Helper()
	t.Chdir(directory)
	var stdout, stderr bytes.Buffer
	stdin := strings.NewReader(input)
	exitCode := tix.Program.Run(arguments, stdin, &stdout, &stderr)
	return exitCode, stdout.String() + stderr.String()
}

func TestStarterVaultTakesItsFirstNotes(t *testing.T) {
	root := t.TempDir()
	if exitCode, output := run(t, root, "init", "--prefix", "bs"); exitCode != cli.ExitCodeSuccess {
		t.Fatalf("tix init exit code = %d, output %q", exitCode, output)
	}
	steps := []struct {
		arguments    []string
		expectedPath string
	}{
		{[]string{"add", "--type", "epic", "First feature"}, "docs/features/first-feature/first-feature.md"},
		{[]string{"add", "--type", "task", "--parent", "first-feature", "First task"}, "docs/features/first-feature/tasks/first-task.md"},
		{[]string{"add", "--type", "task", "Loose task"}, "docs/features/uncategorized/tasks/loose-task.md"},
		{[]string{"doc", "add", "--type", "idea", "First idea"}, "docs/ideas/first-idea.md"},
	}
	for _, step := range steps {
		if exitCode, output := run(t, root, step.arguments...); exitCode != cli.ExitCodeSuccess {
			t.Fatalf("tix %v exit code = %d, output %q", step.arguments, exitCode, output)
		}
		text := readFile(t, filepath.Join(root, step.expectedPath))
		title := step.arguments[len(step.arguments)-1]
		if !strings.Contains(text, "\n# "+title+"\n") {
			t.Errorf("tix %v wrote a note without the H1 %q:\n%s", step.arguments, title, text)
		}
	}
}

func TestChangingATasksFeatureMovesItsNote(t *testing.T) {
	tests := []struct {
		name         string
		pair         string
		expectedPath string
		emptiedPath  string
	}{
		{"By the property the vault names", "feature=second-feature",
			"docs/features/second-feature/tasks/first-task.md", "docs/features/uncategorized"},
		{"By the parent field", "parent=second-feature",
			"docs/features/second-feature/tasks/first-task.md", "docs/features/uncategorized"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			root := t.TempDir()
			commands := [][]string{
				{"init", "--prefix", "bs"},
				{"add", "--type", "epic", "Second feature"},
				{"add", "--type", "task", "First task"},
				{"set", "first-task", tt.pair},
			}
			for _, arguments := range commands {
				if exitCode, output := run(t, root, arguments...); exitCode != cli.ExitCodeSuccess {
					t.Fatalf("tix %v exit code = %d, output %q", arguments, exitCode, output)
				}
			}
			text := readFile(t, filepath.Join(root, tt.expectedPath))
			if !strings.Contains(text, "feature: second-feature\n") {
				t.Errorf("moved note doesn't name its new feature:\n%s", text)
			}
			if _, err := os.Stat(filepath.Join(root, tt.emptiedPath)); !errors.Is(err, fs.ErrNotExist) {
				t.Errorf("stat %s error = %v, expected the emptied directory to be removed", tt.emptiedPath, err)
			}
		})
	}
}

func TestInitRefusesToOverwriteATemplate(t *testing.T) {
	root := t.TempDir()
	existing := filepath.Join(root, "docs", "templates", "task.md")
	if err := os.MkdirAll(filepath.Dir(existing), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(existing, []byte("mine\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	exitCode, output := run(t, root, "init", "--prefix", "bs")

	if exitCode != cli.ExitCodeFailure || !strings.Contains(output, "already exists") {
		t.Fatalf("exit code = %d, output %q; expected a failure saying the template already exists", exitCode, output)
	}
	if readFile(t, existing) != "mine\n" {
		t.Errorf("tix init overwrote %s", existing)
	}
	if _, err := os.Stat(filepath.Join(root, ticket.DirectoryName)); !errors.Is(err, fs.ErrNotExist) {
		t.Errorf("tix init created .tix after refusing")
	}
}
