package tix_test

import (
	"strings"
	"testing"

	"github.com/theroymind/railyard/internal/cli"
)

func TestTypeOptionTakesTheProjectsOwnNames(t *testing.T) {
	tests := []struct {
		name             string
		arguments        []string
		expectedExitCode cli.ExitCode
		expectedRows     []string
		expectedError    string
	}{
		{"ls --type phase lists the epics", []string{"ls", "--type", "phase", "--all"}, cli.ExitCodeSuccess,
			[]string{"enemy-puzzle", "first-fight"}, ""},
		{"ls --type epic still works", []string{"ls", "--type", "epic", "--all"}, cli.ExitCodeSuccess,
			[]string{"enemy-puzzle", "first-fight"}, ""},
		{"ready --type task takes the shared name", []string{"ready", "--type", "task"}, cli.ExitCodeSuccess,
			[]string{"weighted-step-choice"}, ""},
		{"an unknown name lists both vocabularies", []string{"ls", "--type", "milestone"}, cli.ExitCodeUsage, nil,
			`--type "milestone" is invalid: Invalid option: expected one of "epic"|"story"|"task" or one of this project's "phase" (epic), "task" (task)`},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			exitCode, output := run(t, copyVault(t), tt.arguments...)

			if exitCode != tt.expectedExitCode {
				t.Fatalf("exit code = %d, expected %d; output %q", exitCode, tt.expectedExitCode, output)
			}
			if tt.expectedError != "" && !strings.Contains(output, tt.expectedError) {
				t.Errorf("output = %q, expected it to contain %q", output, tt.expectedError)
			}
			for _, row := range tt.expectedRows {
				if !strings.Contains(output, row+" ") {
					t.Errorf("output lacks the row %q:\n%s", row, output)
				}
			}
			lines := strings.Count(output, "\n") - 1
			if tt.expectedError == "" && lines != len(tt.expectedRows) {
				t.Errorf("output has %d rows, expected %d:\n%s", lines, len(tt.expectedRows), output)
			}
		})
	}
}
