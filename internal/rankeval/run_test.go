package rankeval

import (
	"errors"
	"path/filepath"
	"reflect"
	"slices"
	"testing"

	"github.com/theroymind/railyard/internal/cli"
)

func TestParseRunOptions(t *testing.T) {
	vaultPath, err := filepath.Abs("vault")
	if err != nil {
		t.Fatal(err)
	}
	tests := []struct {
		name          string
		arguments     []string
		expected      runOptions
		expectedError error
	}{
		{
			"Defaults around the vault",
			[]string{"--vault", "vault"},
			runOptions{vault: vaultPath, models: []string{ModelNone}, endpoint: DefaultEndpoint, depth: DefaultDepth, top: DefaultTop, runs: DefaultRuns, notes: DefaultNotesDirectory},
			nil,
		},
		{
			"Every option, models in the order given",
			[]string{"--vault", "vault", "--model", "haiku", "--model", "none", "--endpoint", "http://host:1/", "--depth", "0", "--top", "3", "--runs", "2", "--notes", "wiki", "--thinking"},
			runOptions{vault: vaultPath, models: []string{"haiku", "none"}, endpoint: "http://host:1/", depth: 0, top: 3, runs: 2, notes: "wiki", thinking: true},
			nil,
		},
		{"The vault is required", []string{"--model", "none"}, runOptions{}, cli.ErrRequiredOption},
		{"A pack of zero cards", []string{"--vault", "vault", "--top", "0"}, runOptions{}, ErrInvalidNumber},
		{"A depth that is not a number", []string{"--vault", "vault", "--depth", "two"}, runOptions{}, ErrInvalidNumber},
		{"Zero runs", []string{"--vault", "vault", "--runs", "0"}, runOptions{}, ErrInvalidNumber},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			options, err := parseRunOptions(tt.arguments)

			if !errors.Is(err, tt.expectedError) {
				t.Fatalf("parseRunOptions() error = %v, want %v", err, tt.expectedError)
			}
			if err != nil {
				return
			}
			if !slices.Equal(options.models, tt.expected.models) {
				t.Errorf("parseRunOptions() models = %q, want %q", options.models, tt.expected.models)
			}
			if !reflect.DeepEqual(options, tt.expected) {
				t.Errorf("parseRunOptions() = %+v, want %+v", options, tt.expected)
			}
		})
	}
}
