package ticket

import (
	"fmt"

	"go.yaml.in/yaml/v3"
)

type Priority int

const (
	PriorityHighest Priority = 0
	PriorityLowest  Priority = 3
	PriorityDefault Priority = 2
)

func (priority Priority) IsValid() bool {
	return priority >= PriorityHighest && priority <= PriorityLowest
}

// UnmarshalYAML rejects non-integers, which the YAML decoder would otherwise truncate into an int.
func (priority *Priority) UnmarshalYAML(node *yaml.Node) error {
	if node.Kind != yaml.ScalarNode || node.ShortTag() != yamlIntegerTag {
		return &yaml.TypeError{Errors: []string{
			fmt.Sprintf("line %d: cannot unmarshal %s `%s` into %T", node.Line, node.ShortTag(), node.Value, *priority),
		}}
	}
	var value int
	if err := node.Decode(&value); err != nil {
		return err
	}
	*priority = Priority(value)
	return nil
}
