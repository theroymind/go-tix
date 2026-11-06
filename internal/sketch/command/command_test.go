package command

import (
	"fmt"
	"io"
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
	stdin     string
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
		exitCode := Program.Run(step.arguments, strings.NewReader(step.stdin), &stdout, &stderr)
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
		{[]string{"-C", atFile, "fn", "score_strike"}, "", cli.ExitCodeSuccess, atItem + "\n", ""},
		{[]string{"-C", atItem, "in", "actor", "Combatant_Id"}, "", cli.ExitCodeSuccess, atItem + "\n", ""},
		{[]string{"-C", atItem, "a", "who anchors a kill"}, "", cli.ExitCodeSuccess, "", ""},
		{[]string{"-C", atItem, "o", "Camera_Move"}, "", cli.ExitCodeSuccess, "", ""},
		{[]string{"-C", atItem, "set", "3", "Camera_Cut"}, "", cli.ExitCodeSuccess, "", ""},
		{[]string{"-C", atItem, "rm", "1"}, "", cli.ExitCodeSuccess, "", ""},
		{[]string{"-C", atFile, "t", "Camera_Move"}, "", cli.ExitCodeSuccess, "", ""},
		{[]string{"-C", atFile + ":Camera_Move", "rm"}, "", cli.ExitCodeSuccess, atFile + "\n", ""},
		{[]string{"-C", atItem, "mv", "score_hit"}, "", cli.ExitCodeSuccess, atFile + ":score_hit\n", ""},
		{[]string{"show", "fight-camera"}, "", cli.ExitCodeSuccess,
			"TASK           FILE                MARK  TYPE  ITEM       LINES\n" +
				"score-strikes  client/camera.odin  +     fn    score_hit  2\n", ""},
		{[]string{"show", "--md", "fight-camera"}, "", cli.ExitCodeSuccess, "# fight-camera\n\n## score-strikes\n\n" +
			"### `client/camera.odin`\n\n- + **fn** `score_hit`\n  1. ask: who anchors a kill\n  2. out: Camera_Cut\n", ""},
		{[]string{"show", "fight-camera", "--json"}, "", cli.ExitCodeSuccess, "", ""},
	})

	text, err := os.ReadFile(filepath.Join(root, "docs", "features", "fight-camera", "sketch.yml"))
	if err != nil {
		t.Fatalf("the sketch wasn't saved at the repo root: %v", err)
	}
	if !strings.Contains(string(text), "name: score_hit") {
		t.Errorf("sketch.yml =\n%s\nwant the renamed item", text)
	}
}

func TestBatch(t *testing.T) {
	batch := "fn score_strike\n\nin actor Combatant_Id\r\na who anchors a kill\ntype Camera_Move\nfield cut bool\n"
	root := inRepo(t, []run{
		{[]string{"-C", atFile, "-"}, batch, cli.ExitCodeSuccess, atFile + ":Camera_Move\n", ""},
		{[]string{"show", "--md", "fight-camera"}, "", cli.ExitCodeSuccess, "# fight-camera\n\n## score-strikes\n\n" +
			"### `client/camera.odin`\n\n- + **fn** `score_strike`\n  1. in: actor Combatant_Id\n" +
			"  2. ask: who anchors a kill\n- + **type** `Camera_Move`\n  1. field: cut bool\n", ""},
		{[]string{"-C", atItem, "-"}, "out Camera_Move\nrm 9\nout never saved\n", cli.ExitCodeFailure, "",
			"line 2: bad line number"},
		{[]string{"-C", atItem, "-"}, "out Camera_Move\n", cli.ExitCodeSuccess, atItem + "\n", ""},
	})

	text, err := os.ReadFile(filepath.Join(root, "docs", "features", "fight-camera", "sketch.yml"))
	if err != nil {
		t.Fatal(err)
	}
	if strings.Count(string(text), "text: Camera_Move") != 1 || strings.Contains(string(text), "never saved") {
		t.Errorf("sketch.yml =\n%s\nwant only the last batch's out line, and nothing from the failed batch", text)
	}
}

func TestShowJSON(t *testing.T) {
	inRepo(t, []run{
		{[]string{"-C", atFile, "fn", "score_strike"}, "", cli.ExitCodeSuccess, "", ""},
		{[]string{"show", "--json", "fight-camera"}, "", cli.ExitCodeSuccess, `{
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
		{"A line verb outside an item", run{[]string{"-C", atFile, "in", "actor"}, "", cli.ExitCodeFailure, "",
			"in needs an item, and the context is the file " + atFile}},
		{"An unknown verb", run{[]string{"-C", atFile, "struct", "A"}, "", cli.ExitCodeFailure, "", `unknown verb "struct"`}},
		{"A context with no command", run{[]string{"-C", atFile}, "", cli.ExitCodeUsage, "", "needs a context and a command"}},
		{"A bad context", run{[]string{"-C", "../x", "fn", "a"}, "", cli.ExitCodeUsage, "", "bad context"}},
		{"show of a feature with no sketch", run{[]string{"show", "fight-camera"}, "", cli.ExitCodeFailure, "", "no sketch"}},
		{"show with both formats", run{[]string{"show", "--json", "--md", "fight-camera"}, "", cli.ExitCodeUsage, "",
			"can't both be given"}},
		{"show of a task", run{[]string{"show", "fight-camera/score-strikes"}, "", cli.ExitCodeUsage, "", "FEATURE"}},
		{"A batch that fails on its first line", run{[]string{"-C", atFile, "-"}, "in actor\n", cli.ExitCodeFailure, "",
			"line 1: wrong context"}},
		{"An empty batch", run{[]string{"-C", atFile, "-"}, "\n  \n", cli.ExitCodeUsage, "", "stdin holds no commands"}},
		{"A blank single command", run{[]string{"-C", atFile, " "}, "", cli.ExitCodeFailure, "", "empty command"}},
		{"show without a feature", run{[]string{"show"}, "", cli.ExitCodeUsage, "", "FEATURE"}},
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

func TestShowMarksItemsTheirFileNames(t *testing.T) {
	root := t.TempDir()
	for _, dir := range []string{".git", "client"} {
		if err := os.Mkdir(filepath.Join(root, dir), 0o755); err != nil {
			t.Fatal(err)
		}
	}
	code := "package client\n\nCamera_Move :: struct { cut: bool }\nscore_strikes :: proc() {}\n"
	if err := os.WriteFile(filepath.Join(root, "client", "camera.odin"), []byte(code), 0o644); err != nil {
		t.Fatal(err)
	}
	t.Chdir(root)
	for _, step := range []run{
		{[]string{"-C", atFile, "-"}, "type Camera_Move\nfn score_strike\n", cli.ExitCodeSuccess, "", ""},
		{[]string{"-C", "fight-camera/score-strikes/client/hud.odin", "fn", "Camera_Move"}, "", cli.ExitCodeSuccess, "", ""},
		{[]string{"show", "fight-camera"}, "", cli.ExitCodeSuccess,
			"TASK           FILE                MARK  TYPE  ITEM          LINES\n" +
				"score-strikes  client/camera.odin  ~     type  Camera_Move   0\n" +
				"score-strikes  client/camera.odin  +     fn    score_strike  0\n" +
				"score-strikes  client/hud.odin     +     fn    Camera_Move   0\n", ""},
	} {
		var stdout, stderr strings.Builder
		exitCode := Program.Run(step.arguments, strings.NewReader(step.stdin), &stdout, &stderr)
		if exitCode != step.exitCode || (step.stdout != "" && stdout.String() != step.stdout) {
			t.Errorf("sketch %q = %d\n%s\nwant %d\n%s(stderr %q)", step.arguments, exitCode, stdout.String(),
				step.exitCode, step.stdout, stderr.String())
		}
	}
}

func TestQuestionsAndAnswer(t *testing.T) {
	batch := "fn score_strike\nask who anchors a kill\nask does a counter reset it\nt Camera_Move\na cut or ease\n"
	root := inRepo(t, []run{
		{[]string{"-C", atFile, "-"}, batch, cli.ExitCodeSuccess, "", ""},
		{[]string{"questions", "fight-camera"}, "", cli.ExitCodeSuccess,
			"N  CONTEXT                                                     QUESTION\n" +
				"1  " + atItem + "  who anchors a kill\n" +
				"2  " + atItem + "  does a counter reset it\n" +
				"3  " + atFile + ":Camera_Move   cut or ease\n", ""},
		{[]string{"answer", "fight-camera", "1", "the", "striker"}, "", cli.ExitCodeSuccess, "", ""},
		{[]string{"questions", "--json", "fight-camera"}, "", cli.ExitCodeSuccess, `[
  {
    "number": 2,
    "context": "` + atItem + `",
    "question": "does a counter reset it"
  },
  {
    "number": 3,
    "context": "` + atFile + `:Camera_Move",
    "question": "cut or ease"
  }
]
`, ""},
		{[]string{"answer", "fight-camera", "3", "ease"}, "", cli.ExitCodeSuccess, "", ""},
		{[]string{"answer", "fight-camera", "2", "no"}, "", cli.ExitCodeSuccess, "", ""},
		{[]string{"questions", "--json", "fight-camera"}, "", cli.ExitCodeSuccess, "[]\n", ""},
		{[]string{"answer", "fight-camera", "4", "late"}, "", cli.ExitCodeFailure, "", "no such question 4: the sketch has 3"},
		{[]string{"answer", "fight-camera", "one", "the striker"}, "", cli.ExitCodeUsage, "", `"one" isn't a question number`},
		{[]string{"answer", "fight-camera", "1"}, "", cli.ExitCodeUsage, "", "needs FEATURE, a question number"},
		{[]string{"answer", "fight-camera", "1", " "}, "", cli.ExitCodeFailure, "", "an answer is required"},
		{[]string{"questions", "fight-camera/score-strikes"}, "", cli.ExitCodeUsage, "", "FEATURE"},
	})

	text, err := os.ReadFile(filepath.Join(root, "docs", "features", "fight-camera", "sketch.yml"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(text), "answer: the striker") {
		t.Errorf("sketch.yml =\n%s\nwant question 1's answer", text)
	}
}

// fakeTix stands in for tix path: it prints the note path of each feature in notes, and fails like tix for any
// other.
func fakeTix(t *testing.T, notes map[string]string) {
	t.Helper()
	original := runChild
	t.Cleanup(func() { runChild = original })
	runChild = func(_, name string, arguments []string, _ io.Reader) ([]byte, error) {
		if name != "tix" || len(arguments) != 2 || arguments[0] != "path" {
			return nil, fmt.Errorf("unexpected child %s %q", name, arguments)
		}
		path, exists := notes[arguments[1]]
		if !exists {
			return nil, fmt.Errorf("%w: tix: exit status 1: tix path: no ticket matches %q", cli.ErrChildFailed, arguments[1])
		}
		return []byte(path + "\n"), nil
	}
}

func TestInsideATixRepo(t *testing.T) {
	root := t.TempDir()
	noteDir := filepath.Join(root, "docs", "phases", "fight-camera")
	for _, dir := range []string{filepath.Join(root, ".tix"), noteDir} {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			t.Fatal(err)
		}
	}
	fakeTix(t, map[string]string{"fight-camera": filepath.Join(noteDir, "fight-camera.md")})
	t.Chdir(root)
	for _, step := range []run{
		{[]string{"-C", atFile, "fn", "score_strike"}, "", cli.ExitCodeSuccess, atItem + "\n", ""},
		{[]string{"-C", "boss-fight/a/b.odin", "fn", "x"}, "", cli.ExitCodeFailure, "",
			`feature "boss-fight" is not a tix ticket`},
		{[]string{"show", "boss-fight"}, "", cli.ExitCodeFailure, "", `feature "boss-fight" is not a tix ticket`},
	} {
		var stdout, stderr strings.Builder
		exitCode := Program.Run(step.arguments, strings.NewReader(step.stdin), &stdout, &stderr)
		if exitCode != step.exitCode || (step.stdout != "" && stdout.String() != step.stdout) ||
			!strings.Contains(stderr.String(), step.stderrHas) {
			t.Errorf("sketch %q = %d, stdout %q, stderr %q; want %d, %q, mentioning %q", step.arguments, exitCode,
				stdout.String(), stderr.String(), step.exitCode, step.stdout, step.stderrHas)
		}
	}

	if _, err := os.Stat(filepath.Join(noteDir, "sketch.yml")); err != nil {
		t.Errorf("the sketch isn't beside the feature's note: %v", err)
	}
	if _, err := os.Stat(filepath.Join(root, "docs", "features")); err == nil {
		t.Errorf("a tix repo's sketch went to docs/features")
	}
}
