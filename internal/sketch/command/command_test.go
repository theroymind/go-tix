package command

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	cli "github.com/theroymind/railyard-cli"
	"github.com/theroymind/railyard-cli/strictjson"

	"github.com/theroymind/go-tix/contracts/sketch"
)

const orbitNote = `---
feature: sketch-explorer
task: read-notes
---
Orbit the camera.

#fn useOrbit target Ref<Vec3>
? who owns the camera
> the fight
`

// inRepo writes files into a fresh repo, runs sketch with arguments from a subdirectory of it, so the root has to
// be found above it, and returns the exit code, stdout and stderr.
func inRepo(t *testing.T, files map[string]string, arguments ...string) (cli.ExitCode, string, string) {
	t.Helper()
	root := t.TempDir()
	files[".git/HEAD"] = ""
	files["client/.keep"] = ""
	for file, text := range files {
		path := filepath.Join(root, filepath.FromSlash(file))
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte(text), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	t.Chdir(filepath.Join(root, "client"))
	var stdout, stderr strings.Builder
	exitCode := Program.Run(arguments, strings.NewReader(""), &stdout, &stderr)
	return exitCode, stdout.String(), stderr.String()
}

func TestList(t *testing.T) {
	tests := []struct {
		name      string
		files     map[string]string
		arguments []string
		exitCode  cli.ExitCode
		stdout    string
		stderrHas string
	}{
		{"No notes", map[string]string{"main.go": ""}, []string{"ls"}, cli.ExitCodeSuccess,
			"NOTE  TARGET  FEATURE  TASK  ITEMS  QUESTIONS\n", ""},
		{"A table of notes and targets", map[string]string{
			"src/useOrbit.ts.sketch.md": orbitNote,
			"a.go.sketch.md":            "just thinking\n? why\n",
		}, []string{"ls"}, cli.ExitCodeSuccess,
			"NOTE                       TARGET           FEATURE          TASK        ITEMS  QUESTIONS\n" +
				"a.go.sketch.md             a.go             -                -           0      1\n" +
				"src/useOrbit.ts.sketch.md  src/useOrbit.ts  sketch-explorer  read-notes  1      1\n", ""},
		{"No notes as JSON", map[string]string{}, []string{"ls", "--json"}, cli.ExitCodeSuccess, "[]\n", ""},
		{"A note as JSON", map[string]string{"src/useOrbit.ts.sketch.md": orbitNote}, []string{"ls", "--json"},
			cli.ExitCodeSuccess, `[
  {
    "schema_version": 1,
    "path": "src/useOrbit.ts.sketch.md",
    "target": "src/useOrbit.ts",
    "feature": "sketch-explorer",
    "task": "read-notes",
    "items": [
      {
        "type": "fn",
        "name": "useOrbit",
        "text": "target Ref<Vec3>"
      }
    ],
    "questions": [
      {
        "text": "who owns the camera",
        "answer": "the fight",
        "line": 8
      }
    ]
  }
]
`, ""},
		{"A bad note is named after the rest are listed", map[string]string{
			"a.go.sketch.md": "---\nfeature: [x\n---\n",
			"b.go.sketch.md": "#fn b\n",
		}, []string{"ls"}, cli.ExitCodeFailure,
			"NOTE            TARGET  FEATURE  TASK  ITEMS  QUESTIONS\n" +
				"b.go.sketch.md  b.go    -        -     1      0\n", "a.go.sketch.md: bad front matter"},
		{"A note that doesn't validate is named", map[string]string{"c.go.sketch.md": "#fn c\x1bd\n"},
			[]string{"ls", "--json"}, cli.ExitCodeFailure, "[]\n", "c.go.sketch.md: "},
		{"An unknown option", map[string]string{}, []string{"ls", "--md"}, cli.ExitCodeUsage, "", "md"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			exitCode, stdout, stderr := inRepo(t, tt.files, tt.arguments...)

			if exitCode != tt.exitCode {
				t.Fatalf("sketch %q exit = %d, want %d (stderr %q)", tt.arguments, exitCode, tt.exitCode, stderr)
			}
			if stdout != tt.stdout {
				t.Errorf("sketch %q stdout =\n%s\nwant\n%s", tt.arguments, stdout, tt.stdout)
			}
			if !strings.Contains(stderr, tt.stderrHas) {
				t.Errorf("sketch %q stderr = %q, want it to say %q", tt.arguments, stderr, tt.stderrHas)
			}
		})
	}
}

// TestListJSONDecodes reads ls --json back the way a program would, through strictjson and Validate.
func TestListJSONDecodes(t *testing.T) {
	_, stdout, _ := inRepo(t, map[string]string{
		"src/useOrbit.ts.sketch.md": orbitNote,
		"web/app.tsx.sketch.md":     "#component App\n#hook useApp\n? open\n",
	}, "ls", "--json")

	var notes []sketch.Note
	if err := strictjson.Unmarshal([]byte(stdout), &notes); err != nil {
		t.Fatalf("ls --json doesn't decode: %v\n%s", err, stdout)
	}
	if len(notes) != 2 {
		t.Fatalf("ls --json gave %d notes, want 2", len(notes))
	}
	for _, value := range notes {
		if err := value.Validate(); err != nil {
			t.Errorf("note %s doesn't validate: %v", value.Path, err)
		}
	}
}

func TestRepoRoot(t *testing.T) {
	root := t.TempDir()
	nested := filepath.Join(root, "a", "b")
	if err := os.MkdirAll(nested, 0o755); err != nil {
		t.Fatal(err)
	}
	marked := t.TempDir()
	if err := os.Mkdir(filepath.Join(marked, ".tix"), 0o755); err != nil {
		t.Fatal(err)
	}
	markedNested := filepath.Join(marked, "x")
	if err := os.Mkdir(markedNested, 0o755); err != nil {
		t.Fatal(err)
	}
	tests := []struct {
		name     string
		dir      string
		expected string
	}{
		{"A .tix above", markedNested, marked},
		{"The marked directory itself", marked, marked},
		{"No marker anywhere falls back to the directory", nested, nested},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if actual := repoRoot(tt.dir); actual != tt.expected {
				t.Errorf("repoRoot() = %q, want %q", actual, tt.expected)
			}
		})
	}
}

// TestUsageNamesEverything keeps the help text in step with the code: every subcommand, every item type, and
// each note marker.
func TestUsageNamesEverything(t *testing.T) {
	var words []string
	for name := range Program.Commands {
		words = append(words, name)
	}
	words = append(words, "help", sketch.NoteSuffix, "feature:", "task:", "#TYPE", "? QUESTION", "> ANSWER")
	for _, itemType := range sketch.ItemTypes {
		words = append(words, string(itemType))
	}
	for _, word := range words {
		t.Run(word, func(t *testing.T) {
			if !regexp.MustCompile(`(^|[\s,])` + regexp.QuoteMeta(word) + `([\s,]|$)`).MatchString(usage) {
				t.Errorf("usage doesn't name %q", word)
			}
		})
	}
}

func TestHelpPrintsUsage(t *testing.T) {
	exitCode, stdout, _ := inRepo(t, map[string]string{}, "help")

	if exitCode != cli.ExitCodeSuccess || stdout != usage {
		t.Errorf("sketch help = %d %q, want the usage on stdout", exitCode, stdout)
	}
}
