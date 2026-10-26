package tix

import (
	"fmt"
	"io"
	"text/tabwriter"

	"github.com/theroymind/railyard/internal/cli"
	"github.com/theroymind/railyard/internal/contracts/ticket"
)

// writeTickets prints the tickets as a JSON array when asked, and as an ID TYPE STATUS PRIORITY TITLE table otherwise.
func writeTickets(stdout io.Writer, tickets []ticket.Ticket, isJSON bool) error {
	if isJSON {
		return cli.WriteJSON(stdout, tickets)
	}
	table := tabwriter.NewWriter(stdout, tableMinimumWidth, tableTabWidth, tablePadding, tablePaddingCharacter, 0)
	if _, err := fmt.Fprintln(table, tableHeader); err != nil {
		return err
	}
	for _, row := range tickets {
		if _, err := fmt.Fprintf(table, tableRow, row.ID, row.Type, row.Status, row.Priority, row.Title); err != nil {
			return err
		}
	}
	return table.Flush()
}
