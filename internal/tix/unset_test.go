package tix_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/theroymind/railyard/internal/cli"
)

func TestUnsetOnAVault(t *testing.T) {
	const (
		task            = "docs/tasks/golem-guards.md"
		decision        = "docs/decisions/turn-order-is-a-wait-counter.md"
		descriptionLine = "description: Old summary words.\n"
	)
	tests := []struct {
		name             string
		arguments        []string
		note             string
		setup            [2]string
		expectedExitCode cli.ExitCode
		removed          []string
		mentions         []string
	}{
		{"A doc's description key is removed and its body kept",
			[]string{"unset", "turn-order-is-a-wait-counter", "description"}, decision,
			[2]string{"status: proposed\n", "status: proposed\n" + descriptionLine}, cli.ExitCodeSuccess,
			[]string{descriptionLine}, nil},
		{"Several keys are removed together",
			[]string{"unset", "turn", "decided_by", "tasks"}, decision, [2]string{}, cli.ExitCodeSuccess,
			[]string{"decided_by:\n", "tasks: []\n"}, nil},
		{"A ticket's description is its body, so unset refuses it",
			[]string{"unset", "golem-guards", "description"}, task,
			[2]string{"status: todo\n", "status: todo\n" + descriptionLine}, cli.ExitCodeFailure,
			nil, []string{"description", "body"}},
		{"A required key is refused",
			[]string{"unset", "golem-guards", "status"}, task, [2]string{}, cli.ExitCodeFailure,
			nil, []string{"status", "required"}},
		{"A key the note doesn't have is named and nothing is written",
			[]string{"unset", "golem-guards", "scope", "owner"}, task, [2]string{}, cli.ExitCodeFailure,
			nil, []string{"owner"}},
		{"No key is a usage error",
			[]string{"unset", "golem-guards"}, task, [2]string{}, cli.ExitCodeUsage,
			nil, []string{"usage:"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			root := copyVault(t)
			path := filepath.Join(root, tt.note)
			original := readFile(t, path)
			if tt.setup[0] != "" {
				if !strings.Contains(original, tt.setup[0]) {
					t.Fatalf("fixture lacks %q", tt.setup[0])
				}
				original = strings.Replace(original, tt.setup[0], tt.setup[1], 1)
				if err := os.WriteFile(path, []byte(original), 0o644); err != nil {
					t.Fatal(err)
				}
			}

			exitCode, output := run(t, root, tt.arguments...)

			if exitCode != tt.expectedExitCode {
				t.Fatalf("exit code = %d, expected %d; output %q", exitCode, tt.expectedExitCode, output)
			}
			for _, mention := range tt.mentions {
				if !strings.Contains(output, mention) {
					t.Errorf("output %q does not mention %q", output, mention)
				}
			}
			expected := original
			for _, line := range tt.removed {
				if !strings.Contains(expected, line) {
					t.Fatalf("fixture lacks %q", line)
				}
				expected = strings.Replace(expected, line, "", 1)
			}
			if actual := readFile(t, path); actual != expected {
				t.Errorf("note =\n%s\nexpected\n%s", actual, expected)
			}
		})
	}
}

func TestUnsetOnYAML(t *testing.T) {
	tests := []struct {
		name             string
		arguments        []string
		expectedExitCode cli.ExitCode
		replace          [2]string
		mentions         []string
	}{
		{"An optional field is removed and updated_at changes",
			[]string{"unset", yamlID, "parent"}, cli.ExitCodeSuccess,
			[2]string{"parent: bs-x8vd\n", ""}, nil},
		{"A required field is refused",
			[]string{"unset", yamlID, "title"}, cli.ExitCodeFailure,
			[2]string{}, []string{"title", "required"}},
		{"A field the file doesn't have is named",
			[]string{"unset", yamlID, "owner"}, cli.ExitCodeFailure,
			[2]string{}, []string{"owner"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			root, path := yamlRepository(t)
			original := readFile(t, path)

			exitCode, output := run(t, root, tt.arguments...)

			if exitCode != tt.expectedExitCode {
				t.Fatalf("exit code = %d, expected %d; output %q", exitCode, tt.expectedExitCode, output)
			}
			for _, mention := range tt.mentions {
				if !strings.Contains(output, mention) {
					t.Errorf("output %q does not mention %q", output, mention)
				}
			}
			actual := readFile(t, path)
			if tt.replace[0] == "" {
				if actual != original {
					t.Errorf("ticket =\n%s\nexpected it unchanged", actual)
				}
				return
			}
			expected := strings.Replace(original, tt.replace[0], tt.replace[1], 1)
			if updatedAtLine.ReplaceAllString(actual, "") != updatedAtLine.ReplaceAllString(expected, "") {
				t.Errorf("ticket =\n%s\nexpected\n%s", actual, expected)
			}
			if updatedAtLine.FindString(actual) == updatedAtLine.FindString(original) {
				t.Errorf("updated_at unchanged: %s", updatedAtLine.FindString(actual))
			}
		})
	}
}
