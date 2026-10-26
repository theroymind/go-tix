package ticket

import (
	"bytes"
	"encoding/json"
	"errors"
	"slices"
	"strconv"
	"testing"

	"github.com/theroymind/railyard/internal/contracts/validation"
)

const unknownField = "estimate"

func validTicket() Ticket {
	return Ticket{
		ID:         "ry" + IDSeparator + IDCharacters[len(IDCharacters)-IDSuffixLength:],
		Title:      "Port ls and ready",
		Type:       TypeTask,
		Status:     StatusOpen,
		Priority:   PriorityDefault,
		Tags:       []string{"tix"},
		Acceptance: []string{"ls lists open tickets"},
		CreatedAt:  "2026-10-25T09:00:00-04:00",
		UpdatedAt:  "2026-10-25T09:00:00-04:00",
	}
}

func failingFields(ticket Ticket) []string {
	var problems validation.Errors
	if errors.As(ticket.Validate(), &problems) {
		return problems.Fields()
	}
	return nil
}

func TestValidateAccepts(t *testing.T) {
	withUnknownField := validTicket()
	withUnknownField.UnknownFields = map[string]any{unknownField: "2d"}

	tests := []struct {
		name   string
		ticket Ticket
	}{
		{"A valid ticket", validTicket()},
		{"A ticket with a field it doesn't know about", withUnknownField},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := tt.ticket.Validate(); err != nil {
				t.Errorf("Validate() = %v, want nil", err)
			}
		})
	}
}

func TestValidateRejects(t *testing.T) {
	tests := []struct {
		field  string
		change func(*Ticket)
	}{
		{FieldType, func(ticket *Ticket) { ticket.Type = "bug" }},
		{FieldStatus, func(ticket *Ticket) { ticket.Status = "done" }},
		{FieldPriority, func(ticket *Ticket) { ticket.Priority = PriorityLowest + 1 }},
		{FieldPriority, func(ticket *Ticket) { ticket.Priority = PriorityHighest - 1 }},
		{FieldTitle, func(ticket *Ticket) { ticket.Title = "" }},
		{FieldTitle, func(ticket *Ticket) { ticket.Title = "   " }},
		{FieldID, func(ticket *Ticket) { ticket.ID = "ry-iiii" }},
		{FieldID, func(ticket *Ticket) { ticket.ID = "ry" + IDSeparator + IDCharacters[:IDSuffixLength+1] }},
		{FieldParent, func(ticket *Ticket) { ticket.Parent = "not-an-id" }},
		{FieldDependencies + ".1", func(ticket *Ticket) { ticket.Dependencies = []string{ticket.ID, "not-an-id"} }},
		{FieldCreatedAt, func(ticket *Ticket) { ticket.CreatedAt = "2026-10-25T09:00:00" }},
	}

	for _, tt := range tests {
		t.Run(tt.field, func(t *testing.T) {
			ticket := validTicket()
			tt.change(&ticket)

			err := ticket.Validate()

			if !errors.Is(err, validation.ErrInvalid) {
				t.Fatalf("Validate() = %v, want %v", err, validation.ErrInvalid)
			}
			if got := failingFields(ticket); !slices.Equal(got, []string{tt.field}) {
				t.Errorf("failing fields = %v, want [%s]", got, tt.field)
			}
		})
	}
}

func TestMarshalJSONKeepsUnknownFieldsAfterKnownOnes(t *testing.T) {
	ticket := validTicket()
	ticket.Title = "Fix <a> & <b>"
	ticket.UnknownFields = map[string]any{unknownField: "2d"}

	encoded, err := json.Marshal(ticket)
	if err != nil {
		t.Fatalf("json.Marshal() error = %v", err)
	}

	var decoded map[string]any
	if err := json.Unmarshal(encoded, &decoded); err != nil {
		t.Fatalf("json.Unmarshal(%s) error = %v", encoded, err)
	}
	if decoded[unknownField] != ticket.UnknownFields[unknownField] {
		t.Errorf("%s = %v, want %v", unknownField, decoded[unknownField], ticket.UnknownFields[unknownField])
	}
	if decoded[FieldTitle] != ticket.Title {
		t.Errorf("%s = %v, want %v", FieldTitle, decoded[FieldTitle], ticket.Title)
	}
	lastKnown := bytes.Index(encoded, []byte(strconv.Quote(FieldUpdatedAt)))
	firstUnknown := bytes.Index(encoded, []byte(strconv.Quote(unknownField)))
	if firstUnknown < lastKnown {
		t.Errorf("unknown field written before known fields: %s", encoded)
	}
}
