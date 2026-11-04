package rankeval

import (
	"fmt"
	"strconv"

	"github.com/theroymind/go-tix/contracts/card"
	"github.com/theroymind/go-tix/contracts/ticket"
	"github.com/theroymind/railyard-cli"
)

// vault reads a vault through the tix binary run at its root: its done tasks, one task with its criteria, and
// the cards a walk from a task reaches.
type vault struct {
	root string
}

type ticketList []ticket.Ticket

func (list ticketList) Validate() error {
	for index, item := range list {
		if err := item.Validate(); err != nil {
			return fmt.Errorf("%d: %w", index, err)
		}
	}
	return nil
}

type cardList []card.Card

func (list cardList) Validate() error {
	for index, item := range list {
		if err := item.Validate(); err != nil {
			return fmt.Errorf("%d: %w", index, err)
		}
	}
	return nil
}

func (store vault) doneTasks() ([]ticket.Ticket, error) {
	listed, err := cli.ReadChildJSON[ticketList](store.root, tixExecutable, []string{"ls", "--type", string(ticket.TypeTask), "--all", "--json"}, nil)
	if err != nil {
		return nil, err
	}
	var done []ticket.Ticket
	for _, item := range listed {
		if item.Status == ticket.StatusCompleted {
			done = append(done, item)
		}
	}
	return done, nil
}

func (store vault) show(id string) (ticket.Ticket, error) {
	return cli.ReadChildJSON[ticket.Ticket](store.root, tixExecutable, []string{"show", id, "--json"}, nil)
}

func (store vault) context(id string, depth int) ([]card.Card, error) {
	return cli.ReadChildJSON[cardList](store.root, tixExecutable, []string{"context", id, "--depth", strconv.Itoa(depth), "--json"}, nil)
}
