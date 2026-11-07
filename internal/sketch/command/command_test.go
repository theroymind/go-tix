package command

import (
	"maps"
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
			"NOTE                       TARGET            FEATURE          TASK        ITEMS      QUESTIONS\n" +
				"a.go.sketch.md             +a.go             -                -           -          1\n" +
				"src/useOrbit.ts.sketch.md  +src/useOrbit.ts  sketch-explorer  read-notes  +useOrbit  1\n", ""},
		{"Marks against targets that exist", map[string]string{
			"src/useOrbit.ts":           "export function useOrbit() {}\nconst useOrbitSpeed = 1\n",
			"src/useOrbit.ts.sketch.md": "#fn useOrbit\n#fn useOrbitSpeed\n#tune Speed\n#type Orbit\n",
		}, []string{"ls"}, cli.ExitCodeSuccess,
			"NOTE                       TARGET            FEATURE  TASK  ITEMS                                      QUESTIONS\n" +
				"src/useOrbit.ts.sketch.md  ~src/useOrbit.ts  -        -     ~useOrbit, ~useOrbitSpeed, +Speed, +Orbit  0\n", ""},
		{"No notes as JSON", map[string]string{}, []string{"ls", "--json"}, cli.ExitCodeSuccess, "[]\n", ""},
		{"A note as JSON", map[string]string{"src/useOrbit.ts.sketch.md": orbitNote, "src/useOrbit.ts": "useOrbitSpeed"},
			[]string{"ls", "--json"},
			cli.ExitCodeSuccess, `[
  {
    "schema_version": 1,
    "path": "src/useOrbit.ts.sketch.md",
    "target": "src/useOrbit.ts",
    "mark": "changed",
    "feature": "sketch-explorer",
    "task": "read-notes",
    "items": [
      {
        "type": "fn",
        "name": "useOrbit",
        "text": "target Ref<Vec3>",
        "mark": "new"
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
				"b.go.sketch.md  +b.go   -        -     +b     0\n", "a.go.sketch.md: bad front matter"},
		{"A note whose target is a folder is named", map[string]string{
			"web/.keep": "", "web.sketch.md": "#fn a\n", "b.go.sketch.md": "",
		}, []string{"ls"}, cli.ExitCodeFailure,
			"NOTE            TARGET  FEATURE  TASK  ITEMS  QUESTIONS\n" +
				"b.go.sketch.md  +b.go   -        -     -      0\n", "web.sketch.md: "},
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

func TestQuestions(t *testing.T) {
	notes := map[string]string{
		"web/app.tsx.sketch.md":     "? which router\n",
		"src/useOrbit.ts.sketch.md": orbitNote + "? how fast\n\n? what eases it\n",
		"a.go.sketch.md":            "? answered\n> yes\n",
	}
	tests := []struct {
		name      string
		files     map[string]string
		arguments []string
		exitCode  cli.ExitCode
		stdout    string
		stderrHas string
	}{
		{"No notes", map[string]string{}, []string{"questions"}, cli.ExitCodeSuccess, "", ""},
		{"Open questions by note, then line", notes, []string{"questions"}, cli.ExitCodeSuccess,
			"src/useOrbit.ts.sketch.md:10: how fast\n" +
				"src/useOrbit.ts.sketch.md:12: what eases it\n" +
				"web/app.tsx.sketch.md:1: which router\n", ""},
		{"No open questions as JSON", map[string]string{"a.go.sketch.md": "? answered\n> yes\n"},
			[]string{"questions", "--json"}, cli.ExitCodeSuccess, "[]\n", ""},
		{"Open questions as JSON", map[string]string{"web/app.tsx.sketch.md": "# App\n? which router\n"},
			[]string{"questions", "--json"}, cli.ExitCodeSuccess, `[
  {
    "path": "web/app.tsx.sketch.md",
    "line": 2,
    "text": "which router"
  }
]
`, ""},
		{"A bad note is named after the rest are listed", map[string]string{
			"a.go.sketch.md": "---\nfeature: [x\n---\n? hidden\n",
			"b.go.sketch.md": "? shown\n",
		}, []string{"questions"}, cli.ExitCodeFailure, "b.go.sketch.md:1: shown\n", "a.go.sketch.md: bad front matter"},
		{"An unknown option", map[string]string{}, []string{"questions", "--md"}, cli.ExitCodeUsage, "", "md"},
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

// fakeTix puts a tix on a fresh PATH that prints tickets and logs each run's arguments, one line per run, to the
// file it returns. With tickets empty, PATH holds no tix at all.
func fakeTix(t *testing.T, tickets string) string {
	t.Helper()
	bin := t.TempDir()
	log := filepath.Join(bin, "runs.log")
	t.Setenv("PATH", bin)
	if tickets == "" {
		return log
	}
	script := "#!/bin/sh\necho \"$@\" >> '" + log + "'\nprintf '%s\\n' '" + tickets + "'\n"
	if err := os.WriteFile(filepath.Join(bin, "tix"), []byte(script), 0o755); err != nil {
		t.Fatal(err)
	}
	return log
}

func TestListResolvesFeatures(t *testing.T) {
	tickets := `[{"id":"sketch-explorer","title":"Sketch explorer","type":"epic","status":"open"}]`
	notes := map[string]string{
		"a.go.sketch.md": "---\nfeature: sketch-explorer\n---\n",
		"b.go.sketch.md": "---\nfeature: sketch-explorr\n---\n",
		"c.go.sketch.md": "",
	}
	tests := []struct {
		name      string
		tixRepo   bool
		tickets   string
		files     map[string]string
		arguments []string
		exitCode  cli.ExitCode
		stdout    string
		stderrHas string
		runs      string
	}{
		{"Inside a tix repo, an unknown feature is flagged and the rest list", true, tickets, notes, []string{"ls"},
			cli.ExitCodeSuccess,
			"NOTE            TARGET  FEATURE                      TASK  ITEMS  QUESTIONS\n" +
				"a.go.sketch.md  +a.go   sketch-explorer              -     -      0\n" +
				"b.go.sketch.md  +b.go   sketch-explorr (not in tix)  -     -      0\n" +
				"c.go.sketch.md  +c.go   -                            -     -      0\n", "", "ls --all --json\n"},
		{"Flagged as JSON", true, tickets, map[string]string{"b.go.sketch.md": "---\nfeature: sketch-explorr\n---\n"},
			[]string{"ls", "--json"}, cli.ExitCodeSuccess, `[
  {
    "schema_version": 1,
    "path": "b.go.sketch.md",
    "target": "b.go",
    "mark": "new",
    "feature": "sketch-explorr",
    "unknown_feature": true
  }
]
`, "", "ls --all --json\n"},
		{"Inside a tix repo with no note naming a feature, tix isn't run", true, tickets,
			map[string]string{"c.go.sketch.md": ""}, []string{"ls"}, cli.ExitCodeSuccess,
			"NOTE            TARGET  FEATURE  TASK  ITEMS  QUESTIONS\n" +
				"c.go.sketch.md  +c.go   -        -     -      0\n", "", ""},
		{"Outside a tix repo, any feature is accepted without running tix", false, tickets, notes, []string{"ls"},
			cli.ExitCodeSuccess,
			"NOTE            TARGET  FEATURE          TASK  ITEMS  QUESTIONS\n" +
				"a.go.sketch.md  +a.go   sketch-explorer  -     -      0\n" +
				"b.go.sketch.md  +b.go   sketch-explorr   -     -      0\n" +
				"c.go.sketch.md  +c.go   -                -     -      0\n", "", ""},
		{"Inside a tix repo without tix on PATH", true, "", notes, []string{"ls"}, cli.ExitCodeFailure, "",
			"tix isn't on PATH", ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			log := fakeTix(t, tt.tickets)
			files := maps.Clone(tt.files)
			if tt.tixRepo {
				files[".tix/config.yml"] = ""
			}

			exitCode, stdout, stderr := inRepo(t, files, tt.arguments...)

			if exitCode != tt.exitCode {
				t.Fatalf("sketch %q exit = %d, want %d (stderr %q)", tt.arguments, exitCode, tt.exitCode, stderr)
			}
			if stdout != tt.stdout {
				t.Errorf("sketch %q stdout =\n%s\nwant\n%s", tt.arguments, stdout, tt.stdout)
			}
			if !strings.Contains(stderr, tt.stderrHas) {
				t.Errorf("sketch %q stderr = %q, want it to say %q", tt.arguments, stderr, tt.stderrHas)
			}
			if runs, _ := os.ReadFile(log); string(runs) != tt.runs {
				t.Errorf("tix ran with %q, want %q", runs, tt.runs)
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
