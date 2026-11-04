package backend_test

import (
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/theroymind/go-tix/contracts/ticket"
	"github.com/theroymind/go-tix/internal/tix/backend"
)

const vaultConfig = `prefix: odin
backend: vault
vault:
  root: docs
  type_property: type
  tickets:
    task:
      ticket_type: task
      statuses: {todo: open, done: completed}
      fields: {depends_on: dependencies, phase: parent}
    phase:
      ticket_type: epic
      statuses: {active: in_progress, done: completed}
  docs:
    decision:
      statuses: [proposed, accepted]
      template: templates/decision.md
  criteria_heading: Acceptance criteria
  ticket_templates:
    task: templates/task.md
`

func TestReadConfig(t *testing.T) {
	tests := []struct {
		name          string
		config        string
		expectedError error
		mentions      []string
	}{
		{"A tix v1 config loads", "prefix: ry\n", nil, nil},
		{"An empty config fails on its prefix", "", backend.ErrInvalidFile, []string{"prefix"}},
		{"A vault config with a full profile loads", vaultConfig, nil, nil},
		{"A YAML backend may carry a profile", "prefix: ry\nbackend: yaml\nvault:\n  tickets:\n    task: {ticket_type: task, statuses: {todo: open}}\n", nil, nil},
		{"An unknown top-level key is named, on the YAML backend", "prefix: ry\nprefx: ry\n", backend.ErrInvalidFile,
			[]string{"prefx", "line 2"}},
		{"An unknown top-level key is named, on the vault backend", vaultConfig + "colour: blue\n", backend.ErrInvalidFile,
			[]string{"colour"}},
		{"An unknown profile key is named", strings.Replace(vaultConfig, "  root: docs\n", "  rooot: docs\n", 1), backend.ErrInvalidFile,
			[]string{"rooot"}},
		{"An unknown key in a ticket mapping is named", strings.Replace(vaultConfig, "      ticket_type: epic\n", "      ticket_type: epic\n      parent_field: phase\n", 1),
			backend.ErrInvalidFile, []string{"parent_field"}},
		{"An unknown key in a doc mapping is named", strings.Replace(vaultConfig, "      template: templates/decision.md\n", "      templat: x.md\n", 1),
			backend.ErrInvalidFile, []string{"templat"}},
		{"An unknown profile key is named on the YAML backend too", "prefix: ry\nvault:\n  tickets: {}\n  typo: 1\n", backend.ErrInvalidFile,
			[]string{"typo"}},
		{"A duplicate key is rejected", "prefix: ry\nprefix: rz\n", backend.ErrDuplicateKey, []string{"prefix"}},
		{"A ticket type outside the contract names the allowed ones", strings.Replace(vaultConfig, "ticket_type: epic", "ticket_type: saga", 1),
			backend.ErrInvalidFile, []string{"vault.tickets.phase.ticket_type", `"epic"`}},
		{"A status outside the contract names the field", strings.Replace(vaultConfig, "active: in_progress", "active: busy", 1),
			backend.ErrInvalidFile, []string{"vault.tickets.phase.statuses.active", `"in_progress"`}},
		{"Two note statuses mapping to one ticket status are rejected", strings.Replace(vaultConfig, "{todo: open, done: completed}", "{todo: open, done: completed, shipped: completed}", 1),
			backend.ErrInvalidFile, []string{"vault.tickets.task.statuses", `"done", "shipped"`, `"completed"`}},
		{"A note status named after a different ticket status is rejected", strings.Replace(vaultConfig, "{active: in_progress, done: completed}", "{open: in_progress, done: completed}", 1),
			backend.ErrInvalidFile, []string{"vault.tickets.phase.statuses", `"open"`, `"in_progress"`}},
		{"A note status named after the ticket status it maps to loads", strings.Replace(vaultConfig, "{active: in_progress, done: completed}", "{in_progress: in_progress, done: completed}", 1),
			nil, nil},
		{"A mapping with no statuses is rejected", strings.Replace(vaultConfig, "      statuses: {active: in_progress, done: completed}\n", "", 1),
			backend.ErrInvalidFile, []string{"vault.tickets.phase.statuses"}},
		{"A field outside the link fields names them", strings.Replace(vaultConfig, "phase: parent", "phase: title", 1),
			backend.ErrInvalidFile, []string{"vault.tickets.task.fields.phase", `"parent"`}},
		{"A type both ticket and doc is rejected", strings.Replace(vaultConfig, "    decision:\n", "    task:\n", 1),
			backend.ErrInvalidFile, []string{"vault.docs.task"}},
		{"A template for a non-ticket type is rejected", strings.Replace(vaultConfig, "    task: templates/task.md\n", "    spec: templates/spec.md\n", 1),
			backend.ErrInvalidFile, []string{"vault.ticket_templates.spec"}},
		{"An embeddings endpoint and model load", "prefix: ry\nembeddings:\n  endpoint: http://localhost:11434/v1\n  model: all-minilm\n", nil, nil},
		{"An embeddings endpoint that isn't an absolute URL is named", "prefix: ry\nembeddings:\n  endpoint: localhost:11434\n  model: all-minilm\n",
			backend.ErrInvalidFile, []string{"embeddings.endpoint", "absolute URL"}},
		{"An embeddings model is required", "prefix: ry\nembeddings:\n  endpoint: http://localhost:11434/v1\n",
			backend.ErrInvalidFile, []string{"embeddings.model"}},
		{"An unknown embeddings key is named", "prefix: ry\nembeddings:\n  endpoint: http://localhost:11434/v1\n  model: m\n  key: x\n",
			backend.ErrInvalidFile, []string{"key"}},
		{"Required sections for a ticket and a doc type load", strings.Replace(vaultConfig, "  criteria_heading:", "  required_sections:\n    task: [Acceptance criteria, Verification]\n    decision: [Rejected]\n  criteria_heading:", 1),
			nil, nil},
		{"Required sections for an unknown type are named", strings.Replace(vaultConfig, "  criteria_heading:", "  required_sections:\n    spec: [Shape]\n  criteria_heading:", 1),
			backend.ErrInvalidFile, []string{"vault.required_sections.spec"}},
		{"An empty required heading is named", strings.Replace(vaultConfig, "  criteria_heading:", "  required_sections:\n    task: [Verification, \" \"]\n  criteria_heading:", 1),
			backend.ErrInvalidFile, []string{"vault.required_sections.task.1"}},
		{"A judge command and model load", "prefix: ry\njudge:\n  command: claude\n  model: opus\n", nil, nil},
		{"A blank judge model is named", "prefix: ry\njudge:\n  model: \" \"\n", backend.ErrInvalidFile, []string{"judge.model"}},
		{"An unknown judge key is named", "prefix: ry\njudge:\n  modle: opus\n", backend.ErrInvalidFile, []string{"modle"}},
		{"A profile with no ticket types is rejected", "prefix: ry\nbackend: vault\nvault:\n  root: docs\n", backend.ErrInvalidFile,
			[]string{"vault.tickets"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tixDirectory := filepath.Join(t.TempDir(), ticket.DirectoryName)
			if err := os.Mkdir(tixDirectory, backend.DirectoryPermission); err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(backend.ConfigPath(tixDirectory), []byte(tt.config), backend.FilePermission); err != nil {
				t.Fatal(err)
			}

			_, err := backend.ReadConfig(tixDirectory)

			if !errors.Is(err, tt.expectedError) {
				t.Fatalf("ReadConfig() error = %v, want %v", err, tt.expectedError)
			}
			for _, mention := range tt.mentions {
				if !strings.Contains(err.Error(), mention) {
					t.Errorf("ReadConfig() error = %q, want it to mention %s", err, mention)
				}
			}
		})
	}
}

func TestReadConfigOfThisRepository(t *testing.T) {
	config, err := backend.ReadConfig(filepath.Join("..", "..", "..", ticket.DirectoryName))
	if err != nil {
		t.Fatalf("ReadConfig() error = %v", err)
	}
	if config.BackendKind() != backend.KindYAML || config.Vault != nil {
		t.Errorf("ReadConfig() = %+v, want the YAML backend with no profile", config)
	}
}

func TestProfileWithDefaults(t *testing.T) {
	tests := []struct {
		name     string
		profile  backend.Profile
		expected backend.Profile
	}{
		{"Empty values take the defaults", backend.Profile{},
			backend.Profile{Root: backend.DefaultVaultRoot, TypeProperty: backend.DefaultTypeProperty, CriteriaHeading: backend.DefaultCriteriaHeading,
				SummaryProperty: backend.DefaultSummaryProperty, CommitsProperty: backend.DefaultCommitsProperty}},
		{"Set values are kept, and the files heading stays unset unless named",
			backend.Profile{Root: "vault", TypeProperty: "kind", CriteriaHeading: "Done when", SummaryProperty: "gist", CommitsProperty: "hashes"},
			backend.Profile{Root: "vault", TypeProperty: "kind", CriteriaHeading: "Done when", SummaryProperty: "gist", CommitsProperty: "hashes"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.profile.WithDefaults()
			if !reflect.DeepEqual(got, tt.expected) {
				t.Errorf("WithDefaults() = %+v, want %+v", got, tt.expected)
			}
		})
	}
}
