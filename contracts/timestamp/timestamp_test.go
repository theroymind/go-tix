package timestamp

import (
	"strings"
	"testing"
	"time"
)

const (
	fractionSeparator = "."
	utcDesignator     = "Z"
)

func TestFormat(t *testing.T) {
	offsets := []*time.Location{
		time.Local,
		time.UTC,
		time.FixedZone("behind", int((-4 * time.Hour).Seconds())),
		time.FixedZone("ahead with minutes", int((5*time.Hour + 30*time.Minute).Seconds())),
	}
	moments := []time.Time{
		time.Date(2026, time.October, 14, 21, 0, 0, 999_000_000, time.UTC),
		time.Date(2026, time.January, 1, 0, 0, 0, 0, time.UTC),
		time.Date(2026, time.July, 4, 9, 5, 7, 1_000_000, time.UTC),
	}

	for _, location := range offsets {
		for _, moment := range moments {
			moment := moment.In(location)
			t.Run(location.String()+" "+moment.String(), func(t *testing.T) {
				formatted := Format(moment)

				if !formatted.IsValid() {
					t.Errorf("Format(%v) = %q, which is not a valid timestamp", moment, formatted)
				}
				if strings.Contains(string(formatted), fractionSeparator) {
					t.Errorf("Format(%v) = %q, want whole seconds", moment, formatted)
				}
				parsed, err := time.Parse(time.RFC3339, string(formatted))
				if err != nil {
					t.Fatalf("time.Parse(%q) error = %v", formatted, err)
				}
				if want := moment.Truncate(time.Second); !parsed.Equal(want) {
					t.Errorf("Format(%v) parses back to %v, want %v", moment, parsed, want)
				}
				if !strings.HasSuffix(string(formatted), utcDesignator) {
					t.Errorf("Format(%v) = %q, want UTC", moment, formatted)
				}
			})
		}
	}
}

func TestIsValid(t *testing.T) {
	tests := []struct {
		name     string
		input    Timestamp
		expected bool
	}{
		{"Offset timestamp is valid", "2026-10-25T09:00:00-04:00", true},
		{"UTC timestamp is valid", "2026-10-25T09:00:00Z", true},
		{"Fractional seconds are valid", "2026-10-25T09:00:00.5+00:00", true},
		{"Leap day is valid", "2028-02-29T09:00:00Z", true},
		{"Missing offset is invalid", "2026-10-25T09:00:00", false},
		{"Missing seconds are invalid", "2026-10-25T09:00-04:00", false},
		{"Leap day outside a leap year is invalid", "2026-02-29T09:00:00Z", false},
		{"Hour past 23 is invalid", "2026-10-25T24:00:00Z", false},
		{"Offset past 23 hours is invalid", "2026-10-25T09:00:00+24:00", false},
		{"Empty string is invalid", "", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.input.IsValid(); got != tt.expected {
				t.Errorf("Timestamp(%q).IsValid() = %v, want %v", tt.input, got, tt.expected)
			}
		})
	}
}
