package store

import (
	"strings"

	"go.yaml.in/yaml/v3"

	"github.com/theroymind/railyard/internal/contracts/ticket"
	"github.com/theroymind/railyard/internal/contracts/validation"
)

const (
	doubleQuote = `"`
	singleQuote = "'"
)

type Change struct {
	Field string
	Value any
}

func ParseTicket(text []byte, path string) (ticket.Ticket, error) {
	document, err := parseYAML(text, path)
	if err != nil {
		return ticket.Ticket{}, err
	}
	return decodeTicket(document, path)
}

func decodeTicket(document *yaml.Node, path string) (ticket.Ticket, error) {
	if err := validation.RequireFields(ticket.RequiredFields, hasKey(mappingRoot(document))); err != nil {
		return ticket.Ticket{}, InvalidFile(path, err)
	}
	var decoded ticket.Ticket
	if err := decodeYAML(document, &decoded, path); err != nil {
		return ticket.Ticket{}, err
	}
	return decoded, Validate(decoded, path)
}

func SerializeTicket(value ticket.Ticket) ([]byte, error) {
	text, err := encodeYAML(value)
	if err != nil {
		return nil, err
	}
	var document yaml.Node
	if err := yaml.Unmarshal(text, &document); err != nil {
		return nil, err
	}
	preferDoubleQuotes(&document)
	return encodeYAML(&document)
}

// preferDoubleQuotes matches tix v1 files, which double-quote strings that can't be written plain;
// the encoder picks single quotes, and its choice only shows once the output is parsed back.
func preferDoubleQuotes(node *yaml.Node) {
	isOnlyDoubleQuoted := strings.Contains(node.Value, doubleQuote) && !strings.Contains(node.Value, singleQuote)
	if node.Style == yaml.SingleQuotedStyle && !isOnlyDoubleQuoted {
		node.Style = yaml.DoubleQuotedStyle
	}
	for _, child := range node.Content {
		preferDoubleQuotes(child)
	}
}

func UpdateTicketText(text []byte, changes []Change, path string) ([]byte, error) {
	document, err := parseYAML(text, path)
	if err != nil {
		return nil, err
	}
	root := mappingRoot(document)
	if root == nil {
		return nil, InvalidFile(path, validation.RequireFields(ticket.RequiredFields, hasKey(root)))
	}
	for _, change := range changes {
		if err := setField(root, change); err != nil {
			return nil, err
		}
	}
	if _, err := decodeTicket(document, path); err != nil {
		return nil, err
	}
	return encodeYAML(document)
}

func setField(mapping *yaml.Node, change Change) error {
	var value yaml.Node
	if err := value.Encode(change.Value); err != nil {
		return err
	}
	index := valueIndex(mapping, change.Field)
	if index < 0 {
		mapping.Content = append(mapping.Content, &yaml.Node{Kind: yaml.ScalarNode, Value: change.Field}, &value)
		return nil
	}
	existing := mapping.Content[index]
	if existing.Kind == yaml.ScalarNode && value.Kind == yaml.ScalarNode {
		existing.Value, existing.Tag = value.Value, value.Tag
		return nil
	}
	mapping.Content[index] = &value
	return nil
}
