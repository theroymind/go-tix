package tix_test

import (
	"path/filepath"
	"strings"
	"testing"

	"github.com/theroymind/railyard-cli"
)

func TestDoneOnAVault(t *testing.T) {
	tests := []struct {
		name             string
		arguments        []string
		note             string
		expectedExitCode cli.ExitCode
		replace          [2]string
		mentions         []string
	}{
		{"Every box ticked changes only the status line",
			[]string{"done", "golem-guards"}, "golem-guards.md", cli.ExitCodeSuccess,
			[2]string{"status: todo\n", "status: done\n"}, nil},
		{"An unticked box refuses, naming every unticked criterion, and leaves the note alone",
			[]string{"done", "weighted-step-choice"}, "weighted-step-choice.md", cli.ExitCodeFailure,
			[2]string{}, []string{"weighted-step-choice", "--force", "2: The argmax takes the higher score", "3: Only the winning step", "4: A parity test"}},
		{"--force completes a task with an unticked box",
			[]string{"done", "--force", "weighted-step-choice"}, "weighted-step-choice.md", cli.ExitCodeSuccess,
			[2]string{"status: todo\n", "status: done\n"}, nil},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			root := copyVault(t)
			path := filepath.Join(root, "docs", "tasks", tt.note)
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
			expected := original
			if tt.replace[0] != "" {
				expected = strings.Replace(original, tt.replace[0], tt.replace[1], 1)
			}
			if actual := readFile(t, path); actual != expected {
				t.Errorf("note =\n%s\nexpected\n%s", actual, expected)
			}
		})
	}
}

func TestDoneOnYAMLIgnoresCriteriaWithoutADoneState(t *testing.T) {
	root, path := yamlRepository(t)
	original := readFile(t, path)

	exitCode, output := run(t, root, "done", yamlID)

	if exitCode != cli.ExitCodeSuccess {
		t.Fatalf("exit code = %d, output %q", exitCode, output)
	}
	actual := readFile(t, path)
	if actual == original {
		t.Fatal("ticket file unchanged")
	}
	expected := strings.Replace(original, "status: open\n", "status: completed\n", 1)
	if updatedAtLine.ReplaceAllString(actual, "") != updatedAtLine.ReplaceAllString(expected, "") {
		t.Errorf("ticket =\n%s\nexpected only status and updated_at to change from\n%s", actual, original)
	}
	if updatedAtLine.FindString(actual) == updatedAtLine.FindString(original) {
		t.Errorf("updated_at unchanged: %s", updatedAtLine.FindString(actual))
	}
}

func TestDoneOnSeveralTickets(t *testing.T) {
	tests := []struct {
		name             string
		arguments        []string
		expectedExitCode cli.ExitCode
		expectedStatuses map[string]string
		mentions         []string
	}{
		{"Every ticket met completes them all",
			[]string{"done", "golem-guards", "weighted-step-choice", "--force"}, cli.ExitCodeSuccess,
			map[string]string{"golem-guards.md": "done", "weighted-step-choice.md": "done"}, nil},
		{"One ticket unmet changes none, naming it",
			[]string{"done", "golem-guards", "weighted-step-choice"}, cli.ExitCodeFailure,
			map[string]string{"golem-guards.md": "todo", "weighted-step-choice.md": "todo"},
			[]string{"weighted-step-choice has unmet criteria", "2: The argmax"}},
		{"A reference that matches nothing changes none",
			[]string{"done", "golem-guards", "nope"}, cli.ExitCodeFailure,
			map[string]string{"golem-guards.md": "todo"}, []string{"nope"}},
		{"No reference is a usage error",
			[]string{"done"}, cli.ExitCodeUsage, nil, []string{"expected at least one ticket reference"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			root := copyVault(t)

			exitCode, output := run(t, root, tt.arguments...)

			if exitCode != tt.expectedExitCode {
				t.Fatalf("exit code = %d, expected %d; output %q", exitCode, tt.expectedExitCode, output)
			}
			for _, mention := range tt.mentions {
				if !strings.Contains(output, mention) {
					t.Errorf("output %q does not mention %q", output, mention)
				}
			}
			for note, status := range tt.expectedStatuses {
				text := readFile(t, filepath.Join(root, "docs", "tasks", note))
				if !strings.Contains(text, "status: "+status+"\n") {
					t.Errorf("%s does not say status: %s", note, status)
				}
			}
		})
	}
}

func TestDoneWithHandChecks(t *testing.T) {
	const handCheck = "Hand check: The golem looks heavy"
	tests := []struct {
		name             string
		setup            [][]string
		arguments        []string
		expectedExitCode cli.ExitCode
		expectedStatus   string
		mentions         []string
	}{
		{"Only hand checks open completes the task and names each one",
			[][]string{{"ac", "add", "golem-guards", handCheck}},
			[]string{"done", "golem-guards"}, cli.ExitCodeSuccess, "status: done\n",
			[]string{"golem-guards 4 " + handCheck}},
		{"An open criterion beside a hand check still refuses, naming only the criterion",
			[][]string{{"ac", "add", "weighted-step-choice", handCheck}, {"tick", "weighted-step-choice", "2", "3"}},
			[]string{"done", "weighted-step-choice"}, cli.ExitCodeFailure, "status: todo\n",
			[]string{"4: A parity test"}},
		{"A ticked hand check is not named",
			[][]string{{"ac", "add", "golem-guards", handCheck}, {"tick", "golem-guards", "4"}},
			[]string{"done", "golem-guards"}, cli.ExitCodeSuccess, "status: done\n", nil},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			root := copyVault(t)
			for _, arguments := range tt.setup {
				if exitCode, output := run(t, root, arguments...); exitCode != cli.ExitCodeSuccess {
					t.Fatalf("%v exit code = %d, output %q", arguments, exitCode, output)
				}
			}
			note := tt.arguments[len(tt.arguments)-1] + ".md"

			exitCode, output := run(t, root, tt.arguments...)

			if exitCode != tt.expectedExitCode {
				t.Fatalf("exit code = %d, expected %d; output %q", exitCode, tt.expectedExitCode, output)
			}
			for _, mention := range tt.mentions {
				if !strings.Contains(output, mention) {
					t.Errorf("output %q does not mention %q", output, mention)
				}
			}
			if tt.mentions == nil && output != "" {
				t.Errorf("output = %q, expected none", output)
			}
			if exitCode == cli.ExitCodeFailure && strings.Contains(output, handCheck) {
				t.Errorf("output %q names the hand check as unmet", output)
			}
			if actual := readFile(t, filepath.Join(root, "docs", "tasks", note)); !strings.Contains(actual, tt.expectedStatus) {
				t.Errorf("note =\n%s\nexpected it to hold %q", actual, tt.expectedStatus)
			}
		})
	}
}
