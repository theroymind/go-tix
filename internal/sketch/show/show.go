// Package show prints a sketch for a reader: a table of its tasks, files and items, or markdown with each item's
// lines numbered the way rm and set count them.
package show

import (
	"fmt"
	"io"
	"strconv"
	"strings"
	"text/tabwriter"

	"github.com/theroymind/go-tix/contracts/sketch"
)

const (
	tableHeader  = "TASK\tFILE\tTYPE\tITEM\tLINES"
	tableMinimum = 0
	tableTab     = 8
	tablePadding = 2
	tablePad     = ' '
	noValue      = "-"
)

// Table writes one row per item, under the columns TASK, FILE, TYPE, ITEM and LINES, the item's line count. A task
// without files, or a file without items, still gets a row, with - in the columns it lacks.
func Table(writer io.Writer, value sketch.Sketch) error {
	table := tabwriter.NewWriter(writer, tableMinimum, tableTab, tablePadding, tablePad, 0)
	rows := []string{tableHeader}
	for _, task := range value.Tasks {
		if len(task.Files) == 0 {
			rows = append(rows, row(task.Name, noValue, noValue, noValue, noValue))
		}
		for _, file := range task.Files {
			if len(file.Items) == 0 {
				rows = append(rows, row(task.Name, file.Path, noValue, noValue, noValue))
			}
			for _, item := range file.Items {
				rows = append(rows, row(task.Name, file.Path, string(item.Type), item.Name, strconv.Itoa(len(item.Lines))))
			}
		}
	}
	for _, text := range rows {
		if _, err := fmt.Fprintln(table, text); err != nil {
			return err
		}
	}
	return table.Flush()
}

// Markdown writes the sketch of feature as a heading per task and file, a bullet per item, and its lines as a
// numbered list.
func Markdown(writer io.Writer, feature string, value sketch.Sketch) error {
	var text strings.Builder
	fmt.Fprintf(&text, "# %s\n", feature)
	for _, task := range value.Tasks {
		fmt.Fprintf(&text, "\n## %s\n", task.Name)
		for _, file := range task.Files {
			fmt.Fprintf(&text, "\n### `%s`\n", file.Path)
			if len(file.Items) > 0 {
				text.WriteString("\n")
			}
			for _, item := range file.Items {
				fmt.Fprintf(&text, "- **%s** `%s`\n", item.Type, item.Name)
				for index, line := range item.Lines {
					fmt.Fprintf(&text, "  %d. %s: %s\n", index+1, line.Type, line.Text)
				}
			}
		}
	}
	_, err := io.WriteString(writer, text.String())
	return err
}

func row(columns ...string) string {
	return strings.Join(columns, "\t")
}
