package yamlstore

import (
	"strings"

	"go.yaml.in/yaml/v3"

	"github.com/theroymind/railyard/internal/contracts/ticket"
	"github.com/theroymind/railyard/internal/contracts/validation"
	"github.com/theroymind/railyard/internal/tix/backend"
)

func ParseTicket(text []byte, path string) (ticket.Ticket, error) {
	document, err := backend.ParseYAML(text, path)
	if err != nil {
		return ticket.Ticket{}, err
	}
	return decodeTicket(document, path)
}

func decodeTicket(document *yaml.Node, path string) (ticket.Ticket, error) {
	if err := validation.RequireFields(requiredFields, hasKey(mappingRoot(document))); err != nil {
		return ticket.Ticket{}, backend.InvalidFile(path, err)
	}
	var decoded ticket.Ticket
	if err := backend.DecodeYAML(document, &decoded, path); err != nil {
		return ticket.Ticket{}, err
	}
	if err := validation.RequireFields(requiredFields, hasValue(decoded)); err != nil {
		return ticket.Ticket{}, backend.InvalidFile(path, err)
	}
	return decoded, backend.Validate(decoded, path)
}

// hasValue reports whether a field decoded to a value. A key holding null or an empty string leaves
// priority or a timestamp absent, which the contract allows and a ticket file does not.
func hasValue(decoded ticket.Ticket) func(field string) bool {
	return func(field string) bool {
		switch field {
		case ticket.FieldPriority:
			return decoded.Priority != nil
		case ticket.FieldCreatedAt:
			return decoded.CreatedAt != ""
		case ticket.FieldUpdatedAt:
			return decoded.UpdatedAt != ""
		default:
			return true
		}
	}
}

func SerializeTicket(value ticket.Ticket) ([]byte, error) {
	text, err := backend.EncodeYAML(value)
	if err != nil {
		return nil, err
	}
	var document yaml.Node
	if err := yaml.Unmarshal(text, &document); err != nil {
		return nil, err
	}
	preferDoubleQuotes(&document)
	return backend.EncodeYAML(&document)
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

// UpdateTicketText applies changes to a ticket file's text, leaving every line they don't touch as it was.
func UpdateTicketText(text []byte, changes []backend.Change, path string) ([]byte, error) {
	document, err := backend.ParseYAML(text, path)
	if err != nil {
		return nil, err
	}
	root := mappingRoot(document)
	if root == nil {
		return nil, backend.InvalidFile(path, validation.RequireFields(requiredFields, hasKey(root)))
	}
	for _, change := range changes {
		if err := applyChange(root, change, path); err != nil {
			return nil, err
		}
	}
	if _, err := decodeTicket(document, path); err != nil {
		return nil, err
	}
	return backend.EncodeYAML(document)
}

// applyChange edits the mapping; a v1 file has no done state for a criterion, so checking one is unsupported.
func applyChange(mapping *yaml.Node, change backend.Change, path string) error {
	switch change := change.(type) {
	case backend.SetField:
		return setField(mapping, change)
	case backend.SetCriterion:
		return setCriterion(mapping, change, path)
	case backend.AddCriterion:
		return addCriterion(mapping, change, path)
	default:
		return backend.Unsupported(change, backend.KindYAML)
	}
}

func setField(mapping *yaml.Node, change backend.SetField) error {
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

func setCriterion(mapping *yaml.Node, change backend.SetCriterion, path string) error {
	index := valueIndex(mapping, ticket.FieldAcceptance)
	if index < 0 || mapping.Content[index].Kind != yaml.SequenceNode || change.Index < 1 || change.Index > len(mapping.Content[index].Content) {
		return backend.NoCriterion(path, change.Index)
	}
	replacement, err := criterionNode(change.Text)
	if err != nil {
		return err
	}
	existing := mapping.Content[index].Content[change.Index-1]
	if existing.Kind != yaml.ScalarNode {
		mapping.Content[index].Content[change.Index-1] = replacement
		return nil
	}
	existing.Value, existing.Tag, existing.Style = replacement.Value, replacement.Tag, replacement.Style
	return nil
}

func addCriterion(mapping *yaml.Node, change backend.AddCriterion, path string) error {
	added, err := criterionNode(change.Text)
	if err != nil {
		return err
	}
	index := valueIndex(mapping, ticket.FieldAcceptance)
	if index < 0 {
		mapping.Content = append(mapping.Content, &yaml.Node{Kind: yaml.ScalarNode, Value: ticket.FieldAcceptance}, blockSequence(added))
		return nil
	}
	criteria := mapping.Content[index]
	switch {
	case criteria.Kind == yaml.SequenceNode && len(criteria.Content) == 0:
		mapping.Content[index] = blockSequence(added)
	case criteria.Kind == yaml.SequenceNode:
		criteria.Content = append(criteria.Content, added)
	case criteria.Kind == yaml.ScalarNode && criteria.Tag == nullTag:
		mapping.Content[index] = blockSequence(added)
	default:
		var problems validation.Errors
		problems.Add(ticket.FieldAcceptance, problemNotList)
		return backend.InvalidFile(path, problems)
	}
	return nil
}

func blockSequence(items ...*yaml.Node) *yaml.Node {
	return &yaml.Node{Kind: yaml.SequenceNode, Tag: sequenceTag, Content: items}
}

// criterionNode is text as a scalar quoted the way SerializeTicket quotes it, so an edited criterion reads like v1 wrote it.
func criterionNode(text string) (*yaml.Node, error) {
	encoded, err := backend.EncodeYAML(text)
	if err != nil {
		return nil, err
	}
	var document yaml.Node
	if err := yaml.Unmarshal(encoded, &document); err != nil {
		return nil, err
	}
	preferDoubleQuotes(&document)
	return document.Content[0], nil
}
