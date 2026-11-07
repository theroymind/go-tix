package tix

import (
	"fmt"
	"io"
	"strconv"
	"strings"

	"github.com/theroymind/go-tix/contracts/ticket"
	"github.com/theroymind/go-tix/internal/tix/backend"
	"github.com/theroymind/railyard-cli"
)

var doneSpecification = cli.Specification{
	Options:           []cli.Option{{Name: optionForce, Type: cli.OptionTypeBoolean}},
	AllowsPositionals: true,
}

func runDone(arguments []string, _ io.Reader, _ io.Writer) error {
	parsed, err := cli.Parse(arguments, doneSpecification)
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
	// Only a backend whose criteria carry a done state can say a criterion is unmet.
	if _, tracksCriteria := store.(backend.CriterionTracker); tracksCriteria && !parsed.IsEnabled(optionForce) {
		completing, err := store.Read(id)
		if err != nil {
			return err
		}
		if err := requireCriteriaMet(completing); err != nil {
			return err
		}
	}
	return setStatus(store, id, ticket.StatusCompleted)
}

// requireCriteriaMet fails naming every unmet criterion by its 1-based index and text.
func requireCriteriaMet(value ticket.Ticket) error {
	var unmet []string
	for index, criterion := range value.Acceptance {
		if !criterion.Done {
			unmet = append(unmet, strconv.Itoa(index+1)+": "+criterion.Text)
		}
	}
	if len(unmet) == 0 {
		return nil
	}
	return fmt.Errorf("%s %w (--force completes it anyway):\n  %s", value.ID, ErrUnmetCriteria, strings.Join(unmet, "\n  "))
}
