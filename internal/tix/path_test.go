package tix_test

import (
	"path/filepath"
	"strings"
	"testing"

	"github.com/theroymind/go-tix/contracts/ticket"
	"github.com/theroymind/railyard-cli"
)

func TestPath(t *testing.T) {
	tests := []struct {
		name             string
		yaml             bool
		arguments        []string
		expectedExitCode cli.ExitCode
		expectedSuffix   string
		mentions         []string
	}{
		{"A vault ticket prints its note's path", false, []string{"path", "weighted-step-choice"}, cli.ExitCodeSuccess,
			filepath.FromSlash("/docs/tasks/weighted-step-choice.md") + "\n", nil},
		{"A unique prefix resolves", false, []string{"path", "weighted"}, cli.ExitCodeSuccess,
			filepath.FromSlash("/docs/tasks/weighted-step-choice.md") + "\n", nil},
		{"A YAML ticket prints its file's path", true, []string{"path", yamlID}, cli.ExitCodeSuccess,
			string(filepath.Separator) + filepath.Join(ticket.DirectoryName, ticket.FileName(yamlID)) + "\n", nil},
		{"A doc isn't a ticket", false, []string{"path", "enemy-goals"}, cli.ExitCodeFailure, "", []string{"enemy-goals"}},
		{"An unknown ticket is refused", false, []string{"path", "no-such-ticket"}, cli.ExitCodeFailure, "", []string{"no-such-ticket"}},
		{"No reference is a usage error", false, []string{"path"}, cli.ExitCodeUsage, "",
			[]string{"expected exactly one reference"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			root := ""
			if tt.yaml {
				root, _ = yamlRepository(t)
			} else {
				root = copyVault(t)
			}

			exitCode, output := run(t, root, tt.arguments...)

			if exitCode != tt.expectedExitCode {
				t.Fatalf("exit code = %d, expected %d; output %q", exitCode, tt.expectedExitCode, output)
			}
			if tt.expectedExitCode == cli.ExitCodeSuccess &&
				(!filepath.IsAbs(strings.TrimSpace(output)) || !strings.HasSuffix(output, tt.expectedSuffix)) {
				t.Errorf("output = %q, expected an absolute path ending %q", output, tt.expectedSuffix)
			}
			for _, mention := range tt.mentions {
				if !strings.Contains(output, mention) {
					t.Errorf("output %q does not mention %q", output, mention)
				}
			}
		})
	}
}
