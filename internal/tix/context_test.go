package tix_test

import (
	"os"
	"path/filepath"
	"reflect"
	"strconv"
	"strings"
	"testing"

	"github.com/theroymind/railyard/internal/cli"
	"github.com/theroymind/railyard/internal/contracts/card"
	"github.com/theroymind/railyard/internal/contracts/strictjson"
	"github.com/theroymind/railyard/internal/tix"
)

const contextConfig = `prefix: ctx
backend: vault
vault:
  tickets:
    task:
      ticket_type: task
      statuses:
        todo: open
      fields:
        depends_on: dependencies
        phase: parent
    phase:
      ticket_type: epic
      statuses:
        todo: open
  docs:
    spec:
      statuses: [confirmed]
    idea:
      statuses: [open]
  files_heading: Files likely touched
`

// contextNotes is a vault where start depends on dependency, which depends on far, which depends on farther; start
// sits in phase, which links back to it; start links to spec, which links to dependency; and idea links to spec.
var contextNotes = map[string]string{
	"tasks/start.md": "---\ntype: task\nstatus: todo\nphase: phase\ndepends_on: [dependency]\n" +
		"summary: Starts the walk.\ncommits: [2222222222, 1111111111]\n---\n\n# Start\n\nShape in [[spec]].\n\n" +
		"## Files likely touched\n\na.go, `b.go`\n",
	"tasks/dependency.md": "---\ntype: task\nstatus: todo\ndepends_on: [far]\n---\n\n# Dependency\n\n" +
		"## Files likely touched\n\n- `c.go`\n- d.go\n",
	"tasks/far.md":     "---\ntype: task\nstatus: todo\ndepends_on: [farther]\nsummary: Far away.\n---\n\n# Far\n",
	"tasks/farther.md": "---\ntype: task\nstatus: todo\nsummary: Farther still.\n---\n\n# Farther\n",
	"phases/phase.md":  "---\ntype: phase\nstatus: todo\nsummary: Holds the start.\n---\n\n# Phase\n\nTasks: [[start]].\n",
	"spec.md":          "---\ntype: spec\nstatus: confirmed\nsummary: Specifies it.\n---\n\n# Spec\n\nBuilt by [[dependency]] and [[nowhere]].\n",
	"ideas/idea.md":    "---\ntype: idea\nstatus: open\nsummary: An idea.\n---\n\n# Idea\n\nSee [[spec]].\n",
	"unmapped.md":      "# Unmapped\n\nLinks [[start]] without a type.\n",
}

func contextVault(t *testing.T) string {
	t.Helper()
	return contextVaultWith(t, "", nil)
}

// contextVaultWith writes the context vault with more profile lines and more notes.
func contextVaultWith(t *testing.T, profileLines string, notes map[string]string) string {
	t.Helper()
	root := t.TempDir()
	files := map[string]string{filepath.Join(".tix", "config.yml"): contextConfig + profileLines}
	for path, text := range contextNotes {
		files[filepath.Join("docs", path)] = text
	}
	for path, text := range notes {
		files[filepath.Join("docs", path)] = text
	}
	for path, text := range files {
		full := filepath.Join(root, path)
		if err := os.MkdirAll(filepath.Dir(full), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(full, []byte(text), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	return root
}

type reachedCard struct {
	id       string
	distance int
}

func TestContextWalk(t *testing.T) {
	atDefaultDepth := []reachedCard{
		{"start", 0}, {"dependency", 1}, {"phase", 1}, {"spec", 1}, {"far", 2}, {"idea", 2},
	}
	tests := []struct {
		name      string
		arguments []string
		expected  []reachedCard
	}{
		{"The default depth stops two hops out, a link back to the start and a second path to dependency changing nothing",
			[]string{"start"}, atDefaultDepth},
		{"A depth of one keeps the start's own neighbours", []string{"start", "--depth", "1"}, atDefaultDepth[:4]},
		{"A depth of zero is the start alone", []string{"start", "--depth", "0"}, atDefaultDepth[:1]},
		{"One hop more reaches farther", []string{"start", "--depth", strconv.Itoa(tix.DefaultContextDepth + 1)},
			append(append([]reachedCard{}, atDefaultDepth...), reachedCard{"farther", 3})},
		{"A doc starts a walk too, and reaches back along its backlinks", []string{"spec", "--depth", "1"},
			[]reachedCard{{"spec", 0}, {"dependency", 1}, {"idea", 1}, {"start", 1}}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			root := contextVault(t)

			cards := runContextJSON(t, root, tt.arguments...)

			var reached []reachedCard
			for _, value := range cards {
				reached = append(reached, reachedCard{value.ID, value.Distance})
			}
			if !reflect.DeepEqual(reached, tt.expected) {
				t.Errorf("reached %v, expected %v", reached, tt.expected)
			}
		})
	}
}

// hubNotes add a home note that links to start and to lonely, an idea nothing else links to.
var hubNotes = map[string]string{
	"home.md":   "---\ntype: home\n---\n\n# Home\n\nStart at [[start]]; also [[lonely]].\n",
	"lonely.md": "---\ntype: idea\nstatus: open\nsummary: Linked only from home.\n---\n\n# Lonely\n",
}

func TestContextStopTypes(t *testing.T) {
	const homeDoc = "    home:\n      statuses: []\n"
	tests := []struct {
		name         string
		profileLines string
		arguments    []string
		expected     []reachedCard
	}{
		{"Without stop types the hub walks on to lonely", "", []string{"start", "--depth", "2"},
			[]reachedCard{{"start", 0}, {"dependency", 1}, {"home", 1}, {"phase", 1}, {"spec", 1}, {"far", 2}, {"idea", 2}, {"lonely", 2}}},
		{"A hub at distance 1 gets its card but blocks its neighbours at 2", "  context_stop_types: [home]\n",
			[]string{"start", "--depth", "2"},
			[]reachedCard{{"start", 0}, {"dependency", 1}, {"home", 1}, {"phase", 1}, {"spec", 1}, {"far", 2}, {"idea", 2}}},
		{"A walk that starts at the hub still expands", "  context_stop_types: [home]\n", []string{"home", "--depth", "1"},
			[]reachedCard{{"home", 0}, {"lonely", 1}, {"start", 1}}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := strings.Replace(contextConfig, "  docs:\n", "  docs:\n"+homeDoc, 1)
			root := contextVaultWith(t, "", hubNotes)
			writeConfig(t, root, config+tt.profileLines)

			cards := runContextJSON(t, root, tt.arguments...)

			var reached []reachedCard
			for _, value := range cards {
				reached = append(reached, reachedCard{value.ID, value.Distance})
			}
			if !reflect.DeepEqual(reached, tt.expected) {
				t.Errorf("reached %v, expected %v", reached, tt.expected)
			}
		})
	}
}

func TestContextStopTypeMustBeANoteType(t *testing.T) {
	root := contextVaultWith(t, "  context_stop_types: [hub]\n", nil)

	exitCode, output := run(t, root, "context", "start")

	if exitCode != cli.ExitCodeFailure || !strings.Contains(output, "context_stop_types.0") {
		t.Errorf("exit code = %d, output %q; expected a failure naming context_stop_types.0", exitCode, output)
	}
}

func writeConfig(t *testing.T, root, config string) {
	t.Helper()
	if err := os.WriteFile(filepath.Join(root, ".tix", "config.yml"), []byte(config), 0o644); err != nil {
		t.Fatal(err)
	}
}

func TestContextCards(t *testing.T) {
	root := contextVault(t)

	cards := runContextJSON(t, root, "start", "--depth", "1")

	expected := []card.Card{
		{ID: "start", Type: "task", Title: "Start", Summary: "Starts the walk.", HasSummary: true, Distance: 0,
			Commits: []string{"2222222222", "1111111111"}, Files: []string{"a.go", "b.go"}},
		{ID: "dependency", Type: "task", Title: "Dependency", Distance: 1, Commits: []string{}, Files: []string{"c.go", "d.go"}},
		{ID: "phase", Type: "phase", Title: "Phase", Summary: "Holds the start.", HasSummary: true, Distance: 1,
			Commits: []string{}, Files: []string{}},
		{ID: "spec", Type: "spec", Title: "Spec", Summary: "Specifies it.", HasSummary: true, Distance: 1,
			Commits: []string{}, Files: []string{}},
	}
	if !reflect.DeepEqual(cards, expected) {
		t.Errorf("cards =\n%+v\nexpected\n%+v", cards, expected)
	}
}

func TestContextTable(t *testing.T) {
	root := contextVault(t)

	exitCode, output := run(t, root, "context", "start", "--depth", "1")

	if exitCode != cli.ExitCodeSuccess {
		t.Fatalf("exit code = %d, output %q", exitCode, output)
	}
	lines := strings.Split(strings.TrimSpace(output), "\n")
	expectedRows := [][]string{
		{"DISTANCE", "ID", "TYPE", "COMMITS", "SUMMARY"},
		{"0", "start", "task", "2222222,1111111", "Starts the walk."},
		{"1", "dependency", "task", "-", "(no summary)"},
		{"1", "phase", "phase", "-", "Holds the start."},
		{"1", "spec", "spec", "-", "Specifies it."},
	}
	if len(lines) != len(expectedRows) {
		t.Fatalf("output has %d lines, expected %d:\n%s", len(lines), len(expectedRows), output)
	}
	for index, row := range expectedRows {
		for _, cell := range row {
			if !strings.Contains(lines[index], cell) {
				t.Errorf("line %q does not hold %q", lines[index], cell)
			}
		}
	}
}

func TestContextOnTheYAMLBackend(t *testing.T) {
	root, _ := yamlRepository(t)
	middle := addYAMLTicket(t, root, "--dependency", yamlID, "Middle")
	start := addYAMLTicket(t, root, "--dependency", middle, "Start")
	dependent := addYAMLTicket(t, root, "--dependency", start, "Dependent")

	cards := runContextJSON(t, root, start)

	var reached []reachedCard
	for _, value := range cards {
		reached = append(reached, reachedCard{value.ID, value.Distance})
		if value.HasSummary || len(value.Commits) > 0 || len(value.Files) > 0 {
			t.Errorf("card %+v, expected a yaml ticket to carry no summary, commits or files", value)
		}
	}
	expected := []reachedCard{{start, 0}, {middle, 1}, {yamlID, 2}}
	if !reflect.DeepEqual(reached, expected) {
		t.Errorf("reached %v, expected %v, with %s, which depends on the start, left out", reached, expected, dependent)
	}
}

func TestContextRefusals(t *testing.T) {
	tests := []struct {
		name             string
		arguments        []string
		expectedExitCode cli.ExitCode
		mentions         []string
	}{
		{"No reference is a usage error", []string{"context"}, cli.ExitCodeUsage, []string{"expected exactly one reference"}},
		{"A negative depth is a usage error", []string{"context", "start", "--depth=-1"}, cli.ExitCodeUsage,
			[]string{`"-1"`, "is not a depth"}},
		{"A depth that isn't a number is a usage error", []string{"context", "start", "--depth", "two"}, cli.ExitCodeUsage,
			[]string{`"two"`}},
		{"An unknown reference fails", []string{"context", "nowhere"}, cli.ExitCodeFailure, []string{`"nowhere"`}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			root := contextVault(t)

			exitCode, output := run(t, root, tt.arguments...)

			if exitCode != tt.expectedExitCode {
				t.Fatalf("exit code = %d, expected %d; output %q", exitCode, tt.expectedExitCode, output)
			}
			for _, mention := range tt.mentions {
				if !strings.Contains(output, mention) {
					t.Errorf("output %q does not mention %q", output, mention)
				}
			}
		})
	}
}

// runContextJSON runs tix context --json with the arguments and decodes its cards strictly, each one valid.
func runContextJSON(t *testing.T, root string, arguments ...string) []card.Card {
	t.Helper()
	exitCode, output := run(t, root, append([]string{"context", "--json"}, arguments...)...)
	if exitCode != cli.ExitCodeSuccess {
		t.Fatalf("exit code = %d, output %q", exitCode, output)
	}
	var cards []card.Card
	if err := strictjson.Unmarshal([]byte(output), &cards); err != nil {
		t.Fatalf("decoding %s: %v", output, err)
	}
	for _, value := range cards {
		if err := value.Validate(); err != nil {
			t.Errorf("card %+v: %v", value, err)
		}
	}
	return cards
}
