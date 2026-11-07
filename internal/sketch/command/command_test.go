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
	"github.com/theroymind/go-tix/internal/sketch/note"
)

const orbitNote = `---
feature: sketch-explorer
status: ready
---
Orbit the camera.

#fn useOrbit target Ref<Vec3>
-> ./camera.ts : reads the target
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
	exitCode := Run(arguments, strings.NewReader(""), &stdout, &stderr)
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
			"NOTE  TARGET  FEATURE  STATUS  LINKS  QUESTIONS\n", ""},
		{"A table of notes and targets", map[string]string{
			"src/useOrbit.ts.sketch.md": orbitNote,
			"a.go.sketch.md":            "just thinking\n? why\n-> b.go\n-> c.go\n",
		}, []string{"ls"}, cli.ExitCodeSuccess,
			"NOTE                       TARGET            FEATURE          STATUS  LINKS  QUESTIONS\n" +
				"a.go.sketch.md             +a.go             -                -       2      1\n" +
				"src/useOrbit.ts.sketch.md  +src/useOrbit.ts  sketch-explorer  ready   1      1\n", ""},
		{"Marks against a target that exists", map[string]string{
			"src/useOrbit.ts": "export function useOrbit() {}\n", "src/useOrbit.ts.sketch.md": "#fn useOrbit\n",
		}, []string{"ls"}, cli.ExitCodeSuccess,
			"NOTE                       TARGET            FEATURE  STATUS  LINKS  QUESTIONS\n" +
				"src/useOrbit.ts.sketch.md  ~src/useOrbit.ts  -        -       0      0\n", ""},
		{"No notes as JSON", map[string]string{}, []string{"ls", "--json"}, cli.ExitCodeSuccess, "[]\n", ""},
		{"A note as JSON", map[string]string{"src/useOrbit.ts.sketch.md": orbitNote, "src/useOrbit.ts": "useOrbitSpeed"},
			[]string{"ls", "--json"},
			cli.ExitCodeSuccess, `[
  {
    "schema_version": 2,
    "path": "src/useOrbit.ts.sketch.md",
    "target": "src/useOrbit.ts",
    "mark": "changed",
    "feature": "sketch-explorer",
    "status": "ready",
    "body": "Orbit the camera.\n\n#fn useOrbit target Ref<Vec3>\n-> ./camera.ts : reads the target\n? who owns the camera\n> the fight\n",
    "edges": [
      {
        "to": "src/camera.ts",
        "label": "reads the target"
      }
    ],
    "questions": [
      {
        "text": "who owns the camera",
        "answer": "the fight",
        "line": 9
      }
    ]
  }
]
`, ""},
		{"Edges marked against the repo and its notes", map[string]string{
			"a.go.sketch.md": "-> b.go\n-> c.go\n-> d.go\n", "b.go": "", "c.go.sketch.md": "",
		}, []string{"ls", "--json"}, cli.ExitCodeSuccess, `[
  {
    "schema_version": 2,
    "path": "a.go.sketch.md",
    "target": "a.go",
    "mark": "new",
    "body": "-> b.go\n-> c.go\n-> d.go\n",
    "edges": [
      {
        "to": "b.go",
        "mark": "changed"
      },
      {
        "to": "c.go",
        "mark": "new"
      },
      {
        "to": "d.go"
      }
    ]
  },
  {
    "schema_version": 2,
    "path": "c.go.sketch.md",
    "target": "c.go",
    "mark": "new",
    "body": ""
  }
]
`, ""},
		{"A bad note is named after the rest are listed", map[string]string{
			"a.go.sketch.md": "---\nfeature: [x\n---\n",
			"b.go.sketch.md": "? b\n",
		}, []string{"ls"}, cli.ExitCodeFailure,
			"NOTE            TARGET  FEATURE  STATUS  LINKS  QUESTIONS\n" +
				"b.go.sketch.md  +b.go   -        -       0      1\n", "a.go.sketch.md: bad front matter"},
		{"A note whose target is a folder is named", map[string]string{
			"web/.keep": "", "web.sketch.md": "a\n", "b.go.sketch.md": "",
		}, []string{"ls"}, cli.ExitCodeFailure,
			"NOTE            TARGET  FEATURE  STATUS  LINKS  QUESTIONS\n" +
				"b.go.sketch.md  +b.go   -        -       0      0\n", "web.sketch.md: plans a folder"},
		{"A note that doesn't validate is named", map[string]string{"c.go.sketch.md": "---\nstatus: done\n---\n"},
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
		"src/useOrbit.ts.sketch.md": orbitNote + "? how fast\n\n\n? what eases it\n",
		"a.go.sketch.md":            "? answered\n\n> yes\n> and more\n",
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
			"src/useOrbit.ts.sketch.md:11: how fast\n" +
				"src/useOrbit.ts.sketch.md:14: what eases it\n" +
				"web/app.tsx.sketch.md:1: which router\n", ""},
		{"Every question, with its answer below it", notes, []string{"questions", "--all"}, cli.ExitCodeSuccess,
			"a.go.sketch.md:1: answered\n" +
				"  > yes\n" +
				"  > and more\n" +
				"src/useOrbit.ts.sketch.md:9: who owns the camera\n" +
				"  > the fight\n" +
				"src/useOrbit.ts.sketch.md:11: how fast\n" +
				"src/useOrbit.ts.sketch.md:14: what eases it\n" +
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
		{"Every question as JSON", map[string]string{
			"a.go.sketch.md": "? answered\n> yes\n> and more\n? open\n",
		}, []string{"questions", "--json", "--all"}, cli.ExitCodeSuccess, `[
  {
    "path": "a.go.sketch.md",
    "line": 1,
    "text": "answered",
    "answer": "yes\nand more"
  },
  {
    "path": "a.go.sketch.md",
    "line": 4,
    "text": "open"
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
			"NOTE            TARGET  FEATURE                      STATUS  LINKS  QUESTIONS\n" +
				"a.go.sketch.md  +a.go   sketch-explorer              -       0      0\n" +
				"b.go.sketch.md  +b.go   sketch-explorr (not in tix)  -       0      0\n" +
				"c.go.sketch.md  +c.go   -                            -       0      0\n", "", "ls --all --json\n"},
		{"Inside a tix repo with no note naming a feature, tix isn't run", true, tickets,
			map[string]string{"c.go.sketch.md": ""}, []string{"ls"}, cli.ExitCodeSuccess,
			"NOTE            TARGET  FEATURE  STATUS  LINKS  QUESTIONS\n" +
				"c.go.sketch.md  +c.go   -        -       0      0\n", "", ""},
		{"Outside a tix repo, any feature is accepted without running tix", false, tickets, notes, []string{"ls"},
			cli.ExitCodeSuccess,
			"NOTE            TARGET  FEATURE          STATUS  LINKS  QUESTIONS\n" +
				"a.go.sketch.md  +a.go   sketch-explorer  -       0      0\n" +
				"b.go.sketch.md  +b.go   sketch-explorr   -       0      0\n" +
				"c.go.sketch.md  +c.go   -                -       0      0\n", "", ""},
		{"Inside a tix repo without tix on PATH, the notes list with a warning", true, "",
			map[string]string{"b.go.sketch.md": "---\nfeature: sketch-explorr\n---\n"}, []string{"ls", "--json"},
			cli.ExitCodeSuccess, `[
  {
    "schema_version": 2,
    "path": "b.go.sketch.md",
    "target": "b.go",
    "mark": "new",
    "feature": "sketch-explorr",
    "body": ""
  }
]
`, "sketch: warning: not checking features: this is a tix repo, but tix isn't on PATH", ""},
		{"Inside a tix repo where tix fails, the notes list with a warning", true, "not json", notes, []string{"ls"},
			cli.ExitCodeSuccess,
			"NOTE            TARGET  FEATURE          STATUS  LINKS  QUESTIONS\n" +
				"a.go.sketch.md  +a.go   sketch-explorer  -       0      0\n" +
				"b.go.sketch.md  +b.go   sketch-explorr   -       0      0\n" +
				"c.go.sketch.md  +c.go   -                -       0      0\n",
			"sketch: warning: not checking features: tix ls printed tickets sketch can't read", "ls --all --json\n"},
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
			if !strings.Contains(stderr, tt.stderrHas) || (tt.stderrHas == "" && stderr != "") {
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
		"web/app.tsx.sketch.md":     "---\nstatus: draft\n---\n-> ../src/useOrbit.ts\n? open\n> over\n> lines\n",
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

// TestUsageNamesEverything keeps the help text in step with the code: every subcommand and option, each line
// sketch reads, and each suggested marker.
func TestUsageNamesEverything(t *testing.T) {
	var words []string
	for name := range newProgram(nil).Commands {
		words = append(words, name)
	}
	words = append(words, "help", "--json", "--all", "--feature", sketch.NoteSuffix, "feature:", "status:", "-> PATH", "? QUESTION",
		"> ANSWER", "#fn", "#type", "#component", "#flow", "#test")
	for _, status := range sketch.Statuses {
		words = append(words, string(status))
	}
	for _, word := range words {
		t.Run(word, func(t *testing.T) {
			if !regexp.MustCompile(`(^|[\s,|\[])` + regexp.QuoteMeta(word) + `([\s,|\]]|$)`).MatchString(usage) {
				t.Errorf("usage doesn't name %q", word)
			}
		})
	}
}

// TestUsageExampleParses checks that the example note in the help text reads as the note it describes.
func TestUsageExampleParses(t *testing.T) {
	_, example, _ := strings.Cut(usage, "For example, src/useOrbit.ts.sketch.md:\n\n")
	example, _, _ = strings.Cut(example, "\n\nThe repo root")
	lines := strings.Split(example, "\n")
	for index, line := range lines {
		lines[index] = strings.TrimPrefix(line, "  ")
	}

	parsed, err := note.Parse("src/useOrbit.ts.sketch.md", strings.Join(lines, "\n"))

	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}
	if parsed.Feature != "camera" || parsed.Status != sketch.StatusDraft || len(parsed.Edges) != 1 ||
		len(parsed.Questions) != 1 || parsed.Questions[0].Answer == "" {
		t.Errorf("the example parses as %+v, want a draft camera note with one link and one answered question", parsed)
	}
}

func TestHelpPrintsUsage(t *testing.T) {
	exitCode, stdout, _ := inRepo(t, map[string]string{}, "help")

	if exitCode != cli.ExitCodeSuccess || stdout != usage {
		t.Errorf("sketch help = %d %q, want the usage on stdout", exitCode, stdout)
	}
}

// TestBareSketchOutsideATerminal checks that bare sketch, when stdout isn't a terminal, prints the usage, which
// describes the explorer, as a usage error instead of opening the explorer.
func TestBareSketchOutsideATerminal(t *testing.T) {
	var stdout, stderr strings.Builder

	exitCode := Run(nil, strings.NewReader(""), &stdout, &stderr)

	if exitCode != cli.ExitCodeUsage || stdout.String() != "" || stderr.String() != usage {
		t.Errorf("sketch = %d %q %q, want the usage on stderr and exit 2", exitCode, stdout.String(), stderr.String())
	}
	if !strings.Contains(usage, "explorer") {
		t.Errorf("usage doesn't mention the explorer")
	}
}
