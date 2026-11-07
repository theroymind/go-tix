package command

import (
	"maps"
	"os"
	"slices"
	"strings"
	"testing"

	cli "github.com/theroymind/railyard-cli"
)

func TestDone(t *testing.T) {
	files := map[string]string{
		"a.go":                "",
		"a.go.sketch.md":      "---\nfeature: camera\n---\n",
		"src/b.go.sketch.md":  "---\nfeature: camera\nstatus: ready\n---\n",
		"c.go.sketch.md":      "---\nfeature: orbit\n---\n",
		"d.go.sketch.md":      "no feature\n",
		"camera.go.sketch.md": "---\nfeature: cameras\n---\n",
	}
	allNotes := []string{"a.go.sketch.md", "src/b.go.sketch.md", "c.go.sketch.md", "d.go.sketch.md",
		"camera.go.sketch.md"}
	tests := []struct {
		name      string
		arguments []string
		exitCode  cli.ExitCode
		stdout    string
		stderrHas string
		remaining []string
	}{
		{"Deletes exactly the feature's notes", []string{"done", "camera"}, cli.ExitCodeSuccess,
			"a.go.sketch.md\nsrc/b.go.sketch.md\n", "",
			[]string{"c.go.sketch.md", "d.go.sketch.md", "camera.go.sketch.md"}},
		{"A feature with no notes fails", []string{"done", "missing"}, cli.ExitCodeFailure, "",
			"sketch done: no notes for feature: missing", allNotes},
		{"No feature is a usage error", []string{"done"}, cli.ExitCodeUsage, "", "done takes one feature", allNotes},
		{"Two features are a usage error", []string{"done", "camera", "orbit"}, cli.ExitCodeUsage, "",
			"done takes one feature", allNotes},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			exitCode, stdout, stderr := inRepo(t, maps.Clone(files), tt.arguments...)

			if exitCode != tt.exitCode {
				t.Fatalf("sketch %q exit = %d, want %d (stderr %q)", tt.arguments, exitCode, tt.exitCode, stderr)
			}
			if stdout != tt.stdout {
				t.Errorf("sketch %q stdout = %q, want %q", tt.arguments, stdout, tt.stdout)
			}
			if !strings.Contains(stderr, tt.stderrHas) || (tt.stderrHas == "" && stderr != "") {
				t.Errorf("sketch %q stderr = %q, want it to say %q", tt.arguments, stderr, tt.stderrHas)
			}
			var remaining []string
			for _, notePath := range allNotes {
				if _, err := os.Stat("../" + notePath); err == nil {
					remaining = append(remaining, notePath)
				}
			}
			if !slices.Equal(remaining, tt.remaining) {
				t.Errorf("notes left = %q, want %q", remaining, tt.remaining)
			}
			if _, err := os.Stat("../a.go"); err != nil {
				t.Errorf("a.go is gone: %v", err)
			}
		})
	}
}
