// Package ticket provides the Ticket contract that tix stores and the other Railyard apps read.
package ticket

import (
	"bytes"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/theroymind/go-tix/contracts/timestamp"
	"github.com/theroymind/go-tix/contracts/validation"
)

const (
	FieldID           = "id"
	FieldTitle        = "title"
	FieldType         = "type"
	FieldStatus       = "status"
	FieldPriority     = "priority"
	FieldTags         = "tags"
	FieldDependencies = "dependencies"
	FieldParent       = "parent"
	FieldExternalID   = "external_id"
	FieldExternalURL  = "external_url"
	FieldDescription  = "description"
	FieldAcceptance   = "acceptance"
	FieldCreatedAt    = "created_at"
	FieldUpdatedAt    = "updated_at"
)

// Ticket is one unit of work. Priority, CreatedAt and UpdatedAt are optional: a nil Priority or an empty
// timestamp is absent, and JSON omits it. Priority 0 is the highest priority, not an absent one.
type Ticket struct {
	ID            string              `yaml:"id" json:"id"`
	Title         string              `yaml:"title" json:"title"`
	Type          Type                `yaml:"type" json:"type"`
	Status        Status              `yaml:"status" json:"status"`
	Priority      *Priority           `yaml:"priority,omitempty" json:"priority,omitempty"`
	Tags          []string            `yaml:"tags,omitempty,flow" json:"tags,omitempty"`
	Dependencies  []string            `yaml:"dependencies,omitempty,flow" json:"dependencies,omitempty"`
	Parent        string              `yaml:"parent,omitempty" json:"parent,omitempty"`
	ExternalID    string              `yaml:"external_id,omitempty" json:"external_id,omitempty"`
	ExternalURL   string              `yaml:"external_url,omitempty" json:"external_url,omitempty"`
	Description   string              `yaml:"description,omitempty" json:"description,omitempty"`
	Acceptance    []Criterion         `yaml:"acceptance,omitempty" json:"acceptance,omitempty"`
	CreatedAt     timestamp.Timestamp `yaml:"created_at,omitempty" json:"created_at,omitempty"`
	UpdatedAt     timestamp.Timestamp `yaml:"updated_at,omitempty" json:"updated_at,omitempty"`
	UnknownFields map[string]any      `yaml:",inline" json:"-"`
}

func (ticket Ticket) Validate() error {
	var problems validation.Errors
	if !IsValidID(ticket.ID) {
		problems.Add(FieldID, problemInvalidID)
	}
	if strings.TrimSpace(ticket.Title) == "" {
		problems.Add(FieldTitle, problemTitleRequired)
	}
	if !ticket.Type.IsValid() {
		problems.Add(FieldType, validation.InvalidOption(Types))
	}
	if !ticket.Status.IsValid() {
		problems.Add(FieldStatus, validation.InvalidOption(Statuses))
	}
	if ticket.Priority != nil && *ticket.Priority < PriorityHighest {
		problems.Add(FieldPriority, validation.TooSmall(int(PriorityHighest)))
	}
	if ticket.Priority != nil && *ticket.Priority > PriorityLowest {
		problems.Add(FieldPriority, validation.TooBig(int(PriorityLowest)))
	}
	for index, dependency := range ticket.Dependencies {
		if !IsValidID(dependency) {
			problems.Add(fmt.Sprintf("%s.%d", FieldDependencies, index), problemInvalidID)
		}
	}
	if ticket.Parent != "" && !IsValidID(ticket.Parent) {
		problems.Add(FieldParent, problemInvalidID)
	}
	if ticket.ExternalURL != "" && !IsValidExternalURL(ticket.ExternalURL) {
		problems.Add(FieldExternalURL, problemInvalidExternalURL)
	}
	if ticket.CreatedAt != "" && !ticket.CreatedAt.IsValid() {
		problems.Add(FieldCreatedAt, timestamp.ProblemInvalid)
	}
	if ticket.UpdatedAt != "" && !ticket.UpdatedAt.IsValid() {
		problems.Add(FieldUpdatedAt, timestamp.ProblemInvalid)
	}
	return problems.Err()
}

// PriorityOrDefault is the ticket's priority, or PriorityDefault when it has none.
func (ticket Ticket) PriorityOrDefault() Priority {
	if ticket.Priority == nil {
		return PriorityDefault
	}
	return *ticket.Priority
}

// MarshalJSON writes the known fields in contract order, then the unknown fields, so readers see everything the file holds.
func (ticket Ticket) MarshalJSON() ([]byte, error) {
	type knownFields Ticket
	known, err := marshalWithoutEscaping(knownFields(ticket))
	if err != nil || len(ticket.UnknownFields) == 0 {
		return known, err
	}
	unknown, err := marshalWithoutEscaping(ticket.UnknownFields)
	if err != nil {
		return nil, err
	}
	known = bytes.TrimSuffix(known, []byte("}"))
	return append(append(known, ','), bytes.TrimPrefix(unknown, []byte("{"))...), nil
}

func marshalWithoutEscaping(value any) ([]byte, error) {
	var buffer bytes.Buffer
	encoder := json.NewEncoder(&buffer)
	encoder.SetEscapeHTML(false)
	if err := encoder.Encode(value); err != nil {
		return nil, err
	}
	return bytes.TrimSuffix(buffer.Bytes(), []byte("\n")), nil
}
