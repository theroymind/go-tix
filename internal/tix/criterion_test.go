package tix_test

import (
	"path/filepath"
	"strings"
	"testing"

	"github.com/theroymind/railyard-cli"
)

func TestCriterionCommandsOnYAML(t *testing.T) {
	const (
		first  = "  - Policy buys the cheapest affordable purchasable each tick\n"
		second = "  - \"Quoted: because of the colon\"\n"
	)
	tests := []struct {
		name             string
		arguments        []string
		expectedExitCode cli.ExitCode
		replace          [2]string
		mentions         []string
	}{
		{"tick without a number is a usage error",
			[]string{"tick", yamlID}, cli.ExitCodeUsage,
			[2]string{}, []string{"criterion number", "usage:"}},
		{"tick without a reference is a usage error",
			[]string{"tick"}, cli.ExitCodeUsage,
			[2]string{}, []string{"usage:"}},
		{"tick with a non-integer names it",
			[]string{"tick", yamlID, "abc"}, cli.ExitCodeUsage,
			[2]string{}, []string{`"abc"`}},
		{"tick with zero names it",
			[]string{"tick", yamlID, "0"}, cli.ExitCodeUsage,
			[2]string{}, []string{`"0"`}},
		{"untick with a non-integer names it",
			[]string{"untick", yamlID, "1", "x2"}, cli.ExitCodeUsage,
			[2]string{}, []string{`"x2"`}},
		{"tick on YAML is not supported",
			[]string{"tick", yamlID, "1"}, cli.ExitCodeFailure,
			[2]string{}, []string{"not supported"}},
		{"untick on YAML is not supported",
			[]string{"untick", yamlID, "1"}, cli.ExitCodeFailure,
			[2]string{}, []string{"not supported"}},
		{"ac set replaces only that criterion",
			[]string{"ac", "set", yamlID, "2", "Plain now"}, cli.ExitCodeSuccess,
			[2]string{second, "  - Plain now\n"}, nil},
		{"ac add appends one criterion",
			[]string{"ac", "add", yamlID, "Third: quoted"}, cli.ExitCodeSuccess,
			[2]string{second, second + "  - \"Third: quoted\"\n"}, nil},
		{"ac set past the last criterion names the number",
			[]string{"ac", "set", yamlID, "9", "x"}, cli.ExitCodeFailure,
			[2]string{}, []string{"no criterion 9"}},
		{"ac set with a non-integer names it",
			[]string{"ac", "set", yamlID, "two", "x"}, cli.ExitCodeUsage,
			[2]string{}, []string{`"two"`}},
		{"ac set without text is a usage error",
			[]string{"ac", "set", yamlID, "2"}, cli.ExitCodeUsage,
			[2]string{}, []string{"ac set REF N TEXT"}},
		{"ac add with two texts is a usage error",
			[]string{"ac", "add", yamlID, "a", "b"}, cli.ExitCodeUsage,
			[2]string{}, []string{"ac add REF TEXT"}},
		{"ac with an unknown subcommand is a usage error",
			[]string{"ac", "remove", yamlID, "1"}, cli.ExitCodeUsage,
			[2]string{}, []string{`"remove"`}},
		{"ac alone is a usage error",
			[]string{"ac"}, cli.ExitCodeUsage,
			[2]string{}, []string{"usage:"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			root, path := yamlRepository(t)
			original := readFile(t, path)
			if !strings.Contains(original, first) || !strings.Contains(original, second) {
				t.Fatal("fixture criteria changed")
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

func TestCriterionCommandsOnAVault(t *testing.T) {
	const (
		second = "- [ ] The argmax takes the higher score"
		fourth = "- [ ] A parity test runs the determinism test's scripted fight and asserts every `last_action` matches the pre-change choices, and the roster test still asserts Attack in every profile.\n"
	)
	tests := []struct {
		name             string
		arguments        []string
		expectedExitCode cli.ExitCode
		replace          [2]string
		mentions         []string
	}{
		{"tick ticks only that box",
			[]string{"tick", "weighted-step-choice", "2"}, cli.ExitCodeSuccess,
			[2]string{second, "- [x] The argmax takes the higher score"}, nil},
		{"tick past the last criterion names the number and the count",
			[]string{"tick", "weighted-step-choice", "9"}, cli.ExitCodeFailure,
			[2]string{}, []string{"criterion 9", "has 4"}},
		{"ac set replaces only that criterion's text and keeps its box",
			[]string{"ac", "set", "weighted-step-choice", "4", "Parity holds."}, cli.ExitCodeSuccess,
			[2]string{fourth, "- [ ] Parity holds.\n"}, nil},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			root := copyVault(t)
			path := filepath.Join(root, "docs", "tasks", "weighted-step-choice.md")
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

func TestAcLsOnAVault(t *testing.T) {
	tests := []struct {
		name             string
		arguments        []string
		expectedExitCode cli.ExitCode
		expectedPrefixes []string
		mentions         []string
	}{
		{"ac ls numbers every criterion with its box",
			[]string{"ac", "ls", "weighted-step-choice"}, cli.ExitCodeSuccess,
			[]string{"1. [x] `scoring_target`", "2. [ ] The argmax", "3. [ ] Only the winning step", "4. [ ] A parity test"}, nil},
		{"ac ls --open keeps the numbers tick takes",
			[]string{"ac", "ls", "--open", "weighted-step-choice"}, cli.ExitCodeSuccess,
			[]string{"2. [ ] The argmax", "3. [ ] Only the winning step", "4. [ ] A parity test"}, nil},
		{"acceptance-criteria is ac's long name",
			[]string{"acceptance-criteria", "ls", "--open", "weighted-step-choice"}, cli.ExitCodeSuccess,
			[]string{"2. [ ] The argmax", "3. [ ] Only the winning step", "4. [ ] A parity test"}, nil},
		{"ac ls --open on a finished task prints nothing",
			[]string{"ac", "ls", "--open", "golem-guards"}, cli.ExitCodeSuccess, nil, nil},
		{"ac ls without a reference is a usage error",
			[]string{"ac", "ls"}, cli.ExitCodeUsage, nil, []string{"expected ac ls REF"}},
		{"--open goes only with ac ls",
			[]string{"ac", "add", "--open", "weighted-step-choice", "x"}, cli.ExitCodeUsage, nil, []string{"--open goes only with ac ls"}},
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
			if tt.expectedExitCode != cli.ExitCodeSuccess {
				return
			}
			lines := strings.Split(strings.TrimSuffix(output, "\n"), "\n")
			if output == "" {
				lines = nil
			}
			if len(lines) != len(tt.expectedPrefixes) {
				t.Fatalf("lines = %q, expected %d", lines, len(tt.expectedPrefixes))
			}
			for index, prefix := range tt.expectedPrefixes {
				if !strings.HasPrefix(lines[index], prefix) {
					t.Errorf("line %d = %q, expected it to start %q", index+1, lines[index], prefix)
				}
			}
		})
	}
}

func TestMarkAllOnAVault(t *testing.T) {
	tests := []struct {
		name             string
		arguments        []string
		expectedExitCode cli.ExitCode
		expectedOpen     string
	}{
		{"tick --all ticks every box", []string{"tick", "weighted-step-choice", "--all"}, cli.ExitCodeSuccess, ""},
		{"untick --all unticks every box", []string{"untick", "--all", "weighted-step-choice"}, cli.ExitCodeSuccess, "1234"},
		{"Numbers with --all is a usage error", []string{"tick", "weighted-step-choice", "2", "--all"}, cli.ExitCodeUsage, "234"},
		{"--all without a reference is a usage error", []string{"tick", "--all"}, cli.ExitCodeUsage, "234"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			root := copyVault(t)

			exitCode, output := run(t, root, tt.arguments...)

			if exitCode != tt.expectedExitCode {
				t.Fatalf("exit code = %d, expected %d; output %q", exitCode, tt.expectedExitCode, output)
			}
			_, listed := run(t, root, "ac", "ls", "--open", "weighted-step-choice")
			open := ""
			for line := range strings.Lines(listed) {
				open += line[:1]
			}
			if open != tt.expectedOpen {
				t.Errorf("unticked criteria = %q, expected %q", open, tt.expectedOpen)
			}
		})
	}
}
