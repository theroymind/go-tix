package command

import (
	"maps"
	"os"
	"strings"
	"testing"

	cli "github.com/theroymind/railyard-cli"
)

func TestCheck(t *testing.T) {
	tickets := `[{"id":"camera","title":"Camera","type":"epic","status":"open"}]`
	readyCamera := map[string]string{
		"a.go":           "",
		"a.go.sketch.md": "---\nfeature: camera\nstatus: ready\n---\n? who\n> me\n-> a.go\n-> c.go : planned\n",
		"c.go.sketch.md": "---\nfeature: camera\nstatus: ready\n---\n",
	}
	tests := []struct {
		name      string
		tickets   string
		files     map[string]string
		arguments []string
		exitCode  cli.ExitCode
		stdout    string
		stderrHas string
	}{
		{"Ready notes with answered questions and resolved links pass", tickets, readyCamera,
			[]string{"check", "--feature", "camera"}, cli.ExitCodeSuccess, "", ""},
		{"Each problem is listed", tickets, map[string]string{
			"a.go.sketch.md": "---\nfeature: camera\nstatus: draft\n---\n? who\n-> gone.go\n",
			"b.go.sketch.md": "---\nfeature: camera\n---\n",
			"c.go.sketch.md": "---\nfeature: camera\nstatus: ready\n---\n",
		}, []string{"check", "--feature", "camera"}, cli.ExitCodeFailure,
			"a.go.sketch.md: not ready: draft\n" +
				"a.go.sketch.md:5: open question: who\n" +
				"a.go.sketch.md: dangling link -> gone.go\n" +
				"b.go.sketch.md: not ready: no status\n", "sketch check: not ready to build"},
		{"Other features' notes are left out", tickets, map[string]string{
			"a.go.sketch.md": "---\nfeature: camera\nstatus: ready\n---\n",
			"b.go.sketch.md": "---\nfeature: orbit\n---\n? why\n",
		}, []string{"check", "--feature", "camera"}, cli.ExitCodeSuccess, "", ""},
		{"Without --feature, every note is checked", tickets, map[string]string{
			"a.go.sketch.md": "---\nfeature: camera\nstatus: ready\n---\n",
			"b.go.sketch.md": "? why\n",
		}, []string{"check"}, cli.ExitCodeFailure,
			"b.go.sketch.md: not ready: no status\nb.go.sketch.md:1: open question: why\n", "not ready to build"},
		{"A feature tix doesn't know is a problem", tickets, map[string]string{
			"a.go.sketch.md": "---\nfeature: orbit\nstatus: ready\n---\n",
		}, []string{"check", "--feature", "orbit"}, cli.ExitCodeFailure,
			"a.go.sketch.md: feature orbit not in tix\n", "not ready to build"},
		{"Without tix, features go unchecked with a warning", "", map[string]string{
			"a.go.sketch.md": "---\nfeature: orbit\nstatus: ready\n---\n",
		}, []string{"check", "--feature", "orbit"}, cli.ExitCodeSuccess, "",
			"sketch: warning: not checking features: this is a tix repo, but tix isn't on PATH"},
		{"A feature with no notes is a problem", tickets, readyCamera, []string{"check", "--feature", "orbit"},
			cli.ExitCodeFailure, "orbit: no notes for this feature\n", "not ready to build"},
		{"A note that didn't read is a problem", tickets, map[string]string{
			"a.go.sketch.md": "---\nfeature: camera\nstatus: ready\n---\n",
			"b.go.sketch.md": "---\nfeature: [\n",
		}, []string{"check", "--feature", "camera"}, cli.ExitCodeFailure,
			"b.go.sketch.md: bad front matter: no closing --- line\n", "not ready to build"},
		{"--feature needs a value", tickets, readyCamera, []string{"check", "--feature"}, cli.ExitCodeUsage, "",
			"--feature"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fakeTix(t, tt.tickets)
			files := maps.Clone(tt.files)
			files[".tix/config.yml"] = ""

			exitCode, stdout, stderr := inRepo(t, files, tt.arguments...)

			if exitCode != tt.exitCode {
				t.Fatalf("sketch %q exit = %d, want %d (stdout %q, stderr %q)", tt.arguments, exitCode, tt.exitCode,
					stdout, stderr)
			}
			if stdout != tt.stdout {
				t.Errorf("sketch %q stdout =\n%s\nwant\n%s", tt.arguments, stdout, tt.stdout)
			}
			if !strings.Contains(stderr, tt.stderrHas) || (tt.stderrHas == "" && stderr != "") {
				t.Errorf("sketch %q stderr = %q, want it to say %q", tt.arguments, stderr, tt.stderrHas)
			}
		})
	}
}

func TestCheckOutsideATixRepo(t *testing.T) {
	log := fakeTix(t, `[]`)

	exitCode, stdout, stderr := inRepo(t, map[string]string{"a.go.sketch.md": "---\nfeature: orbit\nstatus: ready\n---\n"},
		"check", "--feature", "orbit")

	if exitCode != cli.ExitCodeSuccess || stdout != "" || stderr != "" {
		t.Errorf("exit %d, stdout %q, stderr %q, want a quiet pass", exitCode, stdout, stderr)
	}
	if runs, _ := os.ReadFile(log); len(runs) > 0 {
		t.Errorf("tix ran with %q outside a tix repo", runs)
	}
}
