package card

import (
	"errors"
	"slices"
	"testing"

	"github.com/theroymind/go-tix/contracts/validation"
)

func TestHitValidate(t *testing.T) {
	valid := Hit{ID: "enemy-choice", Type: "task", Title: "Enemy choice", Summary: "Picks a target.", HasSummary: true, Score: 0.8}
	tests := []struct {
		name           string
		change         func(value *Hit)
		expectedFields []string
	}{
		{"A hit with every field", func(*Hit) {}, nil},
		{"A hit marked missing its summary", func(value *Hit) { value.Summary, value.HasSummary = "", false }, nil},
		{"A summary marked missing", func(value *Hit) { value.HasSummary = false }, []string{FieldHasSummary}},
		{"Identity fields are required", func(value *Hit) { value.ID, value.Type, value.Title = "", " ", "" },
			[]string{FieldID, FieldType, FieldTitle}},
		{"The lowest and highest scores", func(value *Hit) { value.Score = minimumScore }, nil},
		{"A score above 1", func(value *Hit) { value.Score = maximumScore + 0.5 }, []string{FieldScore}},
		{"A score below -1", func(value *Hit) { value.Score = minimumScore - 0.5 }, []string{FieldScore}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			value := valid
			tt.change(&value)

			err := value.Validate()

			var problems validation.Errors
			errors.As(err, &problems)
			if !slices.Equal(problems.Fields(), tt.expectedFields) {
				t.Errorf("Validate() fields = %q, want %q (error %v)", problems.Fields(), tt.expectedFields, err)
			}
		})
	}
}
