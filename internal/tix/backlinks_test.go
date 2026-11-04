package tix_test

import (
	"strings"
	"testing"

	"github.com/theroymind/railyard-cli"
)

func TestBacklinks(t *testing.T) {
	tests := []struct {
		name             string
		arguments        []string
		expectedExitCode cli.ExitCode
		expectedOutput   string
		mentions         []string
	}{
		{"Every note linking to a doc is listed by slug, like grep -rl '[[enemy-goals' docs",
			[]string{"backlinks", "enemy-goals"}, cli.ExitCodeSuccess,
			"README\nenemy-ai\nenemy-puzzle\ngoal-scoring\ngolem-guards\nharpy-hastes\nsim-queries\nweighted-step-choice\n", nil},
		{"A ticket's backlinks are wikilinks only, not bare slugs in properties", []string{"backlinks", "weighted-step-choice"}, cli.ExitCodeSuccess,
			"enemy-puzzle\n", nil},
		{"A unique prefix resolves", []string{"backlinks", "weighted"}, cli.ExitCodeSuccess, "enemy-puzzle\n", nil},
		{"A slug tix doesn't map is taken as written", []string{"backlinks", "mana-pool", "--json"}, cli.ExitCodeSuccess,
			"[\n  \"README\",\n  \"enemy-ai\",\n  \"enemy-choice-is-a-weighted-argmax\",\n  \"enemy-goals\",\n  \"first-fight\",\n  \"mana\",\n  \"wraith-casts\"\n]\n", nil},
		{"No backlinks prints nothing", []string{"backlinks", "nobody-links-here"}, cli.ExitCodeSuccess, "", nil},
		{"No backlinks as JSON is an empty array", []string{"backlinks", "--json", "nobody-links-here"}, cli.ExitCodeSuccess, "[]\n", nil},
		{"An ambiguous prefix is refused", []string{"backlinks", "enemy"}, cli.ExitCodeFailure, "", []string{`"enemy"`, "ambiguous"}},
		{"No reference is a usage error", []string{"backlinks"}, cli.ExitCodeUsage, "", []string{"expected exactly one reference"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			root := copyVault(t)

			exitCode, output := run(t, root, tt.arguments...)

			if exitCode != tt.expectedExitCode {
				t.Fatalf("exit code = %d, expected %d; output %q", exitCode, tt.expectedExitCode, output)
			}
			if tt.expectedExitCode == cli.ExitCodeSuccess && output != tt.expectedOutput {
				t.Errorf("output =\n%s\nexpected\n%s", output, tt.expectedOutput)
			}
			for _, mention := range tt.mentions {
				if !strings.Contains(output, mention) {
					t.Errorf("output %q does not mention %q", output, mention)
				}
			}
		})
	}
}

func TestBacklinksOnTheYAMLBackend(t *testing.T) {
	root, _ := yamlRepository(t)

	exitCode, output := run(t, root, "backlinks", yamlID)

	if exitCode != cli.ExitCodeFailure || !strings.Contains(output, "has no docs") {
		t.Errorf("exit code = %d, output %q; expected failure saying the backend has no docs", exitCode, output)
	}
}
