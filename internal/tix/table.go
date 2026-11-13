package tix

import (
	"fmt"
	"io"
	"strconv"
	"strings"
	"text/tabwriter"

	"github.com/theroymind/go-tix/contracts/ticket"
	"github.com/theroymind/go-tix/internal/tix/backend"
	"github.com/theroymind/railyard-cli"
)

// propertyColumns are the project's own properties a ticket table shows: their names and each ticket's values by id.
type propertyColumns struct {
	names  []string
	values map[string][]string
}

// readPropertyColumns returns the columns a PropertyLister store names, and none for any other store.
func readPropertyColumns(store backend.Backend) (propertyColumns, error) {
	lister, isLister := store.(backend.PropertyLister)
	if !isLister || len(lister.ListProperties()) == 0 {
		return propertyColumns{}, nil
	}
	values, err := lister.PropertyValues()
	if err != nil {
		return propertyColumns{}, err
	}
	return propertyColumns{names: lister.ListProperties(), values: values}, nil
}

// writeTickets prints the tickets as a JSON array when asked, and otherwise as a table of ID, TYPE, STATUS, PRIORITY
// when any ticket has one, each property column, then TITLE. A JSON row leaves out the acceptance criteria, which
// tix show carries, so a list stays a list.
func writeTickets(stdout io.Writer, tickets []ticket.Ticket, properties propertyColumns, isJSON bool) error {
	if isJSON {
		return cli.WriteJSON(stdout, withoutAcceptance(tickets))
	}
	return writeTable(stdout, tickets, nil, properties)
}

// writeTable prints the tickets as writeTickets's table, each ID indented by its depth in depths.
func writeTable(stdout io.Writer, tickets []ticket.Ticket, depths map[string]int, properties propertyColumns) error {
	showsPriority := hasPriority(tickets)
	header := []string{tableColumnID, tableColumnType, tableColumnStatus}
	if showsPriority {
		header = append(header, tableColumnPriority)
	}
	for _, name := range properties.names {
		header = append(header, strings.ToUpper(name))
	}
	header = append(header, tableColumnTitle)

	table := tabwriter.NewWriter(stdout, tableMinimumWidth, tableTabWidth, tablePadding, tablePaddingCharacter, 0)
	if _, err := fmt.Fprintln(table, strings.Join(header, tableSeparator)); err != nil {
		return err
	}
	for _, row := range tickets {
		cells := []string{strings.Repeat(tableTreeIndent, depths[row.ID]) + row.ID, string(row.Type), string(row.Status)}
		if showsPriority {
			cells = append(cells, priorityCell(row.Priority))
		}
		for index := range properties.names {
			cells = append(cells, propertyCell(properties.values[row.ID], index))
		}
		cells = append(cells, row.Title)
		if _, err := fmt.Fprintln(table, strings.Join(cells, tableSeparator)); err != nil {
			return err
		}
	}
	return table.Flush()
}

func hasPriority(tickets []ticket.Ticket) bool {
	for _, row := range tickets {
		if row.Priority != nil {
			return true
		}
	}
	return false
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

// propertyCell is the ticket's value in the property column at index, or a dash when it has none.
func propertyCell(values []string, index int) string {
	if index >= len(values) || values[index] == "" {
		return tableAbsent
	}
	return values[index]
}
