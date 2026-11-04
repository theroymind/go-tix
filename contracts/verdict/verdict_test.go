package verdict_test

import (
	"errors"
	"reflect"
	"testing"

	"github.com/theroymind/go-tix/contracts/validation"
	"github.com/theroymind/go-tix/contracts/verdict"
	"github.com/theroymind/railyard-cli/strictjson"
)

func TestVerdictDecodeAndValidate(t *testing.T) {
	tests := []struct {
		name            string
		text            string
		expectedDecode  bool
		expectedInvalid []string
	}{
		{"An acceptance naming no note is valid",
			`{"verdict":"accept","duplicate_of":"","conflicts_with":"","reason":"New ground."}`, true, nil},
		{"A refusal as a duplicate is valid",
			`{"verdict":"refuse","duplicate_of":"turn-order-is-a-wait-counter","conflicts_with":"","reason":"Restates it."}`, true, nil},
		{"A refusal as a conflict is valid",
			`{"verdict":"refuse","duplicate_of":"","conflicts_with":"turn-order-is-a-wait-counter","reason":"Contradicts it."}`, true, nil},
		{"An unknown field is rejected",
			`{"verdict":"accept","duplicate_of":"","conflicts_with":"","reason":"Fine.","confidence":0.9}`, false, nil},
		{"A field in another case is rejected",
			`{"Verdict":"accept","duplicate_of":"","conflicts_with":"","reason":"Fine."}`, false, nil},
		{"A duplicate field is rejected",
			`{"verdict":"accept","verdict":"refuse","duplicate_of":"","conflicts_with":"","reason":"Fine."}`, false, nil},
		{"Text after the value is rejected",
			`{"verdict":"accept","duplicate_of":"","conflicts_with":"","reason":"Fine."} and more`, false, nil},
		{"An unknown verdict is named",
			`{"verdict":"maybe","duplicate_of":"","conflicts_with":"","reason":"Unsure."}`, true, []string{verdict.FieldVerdict}},
		{"A missing reason is named",
			`{"verdict":"accept","duplicate_of":"","conflicts_with":""}`, true, []string{verdict.FieldReason}},
		{"A refusal naming no note is named",
			`{"verdict":"refuse","duplicate_of":"","conflicts_with":"","reason":"Weak."}`, true, []string{verdict.FieldDuplicateOf}},
		{"An acceptance naming a note is named",
			`{"verdict":"accept","duplicate_of":"","conflicts_with":"x","reason":"Fine."}`, true, []string{verdict.FieldDuplicateOf}},
		{"An id padded with spaces is named",
			`{"verdict":"refuse","duplicate_of":" x","conflicts_with":"","reason":"Same."}`, true, []string{verdict.FieldDuplicateOf}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var value verdict.Verdict
			decodeErr := strictjson.Unmarshal([]byte(tt.text), &value)

			if (decodeErr == nil) != tt.expectedDecode {
				t.Fatalf("Unmarshal() error = %v, expected decoding to succeed: %t", decodeErr, tt.expectedDecode)
			}
			if decodeErr != nil {
				return
			}
			err := value.Validate()
			var problems validation.Errors
			errors.As(err, &problems)
			if !reflect.DeepEqual(problems.Fields(), nonNil(tt.expectedInvalid)) {
				t.Errorf("Validate() fields = %v, expected %v; error %v", problems.Fields(), tt.expectedInvalid, err)
			}
		})
	}
}

func nonNil(fields []string) []string {
	if fields == nil {
		return []string{}
	}
	return fields
}
