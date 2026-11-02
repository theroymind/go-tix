// Package ticket provides the Ticket contract that tix stores and the other Railyard apps read.
package ticket

import (
	"bytes"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/theroymind/railyard/internal/contracts/timestamp"
	"github.com/theroymind/railyard/internal/contracts/validation"
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
	FieldDescription  = "description"
	FieldAcceptance   = "acceptance"
	FieldCreatedAt    = "created_at"
	FieldUpdatedAt    = "updated_at"
)

var RequiredFields = []string{
	FieldID, FieldTitle, FieldType, FieldStatus, FieldPriority, FieldCreatedAt, FieldUpdatedAt,
}

type Ticket struct {
	ID            string              `yaml:"id" json:"id"`
	Title         string              `yaml:"title" json:"title"`
	Type          Type                `yaml:"type" json:"type"`
	Status        Status              `yaml:"status" json:"status"`
	Priority      Priority            `yaml:"priority" json:"priority"`
	Tags          []string            `yaml:"tags,omitempty,flow" json:"tags,omitempty"`
	Dependencies  []string            `yaml:"dependencies,omitempty,flow" json:"dependencies,omitempty"`
	Parent        string              `yaml:"parent,omitempty" json:"parent,omitempty"`
	Description   string              `yaml:"description,omitempty" json:"description,omitempty"`
	Acceptance    []Criterion         `yaml:"acceptance,omitempty" json:"acceptance,omitempty"`
	CreatedAt     timestamp.Timestamp `yaml:"created_at" json:"created_at"`
	UpdatedAt     timestamp.Timestamp `yaml:"updated_at" json:"updated_at"`
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
	if ticket.Priority < PriorityHighest {
		problems.Add(FieldPriority, validation.TooSmall(int(PriorityHighest)))
	}
	if ticket.Priority > PriorityLowest {
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
	if !ticket.CreatedAt.IsValid() {
		problems.Add(FieldCreatedAt, timestamp.ProblemInvalid)
	}
	if !ticket.UpdatedAt.IsValid() {
		problems.Add(FieldUpdatedAt, timestamp.ProblemInvalid)
	}
	return problems.Err()
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
