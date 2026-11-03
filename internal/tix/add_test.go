package tix_test

import (
	"encoding/json"
	"io/fs"
	"maps"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/theroymind/railyard/internal/cli"
	"github.com/theroymind/railyard/internal/contracts/ticket"
)

// ticketFiles lists the file names in root's .tix directory.
func ticketFiles(t *testing.T, root string) []string {
	t.Helper()
	entries, err := os.ReadDir(filepath.Join(root, ticket.DirectoryName))
	if err != nil {
		t.Fatal(err)
	}
	var names []string
	for _, entry := range entries {
		names = append(names, entry.Name())
	}
	return names
}

// showJSON runs show --json on reference and decodes the ticket it prints.
func showJSON(t *testing.T, root, reference string) ticket.Ticket {
	t.Helper()
	exitCode, output := run(t, root, "show", "--json", reference)
	if exitCode != cli.ExitCodeSuccess {
		t.Fatalf("show exit code = %d, output %q", exitCode, output)
	}
	var shown ticket.Ticket
	if err := json.Unmarshal([]byte(output), &shown); err != nil {
		t.Fatalf("show --json printed %q: %v", output, err)
	}
	return shown
}

func TestAddOnYAML(t *testing.T) {
	tests := []struct {
		name                 string
		arguments            []string
		expectedTitle        string
		expectedParent       string
		expectedDependencies []string
	}{
		{"A wrapped title is stored on one line",
			[]string{"add", "--type", "task", "Golem guards\n   twice\tin a row "},
			"Golem guards twice in a row", "", nil},
		{"A title of exactly the maximum length is kept",
			[]string{"add", "--type", "task", strings.Repeat("a", ticket.TitleMaximumLength)},
			strings.Repeat("a", ticket.TitleMaximumLength), "", nil},
		{"Parent and dependencies are stored resolved",
			[]string{"add", "--type", "task", "--parent", "k3f", "--dependency", "k3", "Child"},
			"Child", yamlID, []string{yamlID}},
		{"A dependency named twice is stored once",
			[]string{"add", "--type", "task", "--dependency", "bs-k3f9", "--dependency", "k3", "Child"},
			"Child", "", []string{yamlID}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			root, _ := yamlRepository(t)

			exitCode, output := run(t, root, tt.arguments...)

			if exitCode != cli.ExitCodeSuccess {
				t.Fatalf("exit code = %d, output %q", exitCode, output)
			}
			id := strings.TrimSpace(output)
			text := readFile(t, filepath.Join(root, ticket.DirectoryName, ticket.FileName(id)))
			if tt.expectedParent != "" && !strings.Contains(text, "parent: "+tt.expectedParent+"\n") {
				t.Errorf("file does not hold parent %s:\n%s", tt.expectedParent, text)
			}
			shown := showJSON(t, root, id)
			if shown.Title != tt.expectedTitle {
				t.Errorf("title = %q, expected %q", shown.Title, tt.expectedTitle)
			}
			if shown.Parent != tt.expectedParent {
				t.Errorf("parent = %q, expected %q", shown.Parent, tt.expectedParent)
			}
			if !slices.Equal(shown.Dependencies, tt.expectedDependencies) {
				t.Errorf("dependencies = %q, expected %q", shown.Dependencies, tt.expectedDependencies)
			}
		})
	}
}

func TestAddOnYAMLFails(t *testing.T) {
	tests := []struct {
		name      string
		arguments []string
		mentions  []string
	}{
		{"A title over the maximum length",
			[]string{"add", "--type", "task", strings.Repeat("a", ticket.TitleMaximumLength+1)},
			[]string{"title is too long", "72"}},
		{"An unknown dependency",
			[]string{"add", "--type", "task", "--dependency", yamlID, "--dependency", "zzzz", "Child"},
			[]string{"no ticket matches", `"zzzz"`}},
		{"An unknown parent",
			[]string{"add", "--type", "task", "--parent", "zzzz", "Child"},
			[]string{"no ticket matches", `"zzzz"`}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			root, _ := yamlRepository(t)
			before := ticketFiles(t, root)

			exitCode, output := run(t, root, tt.arguments...)

			if exitCode != cli.ExitCodeFailure {
				t.Fatalf("exit code = %d, expected %d; output %q", exitCode, cli.ExitCodeFailure, output)
			}
			for _, mention := range tt.mentions {
				if !strings.Contains(output, mention) {
					t.Errorf("output %q does not mention %q", output, mention)
				}
			}
			if after := ticketFiles(t, root); !slices.Equal(after, before) {
				t.Errorf("files = %q, expected %q", after, before)
			}
		})
	}
}

func TestShowReadsTitlesThatAddWouldReject(t *testing.T) {
	tests := []struct {
		name  string
		title string
	}{
		{"A title over the maximum length", strings.Repeat("a", ticket.TitleMaximumLength+10)},
		{"A multi-line title", "first line\nsecond line"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			root, path := yamlRepository(t)
			title, err := json.Marshal(tt.title)
			if err != nil {
				t.Fatal(err)
			}
			text := strings.Replace(readFile(t, path), "title: Purchasables and greedy-cheapest policy\n", "title: "+string(title)+"\n", 1)
			if err := os.WriteFile(path, []byte(text), 0o644); err != nil {
				t.Fatal(err)
			}

			if shown := showJSON(t, root, yamlID); shown.Title != tt.title {
				t.Errorf("title = %q, expected %q", shown.Title, tt.title)
			}
		})
	}
}

// vaultSnapshot reads every file under root, by path relative to root.
func vaultSnapshot(t *testing.T, root string) map[string]string {
	t.Helper()
	files := map[string]string{}
	err := filepath.WalkDir(root, func(path string, entry fs.DirEntry, err error) error {
		if err != nil || entry.IsDir() {
			return err
		}
		relative, err := filepath.Rel(root, path)
		files[relative] = readFile(t, path)
		return err
	})
	if err != nil {
		t.Fatal(err)
	}
	return files
}

func TestAddOnAVault(t *testing.T) {
	const template = "docs/templates/task.md"
	tests := []struct {
		name                 string
		arguments            []string
		expectedSlug         string
		replacements         [][2]string
		expectedParent       string
		expectedDependencies []string
		expectedAcceptance   []ticket.Criterion
	}{
		{"The title, parent and a criterion fill the template",
			[]string{"add", "--type", "task", "--parent", "enemy-puzzle", "--acceptance", "x", "Golem guards twice"},
			"golem-guards-twice",
			[][2]string{{"phase:\n", "phase: enemy-puzzle\n"}, {"# Title\n", "# Golem guards twice\n"},
				{"- [ ] An observable, checkable outcome.\n", "- [ ] x\n"}},
			"enemy-puzzle", nil, []ticket.Criterion{{Text: "x"}}},
		{"Two criteria replace the placeholder",
			[]string{"add", "--type", "task", "--acceptance", "a", "--acceptance", "b", "Two"},
			"two",
			[][2]string{{"# Title\n", "# Two\n"}, {"- [ ] An observable, checkable outcome.\n", "- [ ] a\n- [ ] b\n"}},
			"", nil, []ticket.Criterion{{Text: "a"}, {Text: "b"}}},
		{"No criteria keep the placeholder",
			[]string{"add", "--type", "task", "None"},
			"none",
			[][2]string{{"# Title\n", "# None\n"}},
			"", nil, []ticket.Criterion{{Text: "An observable, checkable outcome."}}},
		{"Dependencies are resolved and written as plain slugs",
			[]string{"add", "--type", "task", "--dependency", "goal-scoring", "--dependency", "golem-g", "Deps"},
			"deps",
			[][2]string{{"depends_on: []\n", "depends_on: [goal-scoring, golem-guards]\n"}, {"# Title\n", "# Deps\n"}},
			"", []string{"goal-scoring", "golem-guards"}, []ticket.Criterion{{Text: "An observable, checkable outcome."}}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			root := copyVault(t)
			before := vaultSnapshot(t, root)

			exitCode, output := run(t, root, tt.arguments...)

			if exitCode != cli.ExitCodeSuccess {
				t.Fatalf("exit code = %d, output %q", exitCode, output)
			}
			if slug := strings.TrimSpace(output); slug != tt.expectedSlug {
				t.Errorf("printed %q, expected %q", slug, tt.expectedSlug)
			}
			path := filepath.Join("docs", "tasks", tt.expectedSlug+".md")
			expected := before[filepath.FromSlash(template)]
			for _, replacement := range tt.replacements {
				if !strings.Contains(expected, replacement[0]) {
					t.Fatalf("template lacks %q", replacement[0])
				}
				expected = strings.Replace(expected, replacement[0], replacement[1], 1)
			}
			if text := readFile(t, filepath.Join(root, path)); text != expected {
				t.Errorf("%s =\n%s\nexpected\n%s", path, text, expected)
			}
			shown := showJSON(t, root, tt.expectedSlug)
			if shown.Type != ticket.TypeTask || shown.Status != ticket.StatusOpen || shown.Parent != tt.expectedParent ||
				!slices.Equal(shown.Dependencies, tt.expectedDependencies) || !slices.Equal(shown.Acceptance, tt.expectedAcceptance) {
				t.Errorf("show = %+v, expected type task, status open, parent %q, dependencies %q and criteria %+v",
					shown, tt.expectedParent, tt.expectedDependencies, tt.expectedAcceptance)
			}
		})
	}
}

func TestAddOnAVaultFails(t *testing.T) {
	tests := []struct {
		name      string
		arguments []string
		mentions  []string
	}{
		{"A slug a note already has names the note",
			[]string{"add", "--type", "task", "Golem guards"},
			[]string{filepath.Join("docs", "tasks", "golem-guards.md")}},
		{"A type with no template names the type",
			[]string{"add", "--type", "epic", "New phase"},
			[]string{`"epic"`}},
		{"A draft names the allowed statuses",
			[]string{"add", "--type", "task", "--draft", "Drafted"},
			[]string{`"draft"`, `"done", "todo"`}},
		{"A title with no letter or digit",
			[]string{"add", "--type", "task", "!!!"},
			[]string{`"!!!"`}},
		{"An unknown parent",
			[]string{"add", "--type", "task", "--parent", "nowhere", "New"},
			[]string{"no ticket matches", `"nowhere"`}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			root := copyVault(t)
			before := vaultSnapshot(t, root)

			exitCode, output := run(t, root, tt.arguments...)

			if exitCode != cli.ExitCodeFailure {
				t.Fatalf("exit code = %d, expected %d; output %q", exitCode, cli.ExitCodeFailure, output)
			}
			for _, mention := range tt.mentions {
				if !strings.Contains(output, mention) {
					t.Errorf("output %q does not mention %q", output, mention)
				}
			}
			if after := vaultSnapshot(t, root); !maps.Equal(after, before) {
				t.Errorf("add changed the vault")
			}
		})
	}
}
