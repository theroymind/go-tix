package ticket

import (
	"encoding/json"
	"slices"
	"strings"
	"testing"

	"go.yaml.in/yaml/v3"

	"github.com/theroymind/railyard-cli/strictjson"
)

func TestCriterionYAMLIsAPlainString(t *testing.T) {
	tests := []struct {
		name     string
		text     string
		criteria []Criterion
	}{
		{"Plain strings load as unmet criteria", "- one\n- two\n", []Criterion{{Text: "one"}, {Text: "two"}}},
		{"A quoted string keeps its text", "- \"Quoted: because of the colon\"\n", []Criterion{{Text: "Quoted: because of the colon"}}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var decoded []Criterion
			if err := yaml.Unmarshal([]byte(tt.text), &decoded); err != nil {
				t.Fatalf("yaml.Unmarshal(%q) error = %v", tt.text, err)
			}
			if !slices.Equal(decoded, tt.criteria) {
				t.Errorf("yaml.Unmarshal(%q) = %v, want %v", tt.text, decoded, tt.criteria)
			}
			encoded, err := yaml.Marshal(decoded)
			if err != nil {
				t.Fatalf("yaml.Marshal() error = %v", err)
			}
			var asStrings []string
			if err := yaml.Unmarshal(encoded, &asStrings); err != nil {
				t.Errorf("yaml.Marshal() = %q, not a list of strings: %v", encoded, err)
			}
		})
	}
}

func TestCriterionYAMLMarshalsMetCriterionAsItsText(t *testing.T) {
	encoded, err := yaml.Marshal([]Criterion{{Text: "done already", Done: true}})
	if err != nil {
		t.Fatalf("yaml.Marshal() error = %v", err)
	}
	if want := "- done already\n"; string(encoded) != want {
		t.Errorf("yaml.Marshal() = %q, want %q", encoded, want)
	}
}

func TestCriterionYAMLRejectsANonString(t *testing.T) {
	tests := []struct {
		name string
		text string
	}{
		{"A mapping", "- {text: one, done: true}\n"},
		{"A list", "- [one]\n"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var decoded []Criterion
			if err := yaml.Unmarshal([]byte(tt.text), &decoded); err == nil {
				t.Errorf("yaml.Unmarshal(%q) error = nil, want an error", tt.text)
			}
		})
	}
}

func TestCriterionJSON(t *testing.T) {
	tests := []struct {
		name      string
		criterion Criterion
		json      string
	}{
		{"An unmet criterion", Criterion{Text: "one"}, `{"text":"one","done":false}`},
		{"A met criterion", Criterion{Text: "two", Done: true}, `{"text":"two","done":true}`},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			encoded, err := json.Marshal(tt.criterion)
			if err != nil {
				t.Fatalf("json.Marshal() error = %v", err)
			}
			if string(encoded) != tt.json {
				t.Errorf("json.Marshal() = %s, want %s", encoded, tt.json)
			}
			var decoded Criterion
			if err := strictjson.Unmarshal(encoded, &decoded); err != nil {
				t.Fatalf("strictjson.Unmarshal(%s) error = %v", encoded, err)
			}
			if decoded != tt.criterion {
				t.Errorf("strictjson.Unmarshal(%s) = %v, want %v", encoded, decoded, tt.criterion)
			}
		})
	}
}

func TestCriterionJSONRejectsLooseFields(t *testing.T) {
	tests := []struct {
		name  string
		input string
	}{
		{"An unknown field", `{"text":"one","done":false,"extra":1}`},
		{"A plain string", `"one"`},
		{"A field name in another case", `{"Text":"one","done":false}`},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var decoded Criterion
			if err := strictjson.Unmarshal([]byte(tt.input), &decoded); err == nil {
				t.Errorf("strictjson.Unmarshal(%s) error = nil, want an error", tt.input)
			}
		})
	}
}

func TestTicketJSONWritesAcceptanceAsObjects(t *testing.T) {
	ticket := validTicket()
	ticket.Acceptance = []Criterion{{Text: "one"}, {Text: "two", Done: true}}

	encoded, err := json.Marshal(ticket)
	if err != nil {
		t.Fatalf("json.Marshal() error = %v", err)
	}
	want := `"acceptance":[{"text":"one","done":false},{"text":"two","done":true}]`
	if !strings.Contains(string(encoded), want) {
		t.Errorf("json.Marshal() = %s, want it to contain %s", encoded, want)
	}
}

func TestNewCriteria(t *testing.T) {
	tests := []struct {
		name  string
		texts []string
		want  []Criterion
	}{
		{"No texts", nil, nil},
		{"Empty texts", []string{}, nil},
		{"Some texts", []string{"one", "two"}, []Criterion{{Text: "one"}, {Text: "two"}}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := NewCriteria(tt.texts); !slices.Equal(got, tt.want) || (got == nil) != (tt.want == nil) {
				t.Errorf("NewCriteria(%v) = %#v, want %#v", tt.texts, got, tt.want)
			}
		})
	}
}
