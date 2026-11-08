package tix_test

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"

	"github.com/theroymind/go-tix/contracts/ticket"
	"github.com/theroymind/railyard-cli"
)

func TestShowNumbersCriteria(t *testing.T) {
	tests := []struct {
		name      string
		isVault   bool
		reference string
		block     string
	}{
		{"A YAML ticket's criteria are numbered without boxes after the file", false, yamlID,
			"\ncriteria:\n" +
				"  1. Policy buys the cheapest affordable purchasable each tick\n" +
				"  2. Quoted: because of the colon\n"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var root, path string
			if tt.isVault {
				root = copyVault(t)
				path = filepath.Join(root, "docs", "tasks", tt.reference+".md")
			} else {
				root, path = yamlRepository(t)
			}

			exitCode, output := run(t, root, "show", tt.reference)

			if exitCode != cli.ExitCodeSuccess {
				t.Fatalf("exit code = %d, output %q", exitCode, output)
			}
			if expected := readFile(t, path) + tt.block; output != expected {
				t.Errorf("output =\n%s\nexpected\n%s", output, expected)
			}
		})
	}
}

func TestShowNumbersANotesCheckboxesInPlace(t *testing.T) {
	root := copyVault(t)
	text := readFile(t, filepath.Join(root, "docs", "tasks", "weighted-step-choice.md"))

	exitCode, output := run(t, root, "show", "weighted-step-choice")

	if exitCode != cli.ExitCodeSuccess {
		t.Fatalf("exit code = %d, output %q", exitCode, output)
	}
	expected := text
	for index, text := range []string{"- [x] `scoring_target`", "- [ ] The argmax", "- [ ] Only the winning", "- [ ] A parity test"} {
		expected = strings.Replace(expected, text, strconv.Itoa(index+1)+". "+text[2:], 1)
	}
	if output != expected {
		t.Errorf("output =\n%s\nexpected\n%s", output, expected)
	}
	if strings.Contains(output, "criteria:") {
		t.Errorf("output repeats the criteria after the note:\n%s", output)
	}
}

func TestShowJSONIsUnchanged(t *testing.T) {
	root := copyVault(t)

	exitCode, output := run(t, root, "show", "--json", "weighted-step-choice")

	if exitCode != cli.ExitCodeSuccess {
		t.Fatalf("exit code = %d, output %q", exitCode, output)
	}
	var shown ticket.Ticket
	if err := json.Unmarshal([]byte(output), &shown); err != nil {
		t.Fatalf("output is not one JSON ticket: %v\n%s", err, output)
	}
	if len(shown.Acceptance) != 4 {
		t.Errorf("acceptance = %d criteria, expected 4", len(shown.Acceptance))
	}
}

func TestShowTextWithoutATrailingNewline(t *testing.T) {
	const withoutCriteria = "id: bs-k3f9\ntitle: No criteria\ntype: task\nstatus: open\npriority: 1\n" +
		"created_at: 2026-10-14T21:00:00-04:00\nupdated_at: 2026-10-14T21:00:00-04:00"
	tests := []struct {
		name     string
		text     string
		expected string
	}{
		{"Without criteria the text is printed byte for byte", withoutCriteria, withoutCriteria},
		{"With criteria the last line is ended before the blank line",
			withoutCriteria + "\nacceptance:\n  - One",
			withoutCriteria + "\nacceptance:\n  - One\n\ncriteria:\n  1. One\n"},
		{"Without criteria a file ending in a newline is printed byte for byte",
			withoutCriteria + "\n", withoutCriteria + "\n"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			root, path := yamlRepository(t)
			if err := os.WriteFile(path, []byte(tt.text), 0o644); err != nil {
				t.Fatal(err)
			}

			exitCode, output := run(t, root, "show", yamlID)

			if exitCode != cli.ExitCodeSuccess {
				t.Fatalf("exit code = %d, output %q", exitCode, output)
			}
			if output != tt.expected {
				t.Errorf("output = %q, expected %q", output, tt.expected)
			}
		})
	}
}

func TestShowFallsThroughToDocs(t *testing.T) {
	tests := []struct {
		name             string
		arguments        []string
		expectedExitCode cli.ExitCode
		note             string
		mentions         []string
	}{
		{"A doc id shows the note as doc show does",
			[]string{"show", "enemy-ai"}, cli.ExitCodeSuccess, filepath.Join("docs", "ideas", "enemy-ai.md"), nil},
		{"A doc at the docs root shows too",
			[]string{"show", "enemy-goals"}, cli.ExitCodeSuccess, filepath.Join("docs", "enemy-goals.md"), nil},
		{"A unique prefix of a doc alone resolves to the doc",
			[]string{"show", "turn"}, cli.ExitCodeSuccess, filepath.Join("docs", "decisions", "turn-order-is-a-wait-counter.md"), nil},
		{"JSON shows the doc",
			[]string{"show", "--json", "enemy-ai"}, cli.ExitCodeSuccess, "",
			[]string{`"id": "enemy-ai"`, `"type": "idea"`, `"title": "Enemy AI: targeting, behaviour and planning"`}},
		{"A ticket still wins over a doc, with its criteria numbered",
			[]string{"show", "golem-guards"}, cli.ExitCodeSuccess, "", []string{"# Golem guards", "1. [x] On a constructed world"}},
		{"A prefix of both a ticket and a doc is ambiguous",
			[]string{"show", "enemy"}, cli.ExitCodeFailure, "", []string{`"enemy"`, "ambiguous", "enemy-ai"}},
		{"A reference matching neither a ticket nor a doc has no match",
			[]string{"show", "nothing"}, cli.ExitCodeFailure, "", []string{"no ticket matches", `"nothing"`}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			root := copyVault(t)

			exitCode, output := run(t, root, tt.arguments...)

			if exitCode != tt.expectedExitCode {
				t.Fatalf("exit code = %d, expected %d; output %q", exitCode, tt.expectedExitCode, output)
			}
			if tt.note != "" {
				if expected := readFile(t, filepath.Join(root, tt.note)); !strings.HasPrefix(output, expected) {
					t.Errorf("output =\n%s\nexpected to start with the note\n%s", output, expected)
				}
			}
			for _, mention := range tt.mentions {
				if !strings.Contains(output, mention) {
					t.Errorf("output %q does not mention %q", output, mention)
				}
			}
		})
	}
}

func TestShowANoteWithoutCriteria(t *testing.T) {
	root := copyVault(t)
	text := readFile(t, filepath.Join(root, "docs", "phases", "enemy-puzzle.md"))

	exitCode, output := run(t, root, "show", "enemy-puzzle")

	if exitCode != cli.ExitCodeSuccess {
		t.Fatalf("exit code = %d, output %q", exitCode, output)
	}
	if output != text {
		t.Errorf("output =\n%s\nexpected the phase note as it is", output)
	}
}

func TestShowWithParent(t *testing.T) {
	tests := []struct {
		name          string
		arguments     []string
		expectedOrder []string
		absent        []string
	}{
		{"The ticket comes first, then its parent, each under a ==> ID <== line",
			[]string{"show", "weighted-step-choice", "--with-parent"},
			[]string{"==> weighted-step-choice <==\n---\n", "# Weighted step choice", "\n\n==> enemy-puzzle <==\n---\n", "# Enemy puzzle"}, nil},
		{"A ticket without a parent shows alone, as plain show does",
			[]string{"show", "--with-parent", "enemy-puzzle"},
			[]string{"---\ntype: phase", "# Enemy puzzle"}, []string{"==>"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			exitCode, output := run(t, copyVault(t), tt.arguments...)

			if exitCode != cli.ExitCodeSuccess {
				t.Fatalf("exit code = %d, output %q", exitCode, output)
			}
			rest := output
			for _, part := range tt.expectedOrder {
				index := strings.Index(rest, part)
				if index < 0 {
					t.Fatalf("output %q lacks %q after the parts before it", output, part)
				}
				rest = rest[index+len(part):]
			}
			for _, absent := range tt.absent {
				if strings.Contains(output, absent) {
					t.Errorf("output %q mentions %q", output, absent)
				}
			}
		})
	}
}

func TestShowWithParentAsJSON(t *testing.T) {
	exitCode, output := run(t, copyVault(t), "show", "--with-parent", "--json", "weighted-step-choice")
	if exitCode != cli.ExitCodeSuccess {
		t.Fatalf("exit code = %d, output %q", exitCode, output)
	}
	var shown []ticket.Ticket
	if err := json.Unmarshal([]byte(output), &shown); err != nil {
		t.Fatalf("output %q is not a JSON array of tickets: %v", output, err)
	}
	if len(shown) != 2 || shown[0].ID != "weighted-step-choice" || shown[1].ID != "enemy-puzzle" {
		t.Errorf("tickets = %+v, expected weighted-step-choice then enemy-puzzle", shown)
	}
}
