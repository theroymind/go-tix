package vaultstore_test

import (
	"errors"
	"io/fs"
	"maps"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/theroymind/go-tix/contracts/ticket"
	"github.com/theroymind/go-tix/internal/tix/backend"
	"github.com/theroymind/go-tix/internal/tix/backend/vaultstore"
	"github.com/theroymind/go-tix/internal/tix/vault"
)

const fixtureTemplate = "docs/templates/task.md"

// vaultFiles reads every file under root, by path relative to root.
func vaultFiles(t *testing.T, root string) map[string]string {
	t.Helper()
	files := map[string]string{}
	err := filepath.WalkDir(root, func(path string, entry fs.DirEntry, err error) error {
		if err != nil || entry.IsDir() {
			return err
		}
		text, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		relative, err := filepath.Rel(root, path)
		files[filepath.ToSlash(relative)] = string(text)
		return err
	})
	if err != nil {
		t.Fatal(err)
	}
	return files
}

// writeVault writes files, by path relative to a new temporary root, and opens the vault there.
func writeVault(t *testing.T, files map[string]string) (backend.Backend, string) {
	t.Helper()
	root := t.TempDir()
	for path, text := range files {
		path = filepath.Join(root, path)
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte(text), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	backend.Register(backend.KindVault, vaultstore.Open)
	store, err := backend.Open(root)
	if err != nil {
		t.Fatalf("Open() error = %v", err)
	}
	return store, root
}

// replaceAll applies each old, new pair to text once, failing when text lacks old.
func replaceAll(t *testing.T, text string, replacements [][2]string) string {
	t.Helper()
	for _, replacement := range replacements {
		if !strings.Contains(text, replacement[0]) {
			t.Fatalf("text lacks %q:\n%s", replacement[0], text)
		}
		text = strings.Replace(text, replacement[0], replacement[1], 1)
	}
	return text
}

func TestCreate(t *testing.T) {
	two := ticket.Priority(2)
	tests := []struct {
		name         string
		value        ticket.Ticket
		expectedSlug string
		// replacements turn the template into the expected note.
		replacements [][2]string
	}{
		{"The title, parent and a criterion fill the template",
			ticket.Ticket{Title: "Golem guards twice", Type: ticket.TypeTask, Status: ticket.StatusOpen, Parent: "enemy-puzzle",
				Acceptance: ticket.NewCriteria([]string{"x"})},
			"golem-guards-twice",
			[][2]string{{"phase:\n", "phase: enemy-puzzle\n"}, {"# Title\n", "# Golem guards twice\n"},
				{"- [ ] An observable, checkable outcome.\n", "- [ ] x\n"}}},
		{"Two criteria replace the placeholder",
			ticket.Ticket{Title: "Two", Type: ticket.TypeTask, Status: ticket.StatusOpen, Acceptance: ticket.NewCriteria([]string{"a", "b"})},
			"two",
			[][2]string{{"# Title\n", "# Two\n"}, {"- [ ] An observable, checkable outcome.\n", "- [ ] a\n- [ ] b\n"}}},
		{"No criteria keep the placeholder",
			ticket.Ticket{Title: "None", Type: ticket.TypeTask, Status: ticket.StatusOpen},
			"none",
			[][2]string{{"# Title\n", "# None\n"}}},
		{"Dependencies keep the template's flow style",
			ticket.Ticket{Title: "Deps", Type: ticket.TypeTask, Status: ticket.StatusOpen, Dependencies: []string{"goal-scoring", "golem-guards"}},
			"deps",
			[][2]string{{"depends_on: []\n", "depends_on: [goal-scoring, golem-guards]\n"}, {"# Title\n", "# Deps\n"}}},
		{"A description replaces the placeholder paragraph",
			ticket.Ticket{Title: "Described", Type: ticket.TypeTask, Status: ticket.StatusOpen, Description: "First line.\nSecond line."},
			"described",
			[][2]string{{"# Title\n", "# Described\n"}, {"What to build.\n", "First line.\nSecond line.\n"}}},
		{"A description's trailing newline adds no blank line",
			ticket.Ticket{Title: "Piped", Type: ticket.TypeTask, Status: ticket.StatusOpen, Description: "From a file.\n"},
			"piped",
			[][2]string{{"# Title\n", "# Piped\n"}, {"What to build.\n", "From a file.\n"}}},
		{"A ticket status maps back to its note status, and priority and tags are ignored",
			ticket.Ticket{Title: "Done already", Type: ticket.TypeTask, Status: ticket.StatusCompleted, Priority: &two, Tags: []string{"x"}},
			"done-already",
			[][2]string{{"status: todo\n", "status: done\n"}, {"# Title\n", "# Done already\n"}}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			store, root := copyFixture(t)
			before := vaultFiles(t, root)

			slug, err := store.Create(tt.value)

			if err != nil {
				t.Fatalf("Create() error = %v", err)
			}
			if slug != tt.expectedSlug {
				t.Errorf("Create() = %q, expected %q", slug, tt.expectedSlug)
			}
			after := vaultFiles(t, root)
			path := "docs/tasks/" + tt.expectedSlug + ".md"
			expected := replaceAll(t, before[fixtureTemplate], tt.replacements)
			if after[path] != expected {
				t.Errorf("%s =\n%s\nexpected\n%s", path, after[path], expected)
			}
			delete(after, path)
			if !maps.Equal(after, before) {
				t.Errorf("Create() changed files other than %s", path)
			}
			read, err := store.Read(slug)
			if err != nil {
				t.Fatalf("Read() error = %v", err)
			}
			if read.Title != tt.value.Title || read.Status != tt.value.Status {
				t.Errorf("Read() = %+v, expected title %q and status %q", read, tt.value.Title, tt.value.Status)
			}
		})
	}
}

func TestCreateFails(t *testing.T) {
	task := func(title string) ticket.Ticket {
		return ticket.Ticket{Title: title, Type: ticket.TypeTask, Status: ticket.StatusOpen}
	}
	draft := task("Drafted")
	draft.Status = ticket.StatusDraft
	epic := task("New phase")
	epic.Type = ticket.TypeEpic
	story := task("A story")
	story.Type = ticket.TypeStory
	wrapped := task("First\nsecond")
	closingHashes := task("Issue #")
	tests := []struct {
		name          string
		value         ticket.Ticket
		expectedError error
		mentions      []string
	}{
		{"A slug a ticket note already has names the note", task("Golem guards"),
			vaultstore.ErrSlugTaken, []string{`"golem-guards"`, filepath.Join("docs", "tasks", "golem-guards.md")}},
		{"A mapped type without a template names the type", epic,
			vaultstore.ErrNoTemplate, []string{`"epic"`}},
		{"An unmapped type names the type", story,
			vaultstore.ErrNoTemplate, []string{`"story"`}},
		{"A status the profile doesn't map names the allowed statuses", draft,
			vaultstore.ErrUnknownStatus, []string{`"draft"`, `"done", "todo"`}},
		{"A title with no letter or digit has no slug", task("!!! ???"),
			vaultstore.ErrEmptySlug, []string{`"!!! ???"`}},
		{"A title the H1 can't hold on one line", wrapped,
			vault.ErrMultilineText, nil},
		{"A title the H1 would read back differently", closingHashes,
			vaultstore.ErrNotKept, []string{ticket.FieldTitle}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			store, root := copyFixture(t)
			before := vaultFiles(t, root)

			_, err := store.Create(tt.value)

			if !errors.Is(err, tt.expectedError) {
				t.Fatalf("Create() error = %v, expected %v", err, tt.expectedError)
			}
			for _, mention := range tt.mentions {
				if !strings.Contains(err.Error(), mention) {
					t.Errorf("error %q does not mention %q", err, mention)
				}
			}
			if after := vaultFiles(t, root); !maps.Equal(after, before) {
				t.Errorf("Create() changed the vault")
			}
		})
	}
}

// smallProfile maps task notes, with a parent property only when fields says so, and names templates/task.md.
func smallProfile(fields string) string {
	return `prefix: v
backend: vault
vault:
  root: .
  tickets:
    task:
      ticket_type: task
      statuses: {todo: open}
` + fields + `  ticket_templates:
    task: templates/task.md
`
}

const smallNote = "---\ntype: task\nstatus: todo\n---\n\n# Existing\n"

func TestCreateInASmallVault(t *testing.T) {
	tests := []struct {
		name         string
		files        map[string]string
		value        ticket.Ticket
		expectedPath string
		expectedText string
	}{
		{"A template without an H1 gets one after a blank line",
			map[string]string{
				".tix/config.yml":   smallProfile(""),
				"templates/task.md": "---\ntype: task\nstatus: todo\n---\n",
				"tasks/existing.md": smallNote,
			},
			ticket.Ticket{Title: "New", Type: ticket.TypeTask, Status: ticket.StatusOpen},
			"tasks/new.md", "---\ntype: task\nstatus: todo\n---\n\n# New\n"},
		{"A description goes under an inserted H1",
			map[string]string{
				".tix/config.yml":   smallProfile(""),
				"templates/task.md": "---\ntype: task\nstatus: todo\n---\n\nPlaceholder.\n\n## Notes\n",
				"tasks/existing.md": smallNote,
			},
			ticket.Ticket{Title: "New", Type: ticket.TypeTask, Status: ticket.StatusOpen, Description: "Said."},
			"tasks/new.md", "---\ntype: task\nstatus: todo\n---\n\n# New\n\nSaid.\n\n## Notes\n"},
		{"A description is added under an H1 with no paragraph",
			map[string]string{
				".tix/config.yml":   smallProfile(""),
				"templates/task.md": "---\ntype: task\nstatus: todo\n---\n\n# Title\n## Notes\n",
				"tasks/existing.md": smallNote,
			},
			ticket.Ticket{Title: "New", Type: ticket.TypeTask, Status: ticket.StatusOpen, Description: "Said."},
			"tasks/new.md", "---\ntype: task\nstatus: todo\n---\n\n# New\n\nSaid.\n\n## Notes\n"},
		{"The first note of its type goes in the vault's root",
			map[string]string{
				".tix/config.yml":   smallProfile(""),
				"templates/task.md": "---\ntype: task\nstatus: todo\n---\n\n# Title\n",
			},
			ticket.Ticket{Title: "New", Type: ticket.TypeTask, Status: ticket.StatusOpen},
			"new.md", "---\ntype: task\nstatus: todo\n---\n\n# New\n"},
		{"The new note goes where most notes of its type are",
			map[string]string{
				".tix/config.yml":   smallProfile(""),
				"templates/task.md": "---\ntype: task\nstatus: todo\n---\n\n# Title\n",
				"a/one.md":          smallNote,
				"b/two.md":          smallNote,
				"b/three.md":        smallNote,
			},
			ticket.Ticket{Title: "New", Type: ticket.TypeTask, Status: ticket.StatusOpen},
			"b/new.md", "---\ntype: task\nstatus: todo\n---\n\n# New\n"},
		{"The first note of its type goes in the directory the profile names, which is created",
			map[string]string{
				".tix/config.yml":   smallProfile("") + "  ticket_directories:\n    task: work/tasks\n",
				"templates/task.md": "---\ntype: task\nstatus: todo\n---\n\n# Title\n",
			},
			ticket.Ticket{Title: "New", Type: ticket.TypeTask, Status: ticket.StatusOpen},
			"work/tasks/new.md", "---\ntype: task\nstatus: todo\n---\n\n# New\n"},
		{"The directory the profile names wins over where most notes of the type are",
			map[string]string{
				".tix/config.yml":   smallProfile("") + "  ticket_directories:\n    task: a\n",
				"templates/task.md": "---\ntype: task\nstatus: todo\n---\n\n# Title\n",
				"a/one.md":          smallNote,
				"b/two.md":          smallNote,
				"b/three.md":        smallNote,
			},
			ticket.Ticket{Title: "New", Type: ticket.TypeTask, Status: ticket.StatusOpen},
			"a/new.md", "---\ntype: task\nstatus: todo\n---\n\n# New\n"},
		{"A wrapped placeholder criterion is replaced whole, and a ticked one is unticked",
			map[string]string{
				".tix/config.yml":   smallProfile(""),
				"templates/task.md": "---\ntype: task\nstatus: todo\n---\n\n# Title\n\n## Acceptance criteria\n\n* [x] One\n  wrapped.\n* [ ] Two\n\n## End\n",
				"tasks/existing.md": smallNote,
			},
			ticket.Ticket{Title: "New", Type: ticket.TypeTask, Status: ticket.StatusOpen, Acceptance: ticket.NewCriteria([]string{"a", "b", "c"})},
			"tasks/new.md", "---\ntype: task\nstatus: todo\n---\n\n# New\n\n## Acceptance criteria\n\n* [ ] a\n* [ ] b\n* [ ] c\n\n## End\n"},
		{"Placeholders at the end of a note without a trailing newline are replaced",
			map[string]string{
				".tix/config.yml":   smallProfile(""),
				"templates/task.md": "---\ntype: task\nstatus: todo\n---\n# T\n\n## Acceptance criteria\n\n- [ ] a\n- [ ] b",
				"tasks/existing.md": smallNote,
			},
			ticket.Ticket{Title: "New", Type: ticket.TypeTask, Status: ticket.StatusOpen, Acceptance: ticket.NewCriteria([]string{"x"})},
			"tasks/new.md", "---\ntype: task\nstatus: todo\n---\n# New\n\n## Acceptance criteria\n\n- [ ] x"},
		{"A CRLF template without an H1 keeps CRLF in the H1 and description",
			map[string]string{
				".tix/config.yml":   smallProfile(""),
				"templates/task.md": "---\r\ntype: task\r\nstatus: todo\r\n---\r\n\r\nPlaceholder.\r\n",
				"tasks/existing.md": smallNote,
			},
			ticket.Ticket{Title: "New", Type: ticket.TypeTask, Status: ticket.StatusOpen, Description: "Said."},
			"tasks/new.md", "---\r\ntype: task\r\nstatus: todo\r\n---\r\n\r\n# New\r\n\r\nSaid.\r\n"},
		{"The created and updated timestamps are added to the frontmatter",
			map[string]string{
				".tix/config.yml":   smallProfile(""),
				"templates/task.md": "---\ntype: task\nstatus: todo\n---\n\n# Title\n",
				"tasks/existing.md": smallNote,
			},
			ticket.Ticket{Title: "New", Type: ticket.TypeTask, Status: ticket.StatusOpen,
				CreatedAt: "2026-10-25T09:00:00Z", UpdatedAt: "2026-10-25T09:00:00Z"},
			"tasks/new.md",
			"---\ntype: task\nstatus: todo\ncreated_at: 2026-10-25T09:00:00Z\nupdated_at: 2026-10-25T09:00:00Z\n---\n\n# New\n"},
		{"A timestamp the template holds is replaced",
			map[string]string{
				".tix/config.yml":   smallProfile(""),
				"templates/task.md": "---\ntype: task\ncreated_at:\nstatus: todo\n---\n\n# Title\n",
				"tasks/existing.md": smallNote,
			},
			ticket.Ticket{Title: "New", Type: ticket.TypeTask, Status: ticket.StatusOpen,
				CreatedAt: "2026-10-25T09:00:00Z", UpdatedAt: "2026-10-26T10:30:00Z"},
			"tasks/new.md",
			"---\ntype: task\ncreated_at: 2026-10-25T09:00:00Z\nstatus: todo\nupdated_at: 2026-10-26T10:30:00Z\n---\n\n# New\n"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			store, root := writeVault(t, tt.files)

			slug, err := store.Create(tt.value)
			if err != nil {
				t.Fatalf("Create() error = %v", err)
			}

			if text := vaultFiles(t, root)[tt.expectedPath]; text != tt.expectedText {
				t.Errorf("%s = %q, expected %q", tt.expectedPath, text, tt.expectedText)
			}
			listed, err := store.List()
			if err != nil {
				t.Fatalf("List() error = %v", err)
			}
			for _, found := range listed {
				if found.ID == slug && (found.CreatedAt != tt.value.CreatedAt || found.UpdatedAt != tt.value.UpdatedAt) {
					t.Errorf("List() has %s created %q and updated %q, expected %q and %q",
						slug, found.CreatedAt, found.UpdatedAt, tt.value.CreatedAt, tt.value.UpdatedAt)
				}
			}
		})
	}
}

func TestCreateFailsInASmallVault(t *testing.T) {
	tests := []struct {
		name          string
		files         map[string]string
		value         ticket.Ticket
		expectedError error
		mentions      []string
	}{
		{"A file that is not a ticket note already has the path",
			map[string]string{".tix/config.yml": smallProfile(""), "templates/task.md": smallNote,
				"tasks/existing.md": smallNote, "tasks/new.md": "Plain notes.\n"},
			ticket.Ticket{Title: "New", Type: ticket.TypeTask, Status: ticket.StatusOpen},
			vaultstore.ErrSlugTaken, []string{filepath.Join("tasks", "new.md")}},
		{"A parent with no property to hold it",
			map[string]string{".tix/config.yml": smallProfile(""), "templates/task.md": smallNote, "tasks/existing.md": smallNote},
			ticket.Ticket{Title: "New", Type: ticket.TypeTask, Status: ticket.StatusOpen, Parent: "existing"},
			backend.ErrUnsupported, []string{ticket.FieldParent}},
		{"Dependencies with no property to hold them",
			map[string]string{".tix/config.yml": smallProfile("      fields: {phase: parent}\n"), "templates/task.md": smallNote, "tasks/existing.md": smallNote},
			ticket.Ticket{Title: "New", Type: ticket.TypeTask, Status: ticket.StatusOpen, Dependencies: []string{"existing"}},
			backend.ErrUnsupported, []string{ticket.FieldDependencies}},
		{"Criteria with no heading to go under",
			map[string]string{".tix/config.yml": smallProfile(""), "templates/task.md": smallNote, "tasks/existing.md": smallNote},
			ticket.Ticket{Title: "New", Type: ticket.TypeTask, Status: ticket.StatusOpen, Acceptance: ticket.NewCriteria([]string{"a"})},
			vault.ErrSectionNotFound, []string{"Acceptance criteria"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			store, root := writeVault(t, tt.files)
			before := vaultFiles(t, root)

			_, err := store.Create(tt.value)

			if !errors.Is(err, tt.expectedError) {
				t.Fatalf("Create() error = %v, expected %v", err, tt.expectedError)
			}
			for _, mention := range tt.mentions {
				if !strings.Contains(err.Error(), mention) {
					t.Errorf("error %q does not mention %q", err, mention)
				}
			}
			if after := vaultFiles(t, root); !maps.Equal(after, before) {
				t.Errorf("Create() changed the vault")
			}
		})
	}
}
