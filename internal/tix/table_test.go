package tix

import (
	"bytes"
	"encoding/json"
	"slices"
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

			if err := writeTickets(&output, []ticket.Ticket{row}, propertyColumns{}, true); err != nil {
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

func TestWriteTicketsColumns(t *testing.T) {
	scope := propertyColumns{names: []string{"scope"}, values: map[string][]string{"t-0001": {"small"}}}
	tests := []struct {
		name       string
		priorities []*ticket.Priority
		properties propertyColumns
		header     []string
		rows       [][]string
	}{
		{
			"No priority anywhere drops the column",
			[]*ticket.Priority{nil, nil},
			propertyColumns{},
			[]string{tableColumnID, tableColumnType, tableColumnStatus, tableColumnTitle},
			[][]string{{"t-0001", "task", "open", "Title"}, {"t-0002", "task", "open", "Title"}},
		},
		{
			"One priority shows the column, with a dash where it is absent",
			[]*ticket.Priority{new(ticket.PriorityHighest), nil},
			propertyColumns{},
			[]string{tableColumnID, tableColumnType, tableColumnStatus, tableColumnPriority, tableColumnTitle},
			[][]string{{"t-0001", "task", "open", "0", "Title"}, {"t-0002", "task", "open", tableAbsent, "Title"}},
		},
		{
			"A property column sits before the title, with a dash for a ticket without the property",
			[]*ticket.Priority{nil, nil},
			scope,
			[]string{tableColumnID, tableColumnType, tableColumnStatus, "SCOPE", tableColumnTitle},
			[][]string{{"t-0001", "task", "open", "small", "Title"}, {"t-0002", "task", "open", tableAbsent, "Title"}},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var output bytes.Buffer
			tickets := []ticket.Ticket{
				{ID: "t-0001", Type: ticket.TypeTask, Status: ticket.StatusOpen, Priority: tt.priorities[0], Title: "Title"},
				{ID: "t-0002", Type: ticket.TypeTask, Status: ticket.StatusOpen, Priority: tt.priorities[1], Title: "Title"},
			}

			if err := writeTickets(&output, tickets, tt.properties, false); err != nil {
				t.Fatalf("writeTickets() error = %v", err)
			}
			lines := strings.Split(strings.TrimSpace(output.String()), "\n")
			want := append([][]string{tt.header}, tt.rows...)
			if len(lines) != len(want) {
				t.Fatalf("output = %q, want %d lines", output.String(), len(want))
			}
			for index, line := range lines {
				if fields := strings.Fields(line); !slices.Equal(fields, want[index]) {
					t.Errorf("line %d = %q, want %q", index, fields, want[index])
				}
			}
		})
	}
}
