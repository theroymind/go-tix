package tix

import (
	"errors"
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

// runDone completes every ticket the arguments name. Unless --force is given, it first checks every ticket's
// criteria and changes none while any is unmet. A hand check doesn't hold a ticket open, so done names the ones it
// leaves open instead.
func runDone(arguments []string, _ io.Reader, stdout io.Writer) error {
	parsed, err := cli.Parse(arguments, doneSpecification)
	if err != nil {
		return err
	}
	if len(parsed.Positionals) == 0 {
		return cli.NewUsageError(ErrExpectedTicketReferences)
	}
	store, err := openBackend()
	if err != nil {
		return err
	}
	ids, err := resolveAll(store, parsed.Positionals)
	if err != nil {
		return err
	}
	// Only a backend whose criteria carry a done state can say a criterion is unmet.
	if _, tracksCriteria := store.(backend.CriterionTracker); !tracksCriteria {
		return setStatuses(store, ids, ticket.StatusCompleted)
	}
	completing, err := readAll(store, ids)
	if err != nil {
		return err
	}
	if !parsed.IsEnabled(optionForce) {
		if err := requireAllCriteriaMet(completing); err != nil {
			return err
		}
	}
	if err := setStatuses(store, ids, ticket.StatusCompleted); err != nil {
		return err
	}
	return writeOpenHandChecks(stdout, completing)
}

func readAll(store backend.Backend, ids []string) ([]ticket.Ticket, error) {
	tickets := make([]ticket.Ticket, 0, len(ids))
	for _, id := range ids {
		value, err := store.Read(id)
		if err != nil {
			return nil, err
		}
		tickets = append(tickets, value)
	}
	return tickets, nil
}

// requireAllCriteriaMet fails naming every ticket with an unmet criterion, and each of its unmet criteria.
func requireAllCriteriaMet(tickets []ticket.Ticket) error {
	var problems []error
	for _, value := range tickets {
		if err := requireCriteriaMet(value); err != nil {
			problems = append(problems, err)
		}
	}
	return errors.Join(problems...)
}

// requireCriteriaMet fails naming every unmet criterion that isn't a hand check by its 1-based index and text.
func requireCriteriaMet(value ticket.Ticket) error {
	var unmet []string
	for index, criterion := range value.Acceptance {
		if !criterion.Done && !isHandCheck(criterion) {
			unmet = append(unmet, strconv.Itoa(index+1)+": "+criterion.Text)
		}
	}
	if len(unmet) == 0 {
		return nil
	}
	return fmt.Errorf("%s %w (--force completes it anyway):\n  %s", value.ID, ErrUnmetCriteria, strings.Join(unmet, "\n  "))
}
