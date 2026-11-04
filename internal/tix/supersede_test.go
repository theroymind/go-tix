package tix_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/theroymind/railyard-cli"
)

func TestSupersede(t *testing.T) {
	const old = "docs/decisions/turn-order-is-a-wait-counter.md"
	tests := []struct {
		name             string
		arguments        []string
		expectedExitCode cli.ExitCode
		replace          [2]string
		mentions         []string
	}{
		{"The old decision's status and superseded_by change, and nothing else",
			[]string{"supersede", "turn-order-is-a-wait-counter", "enemy-choice-is-a-weighted-argmax"}, cli.ExitCodeSuccess,
			[2]string{"status: proposed\ndecided_by:\ndate: \"2026-10-30\"\ntasks: []\n",
				"status: superseded\ndecided_by:\ndate: \"2026-10-30\"\ntasks: []\nsuperseded_by: enemy-choice-is-a-weighted-argmax\n"}, nil},
		{"Prefixes resolve",
			[]string{"supersede", "turn", "enemy-choice"}, cli.ExitCodeSuccess,
			[2]string{"status: proposed\ndecided_by:\ndate: \"2026-10-30\"\ntasks: []\n",
				"status: superseded\ndecided_by:\ndate: \"2026-10-30\"\ntasks: []\nsuperseded_by: enemy-choice-is-a-weighted-argmax\n"}, nil},
		{"A new decision that doesn't exist writes nothing",
			[]string{"supersede", "turn", "nothing"}, cli.ExitCodeFailure, [2]string{}, []string{"no doc matches", `"nothing"`}},
		{"A ticket isn't a decision",
			[]string{"supersede", "golem-guards", "enemy-choice"}, cli.ExitCodeFailure, [2]string{}, []string{"no doc matches", `"golem-guards"`}},
		{"A doc of a type without a superseded status writes nothing",
			[]string{"supersede", "enemy-ai", "enemy-goals"}, cli.ExitCodeFailure, [2]string{}, []string{"own type", "enemy-ai", "idea", "spec"}},
		{"A decision can't supersede itself",
			[]string{"supersede", "turn", "turn"}, cli.ExitCodeFailure, [2]string{}, []string{"turn-order-is-a-wait-counter", "itself"}},
		{"One reference is a usage error",
			[]string{"supersede", "turn"}, cli.ExitCodeUsage, [2]string{}, []string{"expected supersede OLD NEW"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			root := copyVault(t)
			before := snapshot(t, root)
			path := filepath.Join(root, old)
			original := readFile(t, path)
			if tt.replace[0] != "" && !strings.Contains(original, tt.replace[0]) {
				t.Fatalf("fixture lacks %q", tt.replace[0])
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
			if tt.replace[0] != "" {
				before[path] = strings.Replace(original, tt.replace[0], tt.replace[1], 1)
			}
			if after := snapshot(t, root); !equalSnapshots(before, after) {
				t.Errorf("notes changed beyond the old decision's frontmatter:\n%s", snapshotDiff(before, after))
			}
		})
	}
}

func TestSupersedeOnATypeWithoutTheStatus(t *testing.T) {
	root := copyVault(t)
	path := filepath.Join(root, "docs", "enemy-goals.md")
	other := filepath.Join(root, "docs", "later-goals.md")
	if err := os.WriteFile(other, []byte(readFile(t, path)), 0o644); err != nil {
		t.Fatal(err)
	}
	before := snapshot(t, root)

	exitCode, output := run(t, root, "supersede", "enemy-goals", "later-goals")

	if exitCode != cli.ExitCodeFailure {
		t.Fatalf("exit code = %d, output %q; expected failure", exitCode, output)
	}
	for _, mention := range []string{"enemy-goals.md", `"superseded"`, `"proposed", "confirmed"`} {
		if !strings.Contains(output, mention) {
			t.Errorf("output %q does not mention %q", output, mention)
		}
	}
	if after := snapshot(t, root); !equalSnapshots(before, after) {
		t.Errorf("notes changed:\n%s", snapshotDiff(before, after))
	}
}

// snapshot reads every note under root by path.
func snapshot(t *testing.T, root string) map[string]string {
	t.Helper()
	notes := map[string]string{}
	err := filepath.WalkDir(root, func(path string, entry os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if !entry.IsDir() && filepath.Ext(path) == ".md" {
			notes[path] = readFile(t, path)
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	return notes
}

func equalSnapshots(before, after map[string]string) bool {
	if len(before) != len(after) {
		return false
	}
	for path, text := range before {
		if after[path] != text {
			return false
		}
	}
	return true
}

func snapshotDiff(before, after map[string]string) string {
	var lines []string
	for path, text := range after {
		if before[path] != text {
			lines = append(lines, path+":\n"+text)
		}
	}
	return strings.Join(lines, "\n")
}
