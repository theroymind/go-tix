// Package validation provides the field-level problems that contract Validate methods report.
package validation

import (
	"errors"
	"fmt"
	"strconv"
	"strings"
)

const (
	problemMarker  = "✖"
	fieldMarker    = "→ at"
	fieldIndent    = "  "
	optionJoiner   = "|"
	problemMissing = "Invalid input: field is required"
)

var ErrInvalid = errors.New("invalid")

type FieldError struct {
	Field   string
	Problem string
}

func (fieldError FieldError) Error() string {
	return fmt.Sprintf("%s %s\n%s%s %s", problemMarker, fieldError.Problem, fieldIndent, fieldMarker, fieldError.Field)
}

type Errors []FieldError

func (problems *Errors) Add(field, problem string) {
	*problems = append(*problems, FieldError{Field: field, Problem: problem})
}

func (problems Errors) Error() string {
	lines := make([]string, len(problems))
	for index, problem := range problems {
		lines[index] = problem.Error()
	}
	return strings.Join(lines, "\n")
}

func (problems Errors) Is(target error) bool {
	return target == ErrInvalid
}

func (problems Errors) Fields() []string {
	fields := make([]string, len(problems))
	for index, problem := range problems {
		fields[index] = problem.Field
	}
	return fields
}

func (problems Errors) Err() error {
	if len(problems) == 0 {
		return nil
	}
	return problems
}

func RequireFields(fields []string, hasField func(field string) bool) error {
	var problems Errors
	for _, field := range fields {
		if !hasField(field) {
			problems.Add(field, problemMissing)
		}
	}
	return problems.Err()
}

func InvalidOption[Option ~string](options []Option) string {
	quoted := make([]string, len(options))
	for index, option := range options {
		quoted[index] = strconv.Quote(string(option))
	}
	return "Invalid option: expected one of " + strings.Join(quoted, optionJoiner)
}

func TooSmall(minimum int) string {
	return fmt.Sprintf("Too small: expected number to be >=%d", minimum)
}

func TooBig(maximum int) string {
	return fmt.Sprintf("Too big: expected number to be <=%d", maximum)
}
