package card

import (
	"errors"
	"slices"
	"testing"

	"github.com/theroymind/go-tix/contracts/validation"
)

func TestCardValidate(t *testing.T) {
	valid := Card{ID: "grid-screen", Type: "task", Title: "Grid screen", Summary: "Builds the grid.", HasSummary: true,
		Distance: 1, Relation: RelationDependency, Commits: []string{"78c3fac"}, Files: []string{"client/main.odin"}}
	tests := []struct {
		name           string
		change         func(value *Card)
		expectedFields []string
	}{
		{"A card with every field", func(*Card) {}, nil},
		{"A card marked missing its summary", func(value *Card) { value.Summary, value.HasSummary = "", false }, nil},
		{"A summary marked missing", func(value *Card) { value.HasSummary = false }, []string{FieldHasSummary}},
		{"A blank summary marked present", func(value *Card) { value.Summary = " " }, []string{FieldHasSummary}},
		{"Identity fields are required", func(value *Card) { value.ID, value.Type, value.Title = "", " ", "" },
			[]string{FieldID, FieldType, FieldTitle}},
		{"A negative distance", func(value *Card) { value.Distance = minimumDistance - 1 }, []string{FieldDistance}},
		{"The start at distance 0", func(value *Card) { value.Distance, value.Relation = 0, RelationStart }, nil},
		{"An unknown relation", func(value *Card) { value.Relation = "sibling" }, []string{FieldRelation}},
		{"The start relation past distance 0", func(value *Card) { value.Relation = RelationStart }, []string{FieldRelation}},
		{"A hop relation at distance 0", func(value *Card) { value.Distance = 0 }, []string{FieldRelation}},
		{"Empty commits and files", func(value *Card) { value.Commits, value.Files = []string{""}, []string{"a", " "} },
			[]string{FieldCommits + ".0", FieldFiles + ".1"}},
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
