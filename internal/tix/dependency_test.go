package tix_test

import (
	"path/filepath"
	"strings"
	"testing"

	"github.com/theroymind/railyard/internal/cli"
	"github.com/theroymind/railyard/internal/contracts/ticket"
)

func TestDependencyCommandsOnAVault(t *testing.T) {
	const (
		golemGuards = "docs/tasks/golem-guards.md"
		golemLinks  = "depends_on: [weighted-step-choice]\n"
	)
	tests := []struct {
		name             string
		arguments        []string
		note             string
		expectedExitCode cli.ExitCode
		replace          [2]string
		mentions         []string
	}{
		{"dep adds a dependency to the note's list",
			[]string{"dep", "golem-guards", "mana"}, golemGuards, cli.ExitCodeSuccess,
			[2]string{golemLinks, "depends_on: [weighted-step-choice, mana]\n"}, nil},
		{"dep resolves prefixes",
			[]string{"dep", "golem", "ma"}, golemGuards, cli.ExitCodeSuccess,
			[2]string{golemLinks, "depends_on: [weighted-step-choice, mana]\n"}, nil},
		{"dep adds several dependencies at once",
			[]string{"dep", "golem-guards", "mana", "harpy-hastes"}, golemGuards, cli.ExitCodeSuccess,
			[2]string{golemLinks, "depends_on: [weighted-step-choice, mana, harpy-hastes]\n"}, nil},
		{"dep on a dependency the ticket has changes nothing",
			[]string{"dep", "golem-guards", "weighted-step-choice"}, golemGuards, cli.ExitCodeSuccess,
			[2]string{}, nil},
		{"dep keeps a wikilink the note already holds",
			[]string{"dep", "weighted-step-choice", "sim-queries"}, "docs/tasks/weighted-step-choice.md", cli.ExitCodeSuccess,
			[2]string{`depends_on: ["[[goal-scoring]]"]` + "\n", `depends_on: ["[[goal-scoring]]", sim-queries]` + "\n"}, nil},
		{"dep refuses a cycle and names its path",
			[]string{"dep", "sim-queries", "golem-guards"}, "docs/tasks/sim-queries.md", cli.ExitCodeFailure,
			[2]string{}, []string{"cycle", "sim-queries -> golem-guards -> weighted-step-choice -> goal-scoring -> sim-queries"}},
		{"dep refuses a ticket depending on itself",
			[]string{"dep", "golem-guards", "golem"}, golemGuards, cli.ExitCodeFailure,
			[2]string{}, []string{"cycle", "golem-guards -> golem-guards"}},
		{"dep refuses a cycle through another dependency named in the same call",
			[]string{"dep", "sim-queries", "harpy-hastes", "mana"}, "docs/tasks/sim-queries.md", cli.ExitCodeFailure,
			[2]string{}, []string{"cycle", "sim-queries -> harpy-hastes -> weighted-step-choice -> goal-scoring -> sim-queries"}},
		{"dep on an unknown reference names it and writes nothing",
			[]string{"dep", "golem-guards", "mana", "nothing"}, golemGuards, cli.ExitCodeFailure,
			[2]string{}, []string{"no ticket matches", `"nothing"`}},
		{"dep on a doc is refused, since only tickets are dependencies",
			[]string{"dep", "golem-guards", "turn-order-is-a-wait-counter"}, golemGuards, cli.ExitCodeFailure,
			[2]string{}, []string{"no ticket matches", `"turn-order-is-a-wait-counter"`}},
		{"dep on a note type without a dependencies property is unsupported",
			[]string{"dep", "first-fight", "mana"}, "docs/phases/first-fight.md", cli.ExitCodeFailure,
			[2]string{}, []string{"dependencies", "not supported"}},
		{"dep without a dependency is a usage error",
			[]string{"dep", "golem-guards"}, golemGuards, cli.ExitCodeUsage,
			[2]string{}, []string{"usage:"}},
		{"undep removes the last dependency, leaving an empty list",
			[]string{"undep", "golem-guards", "weighted-step-choice"}, golemGuards, cli.ExitCodeSuccess,
			[2]string{golemLinks, "depends_on: []\n"}, nil},
		{"undep resolves a prefix",
			[]string{"undep", "golem-guards", "weighted"}, golemGuards, cli.ExitCodeSuccess,
			[2]string{golemLinks, "depends_on: []\n"}, nil},
		{"undep removes one of several and keeps the rest in order",
			[]string{"undep", "wraith-casts", "mana"}, "docs/tasks/wraith-casts.md", cli.ExitCodeSuccess,
			[2]string{"depends_on: [mana, enemy-behaviour-profiles]\n", "depends_on: [enemy-behaviour-profiles]\n"}, nil},
		{"undep removes a dependency on a ticket that doesn't exist",
			[]string{"undep", "mana", "sim-actions"}, "docs/tasks/mana.md", cli.ExitCodeSuccess,
			[2]string{"depends_on: [sim-actions]\n", "depends_on: []\n"}, nil},
		{"undep removes several at once",
			[]string{"undep", "wraith-casts", "mana", "enemy-behaviour-profiles"}, "docs/tasks/wraith-casts.md", cli.ExitCodeSuccess,
			[2]string{"depends_on: [mana, enemy-behaviour-profiles]\n", "depends_on: []\n"}, nil},
		{"undep of a ticket that isn't a dependency names both and writes nothing",
			[]string{"undep", "golem-guards", "weighted-step-choice", "mana"}, golemGuards, cli.ExitCodeFailure,
			[2]string{}, []string{"golem-guards", "does not depend on", `"mana"`}},
		{"undep without a dependency is a usage error",
			[]string{"undep", "golem-guards"}, golemGuards, cli.ExitCodeUsage,
			[2]string{}, []string{"usage:"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			root := copyVault(t)
			before := snapshot(t, root)
			path := filepath.Join(root, tt.note)
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
			after := snapshot(t, root)
			delete(before, path)
			delete(after, path)
			if !equalSnapshots(before, after) {
				t.Errorf("other files changed:\n%s", snapshotDiff(before, after))
			}
		})
	}
}

// addedTickets are the ids of two tickets added beside the YAML fixture: one depending on it, one standing alone.
type addedTickets struct {
	dependent string
	free      string
}

func TestDependencyCommandsOnYAML(t *testing.T) {
	const dependenciesLine = "dependencies: [bs-2m7q]\n"
	tests := []struct {
		name             string
		arguments        func(added addedTickets) []string
		expectedExitCode cli.ExitCode
		replace          func(added addedTickets) [2]string
		mentions         []string
	}{
		{"dep appends to the flow list",
			func(added addedTickets) []string { return []string{"dep", yamlID, added.free} }, cli.ExitCodeSuccess,
			func(added addedTickets) [2]string {
				return [2]string{dependenciesLine, "dependencies: [bs-2m7q, " + added.free + "]\n"}
			}, nil},
		{"undep of the last dependency removes the key, as v1 omits an empty list",
			func(addedTickets) []string { return []string{"undep", yamlID, "bs-2m7q"} }, cli.ExitCodeSuccess,
			func(addedTickets) [2]string { return [2]string{dependenciesLine, ""} }, nil},
		{"dep refuses a cycle and names its path",
			func(added addedTickets) []string { return []string{"dep", yamlID, added.dependent} }, cli.ExitCodeFailure,
			nil, []string{"cycle", yamlID + " -> "}},
		{"undep of a ticket that isn't a dependency fails",
			func(added addedTickets) []string { return []string{"undep", yamlID, added.free} }, cli.ExitCodeFailure,
			nil, []string{"does not depend on"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			root, path := yamlRepository(t)
			added := addedTickets{
				dependent: addYAMLTicket(t, root, "--dependency", yamlID, "Dependent"),
				free:      addYAMLTicket(t, root, "Free"),
			}
			addedPaths := []string{
				filepath.Join(root, ticket.DirectoryName, ticket.FileName(added.dependent)),
				filepath.Join(root, ticket.DirectoryName, ticket.FileName(added.free)),
			}
			original := readFile(t, path)
			addedOriginals := []string{readFile(t, addedPaths[0]), readFile(t, addedPaths[1])}

			exitCode, output := run(t, root, tt.arguments(added)...)

			if exitCode != tt.expectedExitCode {
				t.Fatalf("exit code = %d, expected %d; output %q", exitCode, tt.expectedExitCode, output)
			}
			for _, mention := range tt.mentions {
				if !strings.Contains(output, mention) {
					t.Errorf("output %q does not mention %q", output, mention)
				}
			}
			for index, addedPath := range addedPaths {
				if actual := readFile(t, addedPath); actual != addedOriginals[index] {
					t.Errorf("an added ticket changed to\n%s", actual)
				}
			}
			actual := readFile(t, path)
			if tt.replace == nil {
				if actual != original {
					t.Errorf("ticket =\n%s\nexpected it unchanged", actual)
				}
				return
			}
			replace := tt.replace(added)
			expected := strings.Replace(original, replace[0], replace[1], 1)
			if updatedAtLine.ReplaceAllString(actual, "") != updatedAtLine.ReplaceAllString(expected, "") {
				t.Errorf("ticket =\n%s\nexpected\n%s", actual, expected)
			}
			if updatedAtLine.FindString(actual) == updatedAtLine.FindString(original) {
				t.Errorf("updated_at unchanged: %s", updatedAtLine.FindString(actual))
			}
		})
	}
}

// addYAMLTicket adds a task with the given options and title and returns its id.
func addYAMLTicket(t *testing.T, root string, arguments ...string) string {
	t.Helper()
	exitCode, output := run(t, root, append([]string{"add", "--type", "task"}, arguments...)...)
	if exitCode != cli.ExitCodeSuccess {
		t.Fatalf("add exit code = %d, output %q", exitCode, output)
	}
	return strings.TrimSpace(output)
}
