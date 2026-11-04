package tix_test

import (
	"bytes"
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/theroymind/go-tix/contracts/ticket"
	"github.com/theroymind/go-tix/internal/tix"
	"github.com/theroymind/railyard-cli"
)

const defaultVaultConfig = `prefix: odin
backend: vault
vault:
  root: docs
  type_property: type
  tickets:
    task:
      ticket_type: task
      statuses:
        done: completed
        in_progress: in_progress
        todo: open
  criteria_heading: Acceptance criteria
`

const customVaultConfig = `prefix: odin
backend: vault
vault:
  root: notes
  type_property: type
  tickets:
    task:
      ticket_type: task
      statuses:
        done: completed
        in_progress: in_progress
        todo: open
  criteria_heading: Acceptance criteria
embeddings:
  endpoint: http://localhost:11434/v1
  model: all-minilm
`

func TestInit(t *testing.T) {
	tests := []struct {
		name             string
		arguments        []string
		input            string
		expectedExitCode cli.ExitCode
		expectedConfig   string
		mentions         []string
	}{
		{"--prefix writes the config without asking anything", []string{"init", "--prefix", "bs"}, "",
			cli.ExitCodeSuccess, "prefix: bs\n", nil},
		{"Without --prefix, blank answers take the YAML backend and haiku search", []string{"init"}, "bs\n\n\n",
			cli.ExitCodeSuccess, "prefix: bs\nclaude:\n  model: haiku\n",
			[]string{"Ticket id prefix: ", "Backend (yaml or vault) [yaml]: ", "Search with haiku", "[haiku]: ", "wrote "}},
		{"Search can be turned off", []string{"init"}, "bs\n\nnone\n",
			cli.ExitCodeSuccess, "prefix: bs\n", nil},
		{"An invalid prefix is asked again", []string{"init"}, "BS\nbs\n\nnone\n",
			cli.ExitCodeSuccess, "prefix: bs\n", []string{`"BS" is not a prefix`}},
		{"An unknown backend is asked again", []string{"init"}, "bs\nsql\nyaml\nnone\n",
			cli.ExitCodeSuccess, "prefix: bs\n", []string{`"sql" is invalid`}},
		{"The vault backend asks only for the docs directory", []string{"init"}, "odin\nvault\n\nnone\n",
			cli.ExitCodeSuccess, defaultVaultConfig, []string{"Docs directory, relative to this one [docs]: "}},
		{"An embeddings endpoint asks for its model",
			[]string{"init"}, "odin\nvault\nnotes\nhttp://localhost:11434/v1\nall-minilm\n",
			cli.ExitCodeSuccess, customVaultConfig, []string{"Embeddings model: "}},
		{"Invalid answers are asked again",
			[]string{"init"}, "odin\nvault\n../docs\nnotes\nlocalhost\nhttp://localhost:11434/v1\n\nall-minilm\n",
			cli.ExitCodeSuccess, customVaultConfig,
			[]string{`"../docs" is not a directory inside this one`, `"localhost" is not haiku, none, or an absolute URL`, "expected an answer"}},
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
