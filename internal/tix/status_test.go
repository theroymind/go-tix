package tix_test

import (
	"path/filepath"
	"strings"
	"testing"

	"github.com/theroymind/railyard-cli"
)

func TestStartAndReviewOnTheStarterVault(t *testing.T) {
	tests := []struct {
		name     string
		command  string
		expected string
	}{
		{"start writes the in-progress note status", "start", "status: in-progress\n"},
		{"review writes the in-review note status", "review", "status: in-review\n"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			root := t.TempDir()
			if exitCode, output := run(t, root, "init", "--prefix", "tx"); exitCode != cli.ExitCodeSuccess {
				t.Fatalf("init exit code = %d, output %q", exitCode, output)
			}
			exitCode, output := run(t, root, "add", "--type", "task", "Draw the map")
			if exitCode != cli.ExitCodeSuccess {
				t.Fatalf("add exit code = %d, output %q", exitCode, output)
			}
			id := strings.TrimSpace(output)
			_, pathOutput := run(t, root, "path", id)
			path := strings.TrimSpace(pathOutput)
			original := readFile(t, path)

			exitCode, output = run(t, root, tt.command, id)

			if exitCode != cli.ExitCodeSuccess {
				t.Fatalf("exit code = %d, output %q", exitCode, output)
			}
			expected := strings.Replace(original, "status: todo\n", tt.expected, 1)
			if actual := readFile(t, path); actual != expected {
				t.Errorf("note =\n%s\nexpected\n%s", actual, expected)
			}
		})
	}
}

func TestReviewRefusesAStatusTheVaultDoesNotMap(t *testing.T) {
	root := copyVault(t)
	path := filepath.Join(root, "docs", "tasks", "golem-guards.md")
	original := readFile(t, path)

	exitCode, output := run(t, root, "review", "golem-guards")

	if exitCode != cli.ExitCodeFailure {
		t.Fatalf("exit code = %d, output %q", exitCode, output)
	}
	if !strings.Contains(output, "in_review") {
		t.Errorf("output %q does not name the status", output)
	}
	if actual := readFile(t, path); actual != original {
		t.Errorf("note changed:\n%s", actual)
	}
}

func TestReviewOnYAML(t *testing.T) {
	root, path := yamlRepository(t)
	original := readFile(t, path)

	exitCode, output := run(t, root, "review", yamlID)

	if exitCode != cli.ExitCodeSuccess {
		t.Fatalf("exit code = %d, output %q", exitCode, output)
	}
	expected := strings.Replace(original, "status: open\n", "status: in_review\n", 1)
	actual := readFile(t, path)
	if updatedAtLine.ReplaceAllString(actual, "") != updatedAtLine.ReplaceAllString(expected, "") {
		t.Errorf("ticket =\n%s\nexpected only status and updated_at to change from\n%s", actual, original)
	}
}

func TestStartOnSeveralTickets(t *testing.T) {
	tests := []struct {
		name             string
		references       []string
		expectedExitCode cli.ExitCode
		expectedStatus   string
	}{
		{"start moves every ticket it names", []string{"draw-the-map", "load-the-tiles"}, cli.ExitCodeSuccess, "in-progress"},
		{"A reference that matches nothing changes none", []string{"draw-the-map", "load-the-tiles", "nope"}, cli.ExitCodeFailure, "todo"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			root := t.TempDir()
			if exitCode, output := run(t, root, "init", "--prefix", "tx"); exitCode != cli.ExitCodeSuccess {
				t.Fatalf("init exit code = %d, output %q", exitCode, output)
			}
			for _, title := range []string{"Draw the map", "Load the tiles"} {
				if exitCode, output := run(t, root, "add", "--type", "task", title); exitCode != cli.ExitCodeSuccess {
					t.Fatalf("add exit code = %d, output %q", exitCode, output)
				}
			}

			exitCode, output := run(t, root, append([]string{"start"}, tt.references...)...)

			if exitCode != tt.expectedExitCode {
				t.Fatalf("exit code = %d, expected %d; output %q", exitCode, tt.expectedExitCode, output)
			}
			for _, id := range []string{"draw-the-map", "load-the-tiles"} {
				_, pathOutput := run(t, root, "path", id)
				if text := readFile(t, strings.TrimSpace(pathOutput)); !strings.Contains(text, "status: "+tt.expectedStatus+"\n") {
					t.Errorf("%s does not say status: %s", id, tt.expectedStatus)
				}
			}
		})
	}
}
