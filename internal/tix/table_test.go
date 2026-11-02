package tix

import (
	"bytes"
	"strings"
	"testing"

	"github.com/theroymind/railyard/internal/contracts/ticket"
)

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
