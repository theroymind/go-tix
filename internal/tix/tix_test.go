package tix_test

import (
	"bytes"
	"os"
	"path/filepath"
	"regexp"
	"testing"

	"github.com/theroymind/go-tix/contracts/ticket"
	"github.com/theroymind/go-tix/internal/tix"
	"github.com/theroymind/go-tix/internal/tix/backend"
	"github.com/theroymind/go-tix/internal/tix/backend/vaultstore"
	"github.com/theroymind/go-tix/internal/tix/backend/yamlstore"
	"github.com/theroymind/railyard-cli"
)

const (
	vaultFixture = "backend/vaultstore/testdata/odin-tbc"
	yamlFixture  = "backend/yamlstore/testdata/ticket.yml"
	yamlID       = "bs-k3f9"
)

var updatedAtLine = regexp.MustCompile(`(?m)^updated_at: .*$`)

func TestMain(m *testing.M) {
	backend.Register(backend.KindYAML, yamlstore.Open)
	backend.Register(backend.KindVault, vaultstore.Open)
	os.Exit(m.Run())
}

func TestProgramDispatch(t *testing.T) {
	tests := []struct {
		name             string
		arguments        []string
		expectedExitCode cli.ExitCode
		expectedOutput   string
	}{
		{"No arguments print the full usage as a usage error", nil, cli.ExitCodeUsage, tix.Program.Usage},
		{"help prints the full usage and succeeds", []string{"help"}, cli.ExitCodeSuccess, tix.Program.Usage},
		{"An unknown command is one line pointing at help", []string{"shwo", "x"}, cli.ExitCodeUsage,
			"tix: unknown command \"shwo\"; run \"tix help\" for usage\n"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			exitCode, output := run(t, t.TempDir(), tt.arguments...)

			if exitCode != tt.expectedExitCode {
				t.Fatalf("exit code = %d, expected %d; output %q", exitCode, tt.expectedExitCode, output)
			}
			if output != tt.expectedOutput {
				t.Errorf("output = %q, expected %q", output, tt.expectedOutput)
			}
		})
	}
}

// run runs tix in directory and returns its exit code and everything it printed.
func run(t *testing.T, directory string, arguments ...string) (cli.ExitCode, string) {
	t.Helper()
	t.Chdir(directory)
	var stdout, stderr bytes.Buffer
	exitCode := tix.Run(arguments, bytes.NewReader(nil), &stdout, &stderr)
	return exitCode, stdout.String() + stderr.String()
}

// copyVault copies the fixture vault into a temporary directory, so that no test writes to testdata.
func copyVault(t *testing.T) string {
	t.Helper()
	root := t.TempDir()
	if err := os.CopyFS(root, os.DirFS(vaultFixture)); err != nil {
		t.Fatal(err)
	}
	return root
}

// yamlRepository makes a temporary .tix directory holding the yamlstore fixture ticket and returns the repository
// root and the ticket's path.
func yamlRepository(t *testing.T) (string, string) {
	t.Helper()
	text, err := os.ReadFile(yamlFixture)
	if err != nil {
		t.Fatal(err)
	}
	root := t.TempDir()
	if exitCode, output := run(t, root, "init", "--prefix", "bs", "--backend", "yaml"); exitCode != cli.ExitCodeSuccess {
		t.Fatalf("tix init exit code = %d, output %q", exitCode, output)
	}
	path := filepath.Join(root, ticket.DirectoryName, ticket.FileName(yamlID))
	if err := os.WriteFile(path, text, 0o644); err != nil {
		t.Fatal(err)
	}
	return root, path
}

func readFile(t *testing.T, path string) string {
	t.Helper()
	text, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	return string(text)
}
