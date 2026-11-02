package backend_test

import (
	"errors"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"

	"github.com/theroymind/railyard/internal/contracts/ticket"
	"github.com/theroymind/railyard/internal/tix/backend"
	"github.com/theroymind/railyard/internal/tix/backend/yamlstore"
)

func TestOpen(t *testing.T) {
	backend.Register(backend.KindYAML, yamlstore.Open)

	tests := []struct {
		name          string
		config        string
		expectedError error
		mentions      []string
	}{
		{"An absent backend opens the YAML store", "prefix: ry\n", nil, nil},
		{"backend: yaml opens the YAML store", "prefix: ry\nbackend: yaml\n", nil, nil},
		{"An unknown backend names the allowed ones", "prefix: ry\nbackend: nonsense\n", backend.ErrInvalidFile,
			[]string{"backend", strconv.Quote(string(backend.KindYAML)), strconv.Quote(string(backend.KindVault))}},
		{"The vault backend needs a profile", "prefix: ry\nbackend: vault\n", backend.ErrInvalidFile,
			[]string{"vault", "profile"}},
		{"An unregistered backend is not implemented",
			"prefix: ry\nbackend: vault\nvault:\n  tickets:\n    task: {ticket_type: task, statuses: {todo: open}}\n",
			backend.ErrNotImplemented, []string{string(backend.KindVault)}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			root := t.TempDir()
			tixDirectory := filepath.Join(root, ticket.DirectoryName)
			if err := os.Mkdir(tixDirectory, backend.DirectoryPermission); err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(backend.ConfigPath(tixDirectory), []byte(tt.config), backend.FilePermission); err != nil {
				t.Fatal(err)
			}

			opened, err := backend.Open(root)

			if !errors.Is(err, tt.expectedError) {
				t.Fatalf("Open() error = %v, want %v", err, tt.expectedError)
			}
			for _, mention := range tt.mentions {
				if !strings.Contains(err.Error(), mention) {
					t.Errorf("Open() error = %q, want it to mention %s", err, mention)
				}
			}
			if _, isYAML := opened.(yamlstore.Store); tt.expectedError == nil && !isYAML {
				t.Errorf("Open() = %T, want yamlstore.Store", opened)
			}
		})
	}
}

func TestWriteConfigLeavesAnAbsentBackendOut(t *testing.T) {
	tests := []struct {
		name     string
		config   backend.Config
		expected string
	}{
		{"No backend writes only the prefix, as tix v1 does", backend.Config{Prefix: "ry"}, "prefix: ry\n"},
		{"A backend is written after the prefix", backend.Config{Prefix: "ry", Backend: backend.KindYAML}, "prefix: ry\nbackend: yaml\n"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			directory := t.TempDir()

			if err := backend.WriteConfig(directory, tt.config); err != nil {
				t.Fatalf("WriteConfig() error = %v", err)
			}

			if got, err := os.ReadFile(backend.ConfigPath(directory)); err != nil || string(got) != tt.expected {
				t.Errorf("config.yml = %q, %v, want %q", got, err, tt.expected)
			}
		})
	}
}
