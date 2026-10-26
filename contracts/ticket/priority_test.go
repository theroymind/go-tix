package ticket

import (
	"errors"
	"strconv"
	"testing"

	"go.yaml.in/yaml/v3"
)

func TestPriorityUnmarshalYAML(t *testing.T) {
	tests := []struct {
		name        string
		input       string
		expected    Priority
		isTypeError bool
	}{
		{"An integer decodes", strconv.Itoa(int(PriorityLowest)), PriorityLowest, false},
		{"A fraction is rejected rather than truncated", "1.5", 0, true},
		{"A string is rejected", "high", 0, true},
		{"A list is rejected", "[1]", 0, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var priority Priority

			err := yaml.Unmarshal([]byte(tt.input), &priority)

			var typeError *yaml.TypeError
			if got := errors.As(err, &typeError); got != tt.isTypeError {
				t.Fatalf("yaml.Unmarshal(%q) error = %v, want type error %v", tt.input, err, tt.isTypeError)
			}
			if priority != tt.expected {
				t.Errorf("yaml.Unmarshal(%q) = %d, want %d", tt.input, priority, tt.expected)
			}
		})
	}
}
