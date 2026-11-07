package refs

import (
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	cli "github.com/theroymind/railyard-cli"
	"github.com/theroymind/railyard-cli/strictjson"
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

func TestTasks(t *testing.T) {
	plain := t.TempDir()
	tixRepo := t.TempDir()
	if err := os.Mkdir(filepath.Join(tixRepo, ".tix"), 0o755); err != nil {
		t.Fatal(err)
	}
	children := `[{"id":"score-strikes","title":"Score strikes","type":"task","status":"open","parent":"fight-camera"},
{"id":"frame-kills","title":"Frame kills","type":"task","status":"completed","parent":"fight-camera"}]`
	tests := []struct {
		name     string
		root     string
		output   string
		runErr   error
		expected []string
		err      error
	}{
		{"Inside a tix repo, every child in tix's order", tixRepo, children, nil, []string{"score-strikes", "frame-kills"}, nil},
		{"No children", tixRepo, "[]\n", nil, []string{}, nil},
		{"Outside one, none", plain, "", nil, nil, nil},
		{"Without tix on PATH", tixRepo, "", &exec.Error{Name: "tix", Err: exec.ErrNotFound}, nil, ErrTixNotFound},
		{"A field tix doesn't write", tixRepo, `[{"id":"a","title":"A","type":"task","status":"open","owner":"x"}]`, nil, nil,
			ErrTixOutput},
		{"A ticket that doesn't validate", tixRepo, `[{"id":"A!","title":"A","type":"task","status":"open"}]`, nil, nil,
			ErrTixOutput},
		{"Trailing output", tixRepo, "[]\n[]", nil, nil, strictjson.ErrTrailingData},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var ran []string
			resolver := Resolver{Root: tt.root, Run: func(dir, name string, arguments []string, _ io.Reader) ([]byte, error) {
				ran = append([]string{name}, arguments...)
				return []byte(tt.output), tt.runErr
			}}

			actual, err := resolver.Tasks("fight-camera")

			if !errors.Is(err, tt.err) {
				t.Fatalf("Tasks() = %v, want %v", err, tt.err)
			}
			if fmt.Sprint(actual) != fmt.Sprint(tt.expected) {
				t.Errorf("Tasks() = %q, want %q", actual, tt.expected)
			}
			if tt.root == tixRepo && fmt.Sprint(ran) != fmt.Sprint([]string{"tix", "ls", "--parent", "fight-camera", "--all", "--json"}) {
				t.Errorf("ran %q, want tix ls --parent fight-camera --all --json", ran)
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
