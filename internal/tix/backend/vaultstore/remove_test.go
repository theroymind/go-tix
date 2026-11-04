package vaultstore_test

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/theroymind/go-tix/contracts/ticket"
	"github.com/theroymind/go-tix/internal/tix/backend"
	"github.com/theroymind/go-tix/internal/tix/vault"
)

func TestRemoveField(t *testing.T) {
	const (
		task     = "docs/tasks/golem-guards.md"
		decision = "docs/decisions/turn-order-is-a-wait-counter.md"
	)
	descriptionLine := "description: Old summary words.\n"
	withDescription := [2]string{"status: todo\n", "status: todo\n" + descriptionLine}
	docWithDescription := [2]string{"status: proposed\n", "status: proposed\n" + descriptionLine}
	tests := []struct {
		name          string
		note          string
		setup         [2]string
		field         string
		replace       [2]string
		expectedError error
		mentions      []string
	}{
		{"A ticket's property is removed", task, [2]string{}, "scope",
			[2]string{"scope: small\n", ""}, nil, nil},
		{"A doc's description property is removed and its body kept", decision, docWithDescription, ticket.FieldDescription,
			[2]string{descriptionLine, ""}, nil, nil},
		{"A doc's other property is removed", decision, [2]string{}, "tasks",
			[2]string{"tasks: []\n", ""}, nil, nil},
		{"A ticket's description is its body, so it isn't removed", task, withDescription, ticket.FieldDescription,
			[2]string{}, backend.ErrUnsupported, []string{ticket.FieldDescription, "body"}},
		{"A ticket's status is required", task, [2]string{}, ticket.FieldStatus,
			[2]string{}, backend.ErrRequiredField, []string{ticket.FieldStatus}},
		{"A ticket's type property is required", task, [2]string{}, "type",
			[2]string{}, backend.ErrRequiredField, []string{"type"}},
		{"A doc's status is required", decision, [2]string{}, ticket.FieldStatus,
			[2]string{}, backend.ErrRequiredField, []string{ticket.FieldStatus}},
		{"A doc's type property is required", decision, [2]string{}, "type",
			[2]string{}, backend.ErrRequiredField, []string{"type"}},
		{"A ticket's title is its heading, so it isn't removed", task, [2]string{}, ticket.FieldTitle,
			[2]string{}, backend.ErrUnsupported, []string{ticket.FieldTitle}},
		{"A property the profile maps to a link field names that field", task, [2]string{}, "depends_on",
			[2]string{}, backend.ErrUnsupported, []string{"depends_on", "maps it to " + ticket.FieldDependencies}},
		{"A key the ticket note doesn't have is named", task, [2]string{}, "owner",
			[2]string{}, backend.ErrNoField, []string{"owner", "golem-guards.md"}},
		{"A key the doc doesn't have is named", decision, [2]string{}, "owner",
			[2]string{}, backend.ErrNoField, []string{"owner"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			store, root := copyFixture(t)
			path := filepath.Join(root, tt.note)
			fixtureText, err := os.ReadFile(path)
			if err != nil {
				t.Fatal(err)
			}
			original := string(fixtureText)
			if tt.setup[0] != "" {
				if !strings.Contains(original, tt.setup[0]) {
					t.Fatalf("fixture lacks %q", tt.setup[0])
				}
				original = strings.Replace(original, tt.setup[0], tt.setup[1], 1)
				write(t, path, original)
			}
			slug := vault.SlugOf(path)
			changes := []backend.Change{backend.RemoveField{Field: tt.field}}

			if strings.Contains(tt.note, "decisions") {
				err = store.(backend.DocStore).ApplyDoc(slug, changes)
			} else {
				err = store.Apply(slug, changes)
			}

			if !errors.Is(err, tt.expectedError) {
				t.Fatalf("error = %v, want %v", err, tt.expectedError)
			}
			for _, mention := range tt.mentions {
				if !strings.Contains(err.Error(), mention) {
					t.Errorf("error = %q, want it to mention %s", err, mention)
				}
			}
			expected := original
			if tt.expectedError == nil {
				if !strings.Contains(expected, tt.replace[0]) {
					t.Fatalf("fixture lacks %q", tt.replace[0])
				}
				expected = strings.Replace(expected, tt.replace[0], tt.replace[1], 1)
			}
			got, err := os.ReadFile(path)
			if err != nil {
				t.Fatal(err)
			}
			if string(got) != expected {
				t.Errorf("note =\n%s\nwant\n%s", got, expected)
			}
		})
	}
}
