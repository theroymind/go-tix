package tix

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"

	"github.com/theroymind/go-tix/contracts/ticket"
)

func TestWriteTicketsJSONOmitsAcceptance(t *testing.T) {
	tests := []struct {
		name       string
		acceptance []ticket.Criterion
	}{
		{"A ticket with criteria lists without them", []ticket.Criterion{{Text: "One"}, {Text: "Two", Done: true}}},
		{"A ticket without criteria is unchanged", nil},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var output bytes.Buffer
			row := ticket.Ticket{ID: "t-0001", Type: ticket.TypeTask, Status: ticket.StatusOpen, Title: "Title", Acceptance: tt.acceptance}

			if err := writeTickets(&output, []ticket.Ticket{row}, true); err != nil {
				t.Fatalf("writeTickets() error = %v", err)
			}
			var rows []ticket.Ticket
			if err := json.Unmarshal(output.Bytes(), &rows); err != nil {
				t.Fatalf("output is not a JSON array: %v\n%s", err, output.String())
			}
			if len(rows) != 1 || rows[0].ID != row.ID || rows[0].Acceptance != nil {
				t.Errorf("rows = %+v, want one row for %s with no acceptance", rows, row.ID)
			}
			if len(row.Acceptance) != len(tt.acceptance) {
				t.Errorf("the input ticket lost its criteria")
			}
		})
	}
}

func TestWriteTicketsPriorityColumn(t *testing.T) {
	tests := []struct {
		name     string
		priority *ticket.Priority
		expected string
	}{
		{"A present priority is its number", new(ticket.PriorityHighest), "0"},
		{"An absent priority is a dash", nil, tableAbsent},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var output bytes.Buffer
			row := ticket.Ticket{ID: "t-0001", Type: ticket.TypeTask, Status: ticket.StatusOpen, Priority: tt.priority, Title: "Title"}

			if err := writeTickets(&output, []ticket.Ticket{row}, false); err != nil {
				t.Fatalf("writeTickets() error = %v", err)
			}
			lines := strings.Split(strings.TrimSpace(output.String()), "\n")
			fields := strings.Fields(lines[len(lines)-1])
			if len(fields) != 5 || fields[3] != tt.expected {
				t.Errorf("row = %q, want priority %q", lines[len(lines)-1], tt.expected)
			}
		})
	}
}
