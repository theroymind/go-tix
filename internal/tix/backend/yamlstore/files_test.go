package yamlstore

import (
	"os"
	"testing"

	"github.com/theroymind/go-tix/internal/tix/backend"
)

func makeDirectory(t *testing.T, path string) {
	t.Helper()
	if err := os.MkdirAll(path, backend.DirectoryPermission); err != nil {
		t.Fatal(err)
	}
}

func writeFile(t *testing.T, path string, contents []byte) {
	t.Helper()
	if err := os.WriteFile(path, contents, backend.FilePermission); err != nil {
		t.Fatal(err)
	}
}

func readFile(t *testing.T, path string) []byte {
	t.Helper()
	contents, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	return contents
}
