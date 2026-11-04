package backend

import (
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/theroymind/go-tix/contracts/ticket"
)

func TestFindDirectory(t *testing.T) {
	tests := []struct {
		name          string
		setUp         func(t *testing.T, root string) (start, expected string)
		expectedError error
	}{
		{"Finds .tix in the start directory", func(t *testing.T, root string) (string, string) {
			tixDirectory := filepath.Join(root, ticket.DirectoryName)
			makeDirectory(t, tixDirectory)
			return root, tixDirectory
		}, nil},
		{"Walks up to the nearest ancestor with .tix", func(t *testing.T, root string) (string, string) {
			project := filepath.Join(root, "project")
			nested := filepath.Join(project, "src", "deep")
			makeDirectory(t, filepath.Join(root, ticket.DirectoryName))
			makeDirectory(t, filepath.Join(project, ticket.DirectoryName))
			makeDirectory(t, nested)
			return nested, filepath.Join(project, ticket.DirectoryName)
		}, nil},
		{"Ignores a .tix that is a file", func(t *testing.T, root string) (string, string) {
			nested := filepath.Join(root, "nested")
			makeDirectory(t, nested)
			writeFile(t, filepath.Join(nested, ticket.DirectoryName), nil)
			makeDirectory(t, filepath.Join(root, ticket.DirectoryName))
			return nested, filepath.Join(root, ticket.DirectoryName)
		}, nil},
		{"Fails when no ancestor has .tix", func(t *testing.T, root string) (string, string) {
			return root, ""
		}, ErrNotInRepository},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			start, expected := tt.setUp(t, t.TempDir())

			got, err := FindDirectory(start)

			if !errors.Is(err, tt.expectedError) {
				t.Fatalf("FindDirectory(%q) error = %v, want %v", start, err, tt.expectedError)
			}
			if got != expected {
				t.Errorf("FindDirectory(%q) = %q, want %q", start, got, expected)
			}
		})
	}
}

func makeDirectory(t *testing.T, path string) {
	t.Helper()
	if err := os.MkdirAll(path, DirectoryPermission); err != nil {
		t.Fatal(err)
	}
}

func writeFile(t *testing.T, path string, contents []byte) {
	t.Helper()
	if err := os.WriteFile(path, contents, FilePermission); err != nil {
		t.Fatal(err)
	}
}
