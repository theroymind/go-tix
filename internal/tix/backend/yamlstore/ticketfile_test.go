package yamlstore

import (
	"bytes"
	"errors"
	"path/filepath"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"testing"

	"github.com/theroymind/go-tix/contracts/ticket"
	"github.com/theroymind/go-tix/contracts/timestamp"
	"github.com/theroymind/go-tix/contracts/validation"
	"github.com/theroymind/go-tix/internal/tix/backend"
)

const (
	unknownField = "estimate"
	completedAt  = timestamp.Timestamp("2026-10-25T09:30:00-04:00")
)

var fixturePath = filepath.Join("testdata", "ticket.yml")

func readFixture(t *testing.T) []byte {
	t.Helper()
	return readFile(t, fixturePath)
}

func parseFixture(t *testing.T) ticket.Ticket {
	t.Helper()
	parsed, err := ParseTicket(readFixture(t), fixturePath)
	if err != nil {
		t.Fatalf("ParseTicket(%s) error = %v", fixturePath, err)
	}
	return parsed
}

func serialize(t *testing.T, value ticket.Ticket) []byte {
	t.Helper()
	text, err := SerializeTicket(value)
	if err != nil {
		t.Fatalf("SerializeTicket() error = %v", err)
	}
	return text
}

func changedLines(before, after []byte) []string {
	beforeLines := strings.Split(string(before), "\n")
	var changed []string
	for index, line := range strings.Split(string(after), "\n") {
		if index >= len(beforeLines) || line != beforeLines[index] {
			changed = append(changed, line)
		}
	}
	return changed
}

func TestSerializeTicketRoundTripsAV1File(t *testing.T) {
	fixture := readFixture(t)

	if got := serialize(t, parseFixture(t)); !bytes.Equal(got, fixture) {
		t.Errorf("SerializeTicket(ParseTicket(fixture)) =\n%s\nwant\n%s", got, fixture)
	}
}

func TestParseTicketLoadsCriteriaUnmet(t *testing.T) {
	want := []ticket.Criterion{
		{Text: "Policy buys the cheapest affordable purchasable each tick"},
		{Text: "Quoted: because of the colon"},
	}

	if got := parseFixture(t).Acceptance; !slices.Equal(got, want) {
		t.Errorf("ParseTicket(fixture).Acceptance = %v, want %v", got, want)
	}
}

func TestSerializeTicketWritesAMetCriterionAsItsText(t *testing.T) {
	met := parseFixture(t)
	for index := range met.Acceptance {
		met.Acceptance[index].Done = true
	}

	if got := serialize(t, met); !bytes.Equal(got, readFixture(t)) {
		t.Errorf("SerializeTicket() with met criteria =\n%s\nwant\n%s", got, readFixture(t))
	}
}

func TestSerializeTicketOmitsEmptyOptionalFields(t *testing.T) {
	minimal := parseFixture(t)
	minimal.Tags = []string{}
	minimal.Dependencies = nil
	minimal.Description = ""

	text := serialize(t, minimal)

	omitted := regexp.MustCompile(`(?m)^(` + ticket.FieldTags + `|` + ticket.FieldDependencies + `|` + ticket.FieldDescription + `):`)
	if omitted.Match(text) {
		t.Errorf("SerializeTicket() wrote an empty optional field:\n%s", text)
	}
}

func TestSerializeTicketQuotesLikeV1(t *testing.T) {
	tests := []struct {
		name     string
		title    string
		expected string
	}{
		{"A plain string stays plain", "Plain title", `title: Plain title`},
		{"A string that can't be plain is double-quoted", "Hello: world", `title: "Hello: world"`},
		{"A string that would read as a number is double-quoted", "123", `title: "123"`},
		{"A string with only double quotes is single-quoted to avoid escaping", `Say "hi": now`, `title: 'Say "hi": now'`},
		{"A string with a single quote stays double-quoted", "It's: here", `title: "It's: here"`},
		{"A long string is not folded", strings.Repeat("long ", 40) + "end", "title: " + strings.Repeat("long ", 40) + "end"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			value := parseFixture(t)
			value.Title = tt.title

			text := serialize(t, value)

			if !slices.Contains(strings.Split(string(text), "\n"), tt.expected) {
				t.Errorf("SerializeTicket() =\n%s\nwant a line %q", text, tt.expected)
			}
			reparsed, err := ParseTicket(text, fixturePath)
			if err != nil {
				t.Fatalf("ParseTicket() error = %v", err)
			}
			if reparsed.Title != tt.title {
				t.Errorf("title round-tripped to %q, want %q", reparsed.Title, tt.title)
			}
		})
	}
}

func TestUpdateTicketTextChangesOnlyTheUpdatedFieldsAndKeepsUnknownOnes(t *testing.T) {
	withUnknownField := append(readFixture(t), []byte(unknownField+": 2d\n")...)
	changes := []backend.Change{
		backend.SetField{Field: ticket.FieldStatus, Value: ticket.StatusCompleted},
		backend.SetField{Field: ticket.FieldUpdatedAt, Value: completedAt},
	}

	updated, err := UpdateTicketText(withUnknownField, changes, fixturePath)
	if err != nil {
		t.Fatalf("UpdateTicketText() error = %v", err)
	}

	expected := []string{
		ticket.FieldStatus + ": " + string(ticket.StatusCompleted),
		ticket.FieldUpdatedAt + ": " + string(completedAt),
	}
	if got := changedLines(withUnknownField, updated); !slices.Equal(got, expected) {
		t.Errorf("changed lines = %q, want %q", got, expected)
	}
	reparsed, err := ParseTicket(updated, fixturePath)
	if err != nil {
		t.Fatalf("ParseTicket() error = %v", err)
	}
	if got := reparsed.UnknownFields[unknownField]; got != "2d" {
		t.Errorf("%s = %v, want 2d", unknownField, got)
	}
}

func TestTicketFileReportsYAMLErrorsAndNamesTheFile(t *testing.T) {
	fixture := string(readFixture(t))
	unclosed := regexp.MustCompile(`(?m)^title: .*$`).ReplaceAllString(fixture, "title: [unclosed")

	tests := []struct {
		name string
		text string
	}{
		{"A duplicate key", fixture + "title: Second title\n"},
		{"An unclosed flow list", unclosed},
		{"A second document", fixture + "---\n" + fixture},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			changes := []backend.Change{backend.SetField{Field: ticket.FieldStatus, Value: ticket.StatusCompleted}}
			_, parseError := ParseTicket([]byte(tt.text), fixturePath)
			_, updateError := UpdateTicketText([]byte(tt.text), changes, fixturePath)

			for _, err := range []error{parseError, updateError} {
				if !errors.Is(err, backend.ErrInvalidYAML) || !strings.HasPrefix(err.Error(), fixturePath) {
					t.Errorf("error = %v, want %v naming %s", err, backend.ErrInvalidYAML, fixturePath)
				}
			}
		})
	}
}

func TestParseTicketRejectsInvalidFiles(t *testing.T) {
	fixture := string(readFixture(t))
	withoutPriority := regexp.MustCompile(`(?m)^priority: .*\n`).ReplaceAllString(fixture, "")
	fractionalPriority := regexp.MustCompile(`(?m)^priority: .*$`).ReplaceAllString(fixture, "priority: 1.5")
	badStatus := regexp.MustCompile(`(?m)^status: .*$`).ReplaceAllString(fixture, "status: done")

	tests := []struct {
		name string
		text string
	}{
		{"A missing required field", withoutPriority},
		{"A fractional priority", fractionalPriority},
		{"An unknown status", badStatus},
		{"An empty file", ""},
		{"A document that is not a mapping", "just text\n"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := ParseTicket([]byte(tt.text), fixturePath)

			if !errors.Is(err, backend.ErrInvalidFile) || !strings.HasPrefix(err.Error(), fixturePath) {
				t.Errorf("ParseTicket() error = %v, want %v naming %s", err, backend.ErrInvalidFile, fixturePath)
			}
		})
	}
}

func TestUpdateTicketTextTellsARejectedChangeFromAnInvalidFile(t *testing.T) {
	fixture := string(readFixture(t))
	badStatus := regexp.MustCompile(`(?m)^status: .*$`).ReplaceAllString(fixture, "status: done")

	tests := []struct {
		name     string
		text     string
		change   backend.SetField
		expected error
		other    error
		mention  string
	}{
		{"A change that makes a valid ticket invalid is rejected, naming the problem", fixture,
			backend.SetField{Field: ticket.FieldStatus, Value: ticket.Status("done")},
			backend.ErrInvalidChange, backend.ErrInvalidFile, "change rejected"},
		{"A file already invalid is reported as the invalid file", badStatus,
			backend.SetField{Field: ticket.FieldTitle, Value: "Renamed"},
			backend.ErrInvalidFile, backend.ErrInvalidChange, "is invalid"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := UpdateTicketText([]byte(tt.text), []backend.Change{tt.change}, fixturePath)

			var problems validation.Errors
			if !errors.Is(err, tt.expected) || errors.Is(err, tt.other) || !strings.HasPrefix(err.Error(), fixturePath) {
				t.Errorf("UpdateTicketText() error = %v, want %v and not %v, naming %s", err, tt.expected, tt.other, fixturePath)
			}
			if !strings.Contains(err.Error(), tt.mention) {
				t.Errorf("UpdateTicketText() error = %v, want it to say %q", err, tt.mention)
			}
			if !errors.As(err, &problems) || !slices.Equal(problems.Fields(), []string{ticket.FieldStatus}) {
				t.Errorf("UpdateTicketText() error = %v, want a problem with %s", err, ticket.FieldStatus)
			}
		})
	}
}

func TestParseTicketRequiresTheFieldsTheContractLeavesOptional(t *testing.T) {
	fixture := string(readFixture(t))

	tests := []struct {
		field       string
		replacement string
	}{
		{ticket.FieldPriority, ""},
		{ticket.FieldCreatedAt, ""},
		{ticket.FieldUpdatedAt, ""},
		{ticket.FieldPriority, ticket.FieldPriority + ": null\n"},
		{ticket.FieldCreatedAt, ticket.FieldCreatedAt + ": null\n"},
		{ticket.FieldUpdatedAt, ticket.FieldUpdatedAt + ": \"\"\n"},
	}

	for _, tt := range tests {
		t.Run(tt.field+" "+strconv.Quote(tt.replacement), func(t *testing.T) {
			text := regexp.MustCompile(`(?m)^`+tt.field+`: .*\n`).ReplaceAllString(fixture, tt.replacement)

			_, err := ParseTicket([]byte(text), fixturePath)

			var problems validation.Errors
			if !errors.Is(err, backend.ErrInvalidFile) || !errors.As(err, &problems) || !slices.Equal(problems.Fields(), []string{tt.field}) {
				t.Errorf("ParseTicket() error = %v, want %v naming %s", err, backend.ErrInvalidFile, tt.field)
			}
		})
	}
}
