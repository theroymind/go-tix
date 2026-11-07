package refs

import (
	"errors"
	"fmt"
	"io"
	"maps"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	cli "github.com/theroymind/railyard-cli"
	"github.com/theroymind/railyard-cli/strictjson"
)

func TestTickets(t *testing.T) {
	tickets := `[{"id":"sketch-explorer","title":"Sketch explorer","type":"epic","status":"open"},
{"id":"read-notes","title":"Read notes","type":"task","status":"completed","parent":"sketch-explorer"}]`
	tests := []struct {
		name     string
		output   string
		runErr   error
		expected map[string]bool
		err      error
	}{
		{"Every ticket, done ones included", tickets, nil, map[string]bool{"sketch-explorer": true, "read-notes": true},
			nil},
		{"No tickets", "[]\n", nil, map[string]bool{}, nil},
		{"Without tix on PATH", "", &exec.Error{Name: "tix", Err: exec.ErrNotFound}, nil, ErrTixNotFound},
		{"tix fails", "", cli.ErrChildFailed, nil, cli.ErrChildFailed},
		{"A ticket that doesn't validate", `[{"id":"A!","title":"A","type":"task","status":"open"}]`, nil, nil,
			ErrTixOutput},
		{"A field tix doesn't write", `[{"id":"a","title":"A","type":"task","status":"open","owner":"x"}]`, nil, nil,
			ErrTixOutput},
		{"Trailing output", "[]\n[]", nil, nil, strictjson.ErrTrailingData},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var ran []string
			resolver := Resolver{Root: t.TempDir(), Run: func(dir, name string, arguments []string, _ io.Reader) ([]byte, error) {
				ran = append([]string{name}, arguments...)
				return []byte(tt.output), tt.runErr
			}}

			actual, err := resolver.Tickets()

			if !errors.Is(err, tt.err) {
				t.Fatalf("Tickets() = %v, want %v", err, tt.err)
			}
			if !maps.Equal(actual, tt.expected) {
				t.Errorf("Tickets() = %v, want %v", actual, tt.expected)
			}
			if !slices.Equal(ran, []string{"tix", "ls", "--all", "--json"}) {
				t.Errorf("ran %q, want tix ls --all --json", ran)
			}
		})
	}
}

func TestFeatures(t *testing.T) {
	tixRepo := t.TempDir()
	if err := os.Mkdir(filepath.Join(tixRepo, ".tix"), 0o755); err != nil {
		t.Fatal(err)
	}
	tickets := `[{"id":"sketch-explorer","title":"Sketch explorer","type":"epic","status":"open"},
{"id":"read-notes","title":"Read notes","type":"task","status":"completed","parent":"sketch-explorer"},
{"id":"fight-camera","title":"Fight camera","type":"epic","status":"completed"}]`
	tests := []struct {
		name     string
		root     string
		output   string
		runErr   error
		expected []string
		err      error
	}{
		{"Every epic, done ones included, in tix's order", tixRepo, tickets, nil,
			[]string{"sketch-explorer", "fight-camera"}, nil},
		{"Outside a tix repo, none", t.TempDir(), "", nil, nil, nil},
		{"tix fails", tixRepo, "", cli.ErrChildFailed, nil, cli.ErrChildFailed},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var ran []string
			resolver := Resolver{Root: tt.root, Run: func(dir, name string, arguments []string, _ io.Reader) ([]byte, error) {
				ran = append([]string{name}, arguments...)
				return []byte(tt.output), tt.runErr
			}}

			actual, err := resolver.Features()

			if !errors.Is(err, tt.err) {
				t.Fatalf("Features() = %v, want %v", err, tt.err)
			}
			if !slices.Equal(actual, tt.expected) {
				t.Errorf("Features() = %q, want %q", actual, tt.expected)
			}
			if tt.root == tixRepo && !slices.Equal(ran, []string{"tix", "ls", "--all", "--json"}) {
				t.Errorf("ran %q, want tix ls --all --json", ran)
			}
		})
	}
}

func TestRunQuiet(t *testing.T) {
	tests := []struct {
		name   string
		script string
		stdout string
		fails  bool
		cause  string
	}{
		{"Success returns stdout", "echo out", "out\n", false, ""},
		{"Failure ends the error with the last stderr line", "echo out; echo first >&2; echo last >&2; exit 3", "out\n",
			true, "last"},
		{"Failure with no stderr still fails", "exit 3", "", true, ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			terminal, err := os.CreateTemp(t.TempDir(), "stderr")
			if err != nil {
				t.Fatal(err)
			}
			saved := os.Stderr
			os.Stderr = terminal
			defer func() { os.Stderr = saved }()

			output, err := RunQuiet(t.TempDir(), "sh", []string{"-c", tt.script}, nil)

			if string(output) != tt.stdout {
				t.Errorf("stdout = %q, want %q", output, tt.stdout)
			}
			if tt.fails != errors.Is(err, cli.ErrChildFailed) {
				t.Errorf("error = %v, want ErrChildFailed: %v", err, tt.fails)
			}
			var exit *exec.ExitError
			if tt.fails && !errors.As(err, &exit) {
				t.Errorf("error = %v, want it to wrap the exit error", err)
			}
			if tt.cause != "" && !strings.HasSuffix(err.Error(), ": "+tt.cause) {
				t.Errorf("error = %q, want it to end with %q", err, tt.cause)
			}
			if leaked, _ := os.ReadFile(terminal.Name()); len(leaked) > 0 {
				t.Errorf("stderr reached the terminal: %q", leaked)
			}
		})
	}
}

func TestTicket(t *testing.T) {
	tests := []struct {
		name     string
		output   string
		runErr   error
		expected string
		err      error
	}{
		{"The ticket with its criteria",
			`{"id":"camera","title":"Camera","type":"epic","status":"open","acceptance":[{"text":"It orbits.","done":true}]}`,
			nil, "Camera [{It orbits. true}]", nil},
		{"Without tix on PATH", "", &exec.Error{Name: "tix", Err: exec.ErrNotFound}, "", ErrTixNotFound},
		{"tix fails, as for an unknown ticket", "", cli.ErrChildFailed, "", cli.ErrChildFailed},
		{"A field tix doesn't write", `{"id":"camera","title":"Camera","type":"epic","status":"open","owner":"x"}`,
			nil, "", ErrTixOutput},
		{"A ticket that doesn't validate", `{"id":"A!","title":"A","type":"epic","status":"open"}`, nil, "",
			ErrTixOutput},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var ran []string
			resolver := Resolver{Root: t.TempDir(), Run: func(dir, name string, arguments []string, _ io.Reader) ([]byte, error) {
				ran = append([]string{name}, arguments...)
				return []byte(tt.output), tt.runErr
			}}

			actual, err := resolver.Ticket("camera")

			if !errors.Is(err, tt.err) {
				t.Fatalf("Ticket() = %v, want %v", err, tt.err)
			}
			if tt.err == nil && fmt.Sprint(actual.Title, " ", actual.Acceptance) != tt.expected {
				t.Errorf("Ticket() = %v %v, want %s", actual.Title, actual.Acceptance, tt.expected)
			}
			if !slices.Equal(ran, []string{"tix", "show", "camera", "--json"}) {
				t.Errorf("ran %q, want tix show camera --json", ran)
			}
		})
	}
}
