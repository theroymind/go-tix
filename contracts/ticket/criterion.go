package ticket

import "go.yaml.in/yaml/v3"

// Criterion is one acceptance criterion and whether it has been met.
// In YAML it is a plain string, as tix v1 writes it, and loads with Done false; in JSON it is an object.
type Criterion struct {
	Text string `json:"text"`
	Done bool   `json:"done"`
}

// NewCriteria returns one unmet criterion per text, or nil when there are none.
func NewCriteria(texts []string) []Criterion {
	if len(texts) == 0 {
		return nil
	}
	criteria := make([]Criterion, len(texts))
	for index, text := range texts {
		criteria[index] = Criterion{Text: text}
	}
	return criteria
}

// MarshalYAML writes the criterion as its text, so .tix/ files keep tix v1's bytes.
func (criterion Criterion) MarshalYAML() (any, error) {
	return criterion.Text, nil
}

// UnmarshalYAML reads a plain string as an unmet criterion, rejecting what a string field would reject.
func (criterion *Criterion) UnmarshalYAML(node *yaml.Node) error {
	var text string
	if err := node.Decode(&text); err != nil {
		return err
	}
	*criterion = Criterion{Text: text}
	return nil
}
