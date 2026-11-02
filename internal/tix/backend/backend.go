// Package backend provides the Backend interface every tix ticket store implements, the changes it applies,
// the .tix directory and config every backend shares, and Open, which opens the configured backend.
package backend

import "github.com/theroymind/railyard/internal/contracts/ticket"

// Backend stores tickets. Each backend chooses its tickets' ids and the format they're kept in.
type Backend interface {
	// Resolve returns the id that an exact id or a unique prefix refers to.
	Resolve(reference string) (string, error)
	List() ([]ticket.Ticket, error)
	Read(id string) (ticket.Ticket, error)
	// Create stores a new ticket under an id the backend assigns, ignoring value.ID, and returns that id.
	Create(value ticket.Ticket) (string, error)
	// Apply makes every change to the ticket or none of them. It fails with ErrUnsupported for a change the
	// backend's format can't hold.
	Apply(id string, changes []Change) error
}

// TextReader is implemented by a backend whose tickets have a source text, such as a file, for tix show to print.
type TextReader interface {
	ReadText(id string) ([]byte, error)
}

// Change is one edit Apply makes to a ticket: SetField, CheckCriterion, UncheckCriterion, SetCriterion or AddCriterion.
type Change interface {
	isChange()
}

// SetField sets the ticket field named by its contract key to Value.
type SetField struct {
	Field string
	Value any
}

// CheckCriterion marks the criterion at the 1-based Index met.
type CheckCriterion struct {
	Index int
}

// UncheckCriterion marks the criterion at the 1-based Index unmet.
type UncheckCriterion struct {
	Index int
}

// SetCriterion replaces the text of the criterion at the 1-based Index.
type SetCriterion struct {
	Index int
	Text  string
}

// AddCriterion appends an unmet criterion.
type AddCriterion struct {
	Text string
}

func (SetField) isChange()         {}
func (CheckCriterion) isChange()   {}
func (UncheckCriterion) isChange() {}
func (SetCriterion) isChange()     {}
func (AddCriterion) isChange()     {}
