package command

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	cli "github.com/theroymind/railyard-cli"

	"github.com/theroymind/go-tix/internal/sketch/grammar"
)

const (
	atFile = "fight-camera/score-strikes/client/camera.odin"
	atItem = atFile + ":score_strike"
)

type run struct {
	arguments []string
	exitCode  cli.ExitCode
	stdout    string
	stderrHas string
}

// inRepo runs each command in order from a subdirectory of a fresh repo, so the root has to be found above it.
func inRepo(t *testing.T, runs []run) string {
	t.Helper()
	root := t.TempDir()
	if err := os.Mkdir(filepath.Join(root, ".git"), 0o755); err != nil {
		t.Fatal(err)
	}
	subdir := filepath.Join(root, "client")
	if err := os.Mkdir(subdir, 0o755); err != nil {
		t.Fatal(err)
	}
	t.Chdir(subdir)
	for _, step := range runs {
		var stdout, stderr strings.Builder
		exitCode := Program.Run(step.arguments, strings.NewReader(""), &stdout, &stderr)
		if exitCode != step.exitCode {
			t.Fatalf("sketch %q exit = %d, want %d (stderr %q)", step.arguments, exitCode, step.exitCode, stderr.String())
		}
		if step.stdout != "" && stdout.String() != step.stdout {
			t.Errorf("sketch %q stdout =\n%s\nwant\n%s", step.arguments, stdout.String(), step.stdout)
		}
		if !strings.Contains(stderr.String(), step.stderrHas) {
			t.Errorf("sketch %q stderr = %q, want it to say %q", step.arguments, stderr.String(), step.stderrHas)
		}
	}
	return root
}

func TestOneShotEditsAndShow(t *testing.T) {
	root := inRepo(t, []run{
		{[]string{"-C", atFile, "fn", "score_strike"}, cli.ExitCodeSuccess, atItem + "\n", ""},
		{[]string{"-C", atItem, "in", "actor", "Combatant_Id"}, cli.ExitCodeSuccess, atItem + "\n", ""},
		{[]string{"-C", atItem, "a", "who anchors a kill"}, cli.ExitCodeSuccess, "", ""},
		{[]string{"-C", atItem, "o", "Camera_Move"}, cli.ExitCodeSuccess, "", ""},
		{[]string{"-C", atItem, "set", "3", "Camera_Cut"}, cli.ExitCodeSuccess, "", ""},
		{[]string{"-C", atItem, "rm", "1"}, cli.ExitCodeSuccess, "", ""},
		{[]string{"-C", atFile, "t", "Camera_Move"}, cli.ExitCodeSuccess, "", ""},
		{[]string{"-C", atFile + ":Camera_Move", "rm"}, cli.ExitCodeSuccess, atFile + "\n", ""},
		{[]string{"-C", atItem, "mv", "score_hit"}, cli.ExitCodeSuccess, atFile + ":score_hit\n", ""},
		{[]string{"show", "fight-camera"}, cli.ExitCodeSuccess,
			"TASK           FILE                TYPE  ITEM       LINES\n" +
				"score-strikes  client/camera.odin  fn    score_hit  2\n", ""},
		{[]string{"show", "--md", "fight-camera"}, cli.ExitCodeSuccess, "# fight-camera\n\n## score-strikes\n\n" +
			"### `client/camera.odin`\n\n- **fn** `score_hit`\n  1. ask: who anchors a kill\n  2. out: Camera_Cut\n", ""},
		{[]string{"show", "fight-camera", "--json"}, cli.ExitCodeSuccess, "", ""},
	})

	text, err := os.ReadFile(filepath.Join(root, "docs", "features", "fight-camera", "sketch.yml"))
	if err != nil {
		t.Fatalf("the sketch wasn't saved at the repo root: %v", err)
	}
	if !strings.Contains(string(text), "name: score_hit") {
		t.Errorf("sketch.yml =\n%s\nwant the renamed item", text)
	}
}

func TestShowJSON(t *testing.T) {
	inRepo(t, []run{
		{[]string{"-C", atFile, "fn", "score_strike"}, cli.ExitCodeSuccess, "", ""},
		{[]string{"show", "--json", "fight-camera"}, cli.ExitCodeSuccess, `{
  "schema_version": 1,
  "tasks": [
    {
      "name": "score-strikes",
      "files": [
        {
          "path": "client/camera.odin",
          "items": [
            {
              "type": "fn",
              "name": "score_strike"
            }
          ]
        }
      ]
    }
  ]
}
`, ""},
	})
}

func TestCommandsReject(t *testing.T) {
	tests := []struct {
		name string
		run  run
	}{
		{"A line verb outside an item", run{[]string{"-C", atFile, "in", "actor"}, cli.ExitCodeFailure, "",
			"in needs an item, and the context is the file " + atFile}},
		{"An unknown verb", run{[]string{"-C", atFile, "struct", "A"}, cli.ExitCodeFailure, "", `unknown verb "struct"`}},
		{"A context with no command", run{[]string{"-C", atFile}, cli.ExitCodeUsage, "", "needs a context and a command"}},
		{"A bad context", run{[]string{"-C", "../x", "fn", "a"}, cli.ExitCodeUsage, "", "bad context"}},
		{"show of a feature with no sketch", run{[]string{"show", "fight-camera"}, cli.ExitCodeFailure, "", "no sketch"}},
		{"show with both formats", run{[]string{"show", "--json", "--md", "fight-camera"}, cli.ExitCodeUsage, "",
			"can't both be given"}},
		{"show of a task", run{[]string{"show", "fight-camera/score-strikes"}, cli.ExitCodeUsage, "", "FEATURE"}},
		{"show without a feature", run{[]string{"show"}, cli.ExitCodeUsage, "", "FEATURE"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			root := inRepo(t, []run{tt.run})

			if _, err := os.Stat(filepath.Join(root, "docs")); err == nil {
				t.Errorf("a rejected command wrote docs/")
			}
		})
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

// TestUsageNamesEveryVerb keeps the help text in step with the parser: each full verb as a word, and each alias
// in parentheses after its verb.
func TestUsageNamesEveryVerb(t *testing.T) {
	for _, verb := range grammar.Verbs() {
		t.Run(verb, func(t *testing.T) {
			pattern := `\b` + regexp.QuoteMeta(verb) + `\b`
			if len(verb) == 1 {
				pattern = `\(` + regexp.QuoteMeta(verb) + `\)`
			}
			if !regexp.MustCompile(pattern).MatchString(usage) {
				t.Errorf("usage doesn't name %q", verb)
			}
		})
	}
}
