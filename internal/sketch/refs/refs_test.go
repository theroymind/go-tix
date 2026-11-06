package refs

import (
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	cli "github.com/theroymind/railyard-cli"
)

var errNoMatch = errors.New("no ticket matches")

func TestFeatureDir(t *testing.T) {
	plain := t.TempDir()
	tixRepo := t.TempDir()
	if err := os.Mkdir(filepath.Join(tixRepo, ".tix"), 0o755); err != nil {
		t.Fatal(err)
	}
	note := filepath.Join(tixRepo, "docs", "phases", "fight-camera", "fight-camera.md")
	tests := []struct {
		name     string
		root     string
		output   string
		runErr   error
		expected string
		err      error
	}{
		{"Outside a tix repo, any name works", plain, "", nil, filepath.Join(plain, "docs", "features", "fight-camera"), nil},
		{"Inside one, the note's directory", tixRepo, note + "\n", nil, filepath.Dir(note), nil},
		{"Inside one, a feature that isn't a ticket", tixRepo, "", fmt.Errorf("%w: %w", cli.ErrChildFailed, errNoMatch), "",
			ErrNotATicket},
		{"Inside one, without tix on PATH", tixRepo, "", fmt.Errorf("%w: tix: %w", cli.ErrChildFailed,
			&exec.Error{Name: "tix", Err: exec.ErrNotFound}), "", ErrTixNotFound},
		{"Inside one, a path that isn't absolute", tixRepo, "docs/fight-camera.md\n", nil, "", ErrRelativePath},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var ran []string
			resolver := Resolver{Root: tt.root, Run: func(dir, name string, arguments []string, _ io.Reader) ([]byte, error) {
				ran = append([]string{dir, name}, arguments...)
				return []byte(tt.output), tt.runErr
			}}

			actual, err := resolver.FeatureDir("fight-camera")

			if !errors.Is(err, tt.err) || (errors.Is(tt.err, ErrNotATicket) && !errors.Is(err, errNoMatch)) {
				t.Fatalf("FeatureDir() = %v, want %v keeping tix's cause", err, tt.err)
			}
			if actual != tt.expected {
				t.Errorf("FeatureDir() = %q, want %q", actual, tt.expected)
			}
			if tt.root == tixRepo && fmt.Sprint(ran) != fmt.Sprint([]string{tixRepo, "tix", "path", "fight-camera"}) {
				t.Errorf("ran %q, want tix path fight-camera in the root", ran)
			}
			if tt.root == plain && ran != nil {
				t.Errorf("ran %q outside a tix repo, want nothing run", ran)
			}
		})
	}
}
