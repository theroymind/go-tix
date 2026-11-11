package tix_test

import (
	"strings"
	"testing"

	"github.com/theroymind/railyard-cli"
)

func TestReadyParentKeepsOneParentsTickets(t *testing.T) {
	tests := []struct {
		name         string
		parent       string
		expectedRows []string
	}{
		{"The ready task's own parent keeps it", "enemy-puzzle", []string{"weighted-step-choice"}},
		{"Another parent leaves nothing", "first-fight", nil},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			exitCode, output := run(t, copyVault(t), "ready", "--parent", tt.parent)

			if exitCode != cli.ExitCodeSuccess {
				t.Fatalf("exit code = %d, output %q", exitCode, output)
			}
			for _, row := range tt.expectedRows {
				if !strings.Contains(output, row+" ") {
					t.Errorf("output lacks the row %q:\n%s", row, output)
				}
			}
			if lines := strings.Count(output, "\n") - 1; lines != len(tt.expectedRows) {
				t.Errorf("output has %d rows, expected %d:\n%s", lines, len(tt.expectedRows), output)
			}
		})
	}
}
