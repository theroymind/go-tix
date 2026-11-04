package tix_test

import (
	"path/filepath"
	"strings"
	"testing"

	"github.com/theroymind/railyard-cli"
)

func TestSetOnADoc(t *testing.T) {
	tests := []struct {
		name             string
		arguments        []string
		expectedExitCode cli.ExitCode
		replace          [2]string
		mentions         []string
	}{
		{"A status in the type's list changes only the status line",
			[]string{"set", "turn-order-is-a-wait-counter", "status=accepted"}, cli.ExitCodeSuccess,
			[2]string{"status: proposed\n", "status: accepted\n"}, nil},
		{"A unique prefix resolves to the doc when no ticket matches",
			[]string{"set", "turn", "status=deferred"}, cli.ExitCodeSuccess,
			[2]string{"status: proposed\n", "status: deferred\n"}, nil},
		{"A status outside the type's list names the type's statuses",
			[]string{"set", "turn-order-is-a-wait-counter", "status=done"}, cli.ExitCodeFailure,
			[2]string{}, []string{`"done"`, `"proposed", "accepted", "deferred", "superseded"`}},
		{"Any other key is written as a property",
			[]string{"set", "turn-order-is-a-wait-counter", "decided_by=user"}, cli.ExitCodeSuccess,
			[2]string{"decided_by:\n", "decided_by: user\n"}, nil},
		{"A list replaces a doc's list property",
			[]string{"set", "turn-order-is-a-wait-counter", "tasks=[mana, golem-guards]"}, cli.ExitCodeSuccess,
			[2]string{"tasks: []\n", "tasks: [mana, golem-guards]\n"}, nil},
		{"A prefix of both a ticket and a doc is ambiguous",
			[]string{"set", "enemy", "decided_by=user"}, cli.ExitCodeFailure,
			[2]string{}, []string{`"enemy"`, "ambiguous", "enemy-ai"}},
		{"A reference matching neither a ticket nor a doc has no match",
			[]string{"set", "nothing", "status=accepted"}, cli.ExitCodeFailure,
			[2]string{}, []string{"no ticket matches", `"nothing"`}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			root := copyVault(t)
			path := filepath.Join(root, "docs", "decisions", "turn-order-is-a-wait-counter.md")
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

func TestSetOnAVault(t *testing.T) {
	tests := []struct {
		name             string
		arguments        []string
		expectedExitCode cli.ExitCode
		replace          [2]string
		mentions         []string
	}{
		{"A note status changes one line",
			[]string{"set", "golem-guards", "status=done"}, cli.ExitCodeSuccess,
			[2]string{"status: todo\n", "status: done\n"}, nil},
		{"A ticket status maps back to its note status",
			[]string{"set", "golem-guards", "status=completed"}, cli.ExitCodeSuccess,
			[2]string{"status: todo\n", "status: done\n"}, nil},
		{"A status the profile doesn't map names the allowed statuses",
			[]string{"set", "golem-guards", "status=nonsense"}, cli.ExitCodeFailure,
			[2]string{}, []string{`"nonsense"`, `"done", "todo"`}},
		{"Priority has no property in a vault yet",
			[]string{"set", "golem-guards", "priority=2"}, cli.ExitCodeFailure,
			[2]string{}, []string{"priority", "not supported"}},
		{"A property the profile maps to dependencies points at that field",
			[]string{"set", "golem-guards", "depends_on=[]"}, cli.ExitCodeFailure,
			[2]string{}, []string{"depends_on", "maps it to dependencies"}},
		{"Dependencies are a list, which set refuses, pointing at dep and undep",
			[]string{"set", "golem-guards", "dependencies=mana"}, cli.ExitCodeUsage,
			[2]string{}, []string{"dependencies is a list", "tix dep", "tix undep"}},
		{"A value written as a list is written as a flow list",
			[]string{"set", "golem-guards", "commits=[0123abcd, 4567ef01]"}, cli.ExitCodeSuccess,
			[2]string{"depends_on: [weighted-step-choice]\n", "depends_on: [weighted-step-choice]\ncommits: [0123abcd, 4567ef01]\n"}, nil},
		{"A list replaces a property's value",
			[]string{"set", "golem-guards", "scope=[a, b]"}, cli.ExitCodeSuccess,
			[2]string{"scope: small\n", "scope: [a, b]\n"}, nil},
		{"An empty list clears a property to an empty list",
			[]string{"set", "golem-guards", "scope=[]"}, cli.ExitCodeSuccess,
			[2]string{"scope: small\n", "scope: []\n"}, nil},
		{"A wikilink stays text",
			[]string{"set", "golem-guards", "scope=[[enemy-goals]]"}, cli.ExitCodeSuccess,
			[2]string{"scope: small\n", "scope: '[[enemy-goals]]'\n"}, nil},
		{"A list holding a list is a usage error",
			[]string{"set", "golem-guards", "scope=[a, [b]]"}, cli.ExitCodeUsage,
			[2]string{}, []string{"scope", "list"}},
		{"Any other key is written as a property",
			[]string{"set", "golem-guards", "scope=x"}, cli.ExitCodeSuccess,
			[2]string{"scope: small\n", "scope: x\n"}, nil},
		{"Several pairs are applied together",
			[]string{"set", "golem-guards", "scope=x", "status=done"}, cli.ExitCodeSuccess,
			[2]string{"status: todo\nphase: enemy-puzzle\nscope: small\n", "status: done\nphase: enemy-puzzle\nscope: x\n"}, nil},
		{"One failing pair writes none of them",
			[]string{"set", "golem-guards", "scope=x", "status=nonsense"}, cli.ExitCodeFailure,
			[2]string{}, nil},
		{"A value may hold an equals sign",
			[]string{"set", "golem-guards", "scope=a=b"}, cli.ExitCodeSuccess,
			[2]string{"scope: small\n", "scope: a=b\n"}, nil},
		{"A pair without an equals sign is a usage error",
			[]string{"set", "golem-guards", "noequals"}, cli.ExitCodeUsage,
			[2]string{}, []string{`"noequals"`, "usage:"}},
		{"A pair with an empty key is a usage error",
			[]string{"set", "golem-guards", "=x"}, cli.ExitCodeUsage,
			[2]string{}, []string{`"=x"`}},
		{"No pair is a usage error",
			[]string{"set", "golem-guards"}, cli.ExitCodeUsage,
			[2]string{}, nil},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			root := copyVault(t)
			path := filepath.Join(root, "docs", "tasks", "golem-guards.md")
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

func TestSetOnYAML(t *testing.T) {
	tests := []struct {
		name             string
		arguments        []string
		expectedExitCode cli.ExitCode
		replace          [2]string
		mentions         []string
	}{
		{"A valid status is written and updated_at changes",
			[]string{"set", yamlID, "status=in_progress"}, cli.ExitCodeSuccess,
			[2]string{"status: open\n", "status: in_progress\n"}, nil},
		{"A status that isn't a ticket status names the allowed statuses and changes nothing",
			[]string{"set", yamlID, "status=nonsense"}, cli.ExitCodeFailure,
			[2]string{}, []string{"change rejected", "status", "in_progress", "completed"}},
		{"A priority is written as a plain integer",
			[]string{"set", yamlID, "priority=2"}, cli.ExitCodeSuccess,
			[2]string{"priority: 1\n", "priority: 2\n"}, nil},
		{"A priority that isn't an integer is a usage error",
			[]string{"set", yamlID, "priority=high"}, cli.ExitCodeUsage,
			[2]string{}, []string{"priority", `"high"`}},
		{"The id can't be set, since the file is named by it",
			[]string{"set", yamlID, "id=bs-zzzz"}, cli.ExitCodeFailure,
			[2]string{}, []string{"not supported"}},
		{"Tags take a value that isn't a list as a usage error",
			[]string{"set", yamlID, "tags=x"}, cli.ExitCodeUsage,
			[2]string{}, []string{"tags is a list"}},
		{"Tags are replaced by a list, quoted like v1",
			[]string{"set", yamlID, `tags=[engine, "slice: 2"]`}, cli.ExitCodeSuccess,
			[2]string{"tags: [engine, slice1]\n", "tags: [engine, \"slice: 2\"]\n"}, nil},
		{"An empty list of tags removes the key, as v1 omits it",
			[]string{"set", yamlID, "tags=[]"}, cli.ExitCodeSuccess,
			[2]string{"tags: [engine, slice1]\n", ""}, nil},
		{"Dependencies are a list, which set refuses",
			[]string{"set", yamlID, "dependencies=bs-2m7q"}, cli.ExitCodeUsage,
			[2]string{}, []string{"dependencies is a list", "tix dep", "tix undep"}},
		{"Acceptance is a list, which set refuses",
			[]string{"set", yamlID, "acceptance=x"}, cli.ExitCodeUsage,
			[2]string{}, []string{"acceptance is a list"}},
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
