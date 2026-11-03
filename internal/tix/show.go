package tix

import (
	"fmt"
	"io"
	"os"

	"github.com/theroymind/railyard/internal/cli"
	"github.com/theroymind/railyard/internal/contracts/ticket"
	"github.com/theroymind/railyard/internal/tix/backend"
)

var showSpecification = cli.Specification{
	Options:           []cli.Option{{Name: optionJSON, Type: cli.OptionTypeBoolean}},
	AllowsPositionals: true,
}

func runShow(arguments []string, _ io.Reader, stdout io.Writer) error {
	parsed, err := cli.Parse(arguments, showSpecification)
	if err != nil {
		return err
	}
	if len(parsed.Positionals) != 1 {
		return cli.NewUsageError(ErrExpectedOneReference)
	}
	store, err := openBackend()
	if err != nil {
		return err
	}
	id, err := store.Resolve(parsed.Positionals[0])
	if err != nil {
		return err
	}
	// A backend without source text shows the ticket as JSON.
	if textReader, hasText := store.(backend.TextReader); hasText && !parsed.IsEnabled(optionJSON) {
		text, err := textReader.ReadText(id)
		if err != nil {
			return err
		}
		if _, err := stdout.Write(text); err != nil {
			return err
		}
		shown, err := store.Read(id)
		if err != nil {
			return err
		}
		_, tracksCriteria := store.(backend.CriterionTracker)
		separator := criteriaSeparator
		if len(text) > 0 && text[len(text)-1] != '\n' {
			// End the text's last line first, so the criteria still follow one blank line.
			separator = "\n" + separator
		}
		return writeCriteria(stdout, shown.Acceptance, tracksCriteria, separator)
	}
	shown, err := store.Read(id)
	if err != nil {
		return err
	}
	return cli.WriteJSON(stdout, shown)
}

// writeCriteria numbers the criteria after separator, with the numbers tick and ac set take, and a box only when
// the backend tracks whether each is met. It writes nothing for a ticket without criteria.
func writeCriteria(stdout io.Writer, criteria []ticket.Criterion, tracksCriteria bool, separator string) error {
	if len(criteria) == 0 {
		return nil
	}
	if _, err := fmt.Fprintf(stdout, "%s%s\n", separator, criteriaHeading); err != nil {
		return err
	}
	for index, criterion := range criteria {
		box := criterionUntracked
		if tracksCriteria {
			box = criterionUnmet
			if criterion.Done {
				box = criterionMet
			}
		}
		if _, err := fmt.Fprintf(stdout, criterionLine, index+1, box, criterion.Text); err != nil {
			return err
		}
	}
	return nil
}

func openBackend() (backend.Backend, error) {
	workingDirectory, err := os.Getwd()
	if err != nil {
		return nil, err
	}
	return backend.Open(workingDirectory)
}
