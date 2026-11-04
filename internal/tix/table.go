package tix

import (
	"fmt"
	"io"
	"strconv"
	"text/tabwriter"

	"github.com/theroymind/go-tix/contracts/ticket"
	"github.com/theroymind/railyard-cli"
)

// writeTickets prints the tickets as a JSON array when asked, and as an ID TYPE STATUS PRIORITY TITLE table otherwise.
// A JSON row leaves out the acceptance criteria, which tix show carries, so a list stays a list.
func writeTickets(stdout io.Writer, tickets []ticket.Ticket, isJSON bool) error {
	if isJSON {
		return cli.WriteJSON(stdout, withoutAcceptance(tickets))
	}
	table := tabwriter.NewWriter(stdout, tableMinimumWidth, tableTabWidth, tablePadding, tablePaddingCharacter, 0)
	if _, err := fmt.Fprintln(table, tableHeader); err != nil {
		return err
	}
	for _, row := range tickets {
		if _, err := fmt.Fprintf(table, tableRow, row.ID, row.Type, row.Status, priorityCell(row.Priority), row.Title); err != nil {
			return err
		}
	}
	return table.Flush()
}

// withoutAcceptance returns copies of the tickets with no acceptance criteria.
func withoutAcceptance(tickets []ticket.Ticket) []ticket.Ticket {
	rows := make([]ticket.Ticket, len(tickets))
	for index, row := range tickets {
		row.Acceptance = nil
		rows[index] = row
	}
	return rows
}

// priorityCell is the priority as a number, or a dash when the ticket has none.
func priorityCell(priority *ticket.Priority) string {
	if priority == nil {
		return tableAbsent
	}
	return strconv.Itoa(int(*priority))
}
