// Package show prints a sketch for a reader: a table of its tasks, files and items, markdown with each item's
// lines numbered the way rm and set count them, or a table of its open questions.
package show

import (
	"fmt"
	"io"
	"strconv"
	"strings"
	"text/tabwriter"

	"github.com/theroymind/go-tix/contracts/sketch"
	"github.com/theroymind/go-tix/internal/sketch/questions"
)

// Marker returns the mark for the item called name in the file at the repo path, such as + for new or ~ for
// changed.
type Marker func(path, name string) (string, error)

const (
	tableHeader     = "TASK\tFILE\tMARK\tTYPE\tITEM\tLINES"
	questionsHeader = "N\tCONTEXT\tQUESTION"
	tableMinimum    = 0
	tableTab        = 8
	tablePadding    = 2
	tablePad        = ' '
	noValue         = "-"
)

// Table writes one row per item, under the columns TASK, FILE, MARK, the item's mark from mark, TYPE, ITEM and
// LINES, the item's line count. A task without files, or a file without items, still gets a row, with - in the
// columns it lacks.
func Table(writer io.Writer, value sketch.Sketch, mark Marker) error {
	table := tabwriter.NewWriter(writer, tableMinimum, tableTab, tablePadding, tablePad, 0)
	rows := []string{tableHeader}
	for _, task := range value.Tasks {
		if len(task.Files) == 0 {
			rows = append(rows, row(task.Name, noValue, noValue, noValue, noValue, noValue))
		}
		for _, file := range task.Files {
			if len(file.Items) == 0 {
				rows = append(rows, row(task.Name, file.Path, noValue, noValue, noValue, noValue))
			}
			for _, item := range file.Items {
				itemMark, err := mark(file.Path, item.Name)
				if err != nil {
					return err
				}
				rows = append(rows, row(task.Name, file.Path, itemMark, string(item.Type), item.Name,
					strconv.Itoa(len(item.Lines))))
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

// Markdown writes the sketch of feature as a heading per task and file, a bullet per item led by its mark, and
// its lines as a numbered list, with a question's answer after it.
func Markdown(writer io.Writer, feature string, value sketch.Sketch, mark Marker) error {
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
				itemMark, err := mark(file.Path, item.Name)
				if err != nil {
					return err
				}
				fmt.Fprintf(&text, "- %s **%s** `%s`\n", itemMark, item.Type, item.Name)
				for index, line := range item.Lines {
					fmt.Fprintf(&text, "  %d. %s: %s\n", index+1, line.Type, line.Text)
					if line.Answer != "" {
						fmt.Fprintf(&text, "     answer: %s\n", line.Answer)
					}
				}
			}
		}
	}
	_, err := io.WriteString(writer, text.String())
	return err
}

// Questions writes one row per question under the columns N, CONTEXT and QUESTION, the numbers answer takes.
func Questions(writer io.Writer, open []questions.Question) error {
	table := tabwriter.NewWriter(writer, tableMinimum, tableTab, tablePadding, tablePad, 0)
	rows := []string{questionsHeader}
	for _, question := range open {
		rows = append(rows, row(strconv.Itoa(question.Number), question.Context, question.Question))
	}
	for _, text := range rows {
		if _, err := fmt.Fprintln(table, text); err != nil {
			return err
		}
	}
	return table.Flush()
}

func row(columns ...string) string {
	return strings.Join(columns, "\t")
}
