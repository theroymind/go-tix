package yamlstore

import (
	"bytes"
	"errors"
	"os"
	"reflect"
	"regexp"
	"slices"
	"strings"
	"testing"

	"github.com/theroymind/railyard/internal/contracts/ticket"
	"github.com/theroymind/railyard/internal/contracts/validation"
	"github.com/theroymind/railyard/internal/tix/backend"
)

const (
	firstCriterion  = "  - Policy buys the cheapest affordable purchasable each tick\n"
	secondCriterion = "  - \"Quoted: because of the colon\"\n"
)

func TestApplyEditsCriteria(t *testing.T) {
	fixture := string(readFixture(t))
	withoutCriteria := regexp.MustCompile(`(?m)^acceptance:\n(  - .*\n)+`).ReplaceAllString(fixture, "")
	withEmptyCriteria := regexp.MustCompile(`(?m)^acceptance:\n(  - .*\n)+`).ReplaceAllString(fixture, "acceptance: []\n")

	tests := []struct {
		name     string
		text     string
		change   backend.Change
		expected string
	}{
		{"SetCriterion replaces one criterion's text", fixture,
			backend.SetCriterion{Index: 1, Text: "Plain text"},
			strings.Replace(fixture, firstCriterion, "  - Plain text\n", 1)},
		{"SetCriterion quotes text like v1", fixture,
			backend.SetCriterion{Index: 1, Text: "Hello: world"},
			strings.Replace(fixture, firstCriterion, "  - \"Hello: world\"\n", 1)},
		{"SetCriterion drops quotes text no longer needs", fixture,
			backend.SetCriterion{Index: 2, Text: "Unquoted"},
			strings.Replace(fixture, secondCriterion, "  - Unquoted\n", 1)},
		{"AddCriterion appends after the last criterion", fixture,
			backend.AddCriterion{Text: "Third: quoted"},
			strings.Replace(fixture, secondCriterion, secondCriterion+"  - \"Third: quoted\"\n", 1)},
		{"AddCriterion adds the key to a ticket without criteria", withoutCriteria,
			backend.AddCriterion{Text: "First"},
			withoutCriteria + "acceptance:\n  - First\n"},
		{"AddCriterion turns an empty flow list into a block list", withEmptyCriteria,
			backend.AddCriterion{Text: "First"},
			strings.Replace(withEmptyCriteria, "acceptance: []\n", "acceptance:\n  - First\n", 1)},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			store := openTestStore(t)
			path := store.TicketPath(parseFixture(t).ID)
			writeFile(t, path, []byte(tt.text))

			if err := store.Apply(parseFixture(t).ID, []backend.Change{tt.change}); err != nil {
				t.Fatalf("Apply() error = %v", err)
			}

			if got := string(readFile(t, path)); got != tt.expected {
				t.Errorf("Apply() wrote\n%s\nwant\n%s", got, tt.expected)
			}
		})
	}
}

func TestApplyRefusesAChangeItCantMakeAndWritesNothing(t *testing.T) {
	setStatus := backend.SetField{Field: ticket.FieldStatus, Value: ticket.StatusCompleted}

	tests := []struct {
		name     string
		changes  []backend.Change
		expected error
	}{
		{"CheckCriterion is unsupported", []backend.Change{backend.CheckCriterion{Index: 1}}, backend.ErrUnsupported},
		{"UncheckCriterion is unsupported", []backend.Change{backend.UncheckCriterion{Index: 1}}, backend.ErrUnsupported},
		{"An unsupported change stops the changes before it", []backend.Change{setStatus, backend.CheckCriterion{Index: 1}}, backend.ErrUnsupported},
		{"SetCriterion past the last criterion", []backend.Change{backend.SetCriterion{Index: 3, Text: "x"}}, backend.ErrNoCriterion},
		{"SetCriterion below 1", []backend.Change{backend.SetCriterion{Index: 0, Text: "x"}}, backend.ErrNoCriterion},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			store := openTestStore(t)
			fixture := readFixture(t)
			id := parseFixture(t).ID
			path := store.TicketPath(id)
			writeFile(t, path, fixture)

			err := store.Apply(id, tt.changes)

			if !errors.Is(err, tt.expected) {
				t.Errorf("Apply() error = %v, want %v", err, tt.expected)
			}
			if got := readFile(t, path); !bytes.Equal(got, fixture) {
				t.Errorf("file changed to\n%s", got)
			}
		})
	}
}

func TestCreateAssignsAnIDAndWritesTheTicket(t *testing.T) {
	store := openTestStore(t)
	value := parseFixture(t)

	id, err := store.Create(value)
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	if !strings.HasPrefix(id, testPrefix+ticket.IDSeparator) || id == value.ID {
		t.Errorf("Create() id = %q, want a new id with prefix %s", id, testPrefix)
	}
	value.ID = id
	if got, err := store.Read(id); err != nil || !reflect.DeepEqual(got, value) {
		t.Errorf("Read(%q) = %+v, %v, want %+v", id, got, err, value)
	}
}

func TestCreateRequiresTheFieldsTheContractLeavesOptional(t *testing.T) {
	tests := []struct {
		field  string
		remove func(value *ticket.Ticket)
	}{
		{ticket.FieldPriority, func(value *ticket.Ticket) { value.Priority = nil }},
		{ticket.FieldCreatedAt, func(value *ticket.Ticket) { value.CreatedAt = "" }},
		{ticket.FieldUpdatedAt, func(value *ticket.Ticket) { value.UpdatedAt = "" }},
	}

	for _, tt := range tests {
		t.Run(tt.field, func(t *testing.T) {
			store := openTestStore(t)
			value := parseFixture(t)
			tt.remove(&value)

			_, err := store.Create(value)

			var problems validation.Errors
			if !errors.Is(err, backend.ErrInvalidFile) || !errors.As(err, &problems) || !slices.Equal(problems.Fields(), []string{tt.field}) {
				t.Errorf("Create() error = %v, want %v naming %s", err, backend.ErrInvalidFile, tt.field)
			}
			if entries, _ := os.ReadDir(store.Directory); len(entries) != 0 {
				t.Errorf("Create() wrote %v", entries)
			}
		})
	}
}
