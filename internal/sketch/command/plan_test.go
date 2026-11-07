package command

import (
	"maps"
	"strings"
	"testing"

	cli "github.com/theroymind/railyard-cli"
)

func TestPlanAndGraph(t *testing.T) {
	camera := `{"id":"camera","title":"Fight camera","type":"epic","status":"open",` +
		`"acceptance":[{"text":"It orbits.","done":false}]}`
	notes := map[string]string{
		"a.go.sketch.md":     "---\nfeature: camera\nstatus: ready\n---\nUse b.\n-> src/b.go : reads\n? who\n> me\n",
		"src/b.go.sketch.md": "---\nfeature: camera\nstatus: ready\n---\nThe base.\n",
		"c.go.sketch.md":     "---\nfeature: orbit\n---\nOther feature.\n",
	}
	tests := []struct {
		name       string
		tixFeature string
		files      map[string]string
		arguments  []string
		exitCode   cli.ExitCode
		stdoutHas  []string // in this order
		stdoutNot  string
		stderrHas  string
	}{
		{"plan prints the feature from tix, then its notes in build order and decisions", camera, notes,
			[]string{"plan", "--feature", "camera"}, cli.ExitCodeSuccess,
			[]string{"# Plan: camera\n", "## Fight camera\n", "1. [ ] It orbits.\n", "```mermaid\nflowchart LR\n",
				"### +src/b.go\n\nThe base.\n", "### +a.go\n\nUse b.\n", "## Decisions\n\n- a.go.sketch.md: who\n  > me\n"},
			"Other feature", ""},
		{"Without tix, plan warns and leaves the feature out", "", notes, []string{"plan", "--feature", "camera"},
			cli.ExitCodeSuccess, []string{"# Plan: camera\n\n## Graph\n", "### +src/b.go\n", "### +a.go\n"}, "Fight camera",
			"sketch: warning: leaving the feature's criteria out of the plan: this is a tix repo, but tix isn't on PATH"},
		{"Without --feature, plan takes every note", camera, notes, []string{"plan"}, cli.ExitCodeSuccess,
			[]string{"# Plan\n\n## Graph\n", "### +c.go\n\nOther feature.\n", "### +src/b.go\n", "### +a.go\n", "## Decisions"}, "Fight camera", ""},
		{"graph prints only the flowchart, a subgraph per folder", camera, notes,
			[]string{"graph", "--mermaid", "--feature", "camera"}, cli.ExitCodeSuccess,
			[]string{"flowchart LR\n", "  subgraph f0[\"repo root\"]\n    n0[\"+a.go\"]\n  end\n",
				"  subgraph f1[\"src\"]\n    n1[\"+b.go\"]\n  end\n", "  n0 -->|\"reads\"| n1\n"}, "Plan", ""},
		{"graph without --mermaid prints the same", camera, notes, []string{"graph"}, cli.ExitCodeSuccess,
			[]string{"flowchart LR\n", "c.go"}, "Plan", ""},
		{"A feature with no notes fails", camera, notes, []string{"plan", "--feature", "nope"}, cli.ExitCodeFailure,
			nil, "Plan", "no notes for feature: nope"},
		{"A note that didn't read fails after the plan prints", camera, map[string]string{
			"a.go.sketch.md": "---\nfeature: camera\n---\n",
			"b.go.sketch.md": "---\nfeature: [\n",
		}, []string{"graph", "--feature", "camera"}, cli.ExitCodeFailure, []string{"flowchart LR\n"}, "b.go",
			"some notes couldn't be read"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fakeTix(t, tt.tixFeature)
			files := maps.Clone(tt.files)
			files[".tix/config.yml"] = ""

			exitCode, stdout, stderr := inRepo(t, files, tt.arguments...)

			if exitCode != tt.exitCode {
				t.Fatalf("sketch %q exit = %d, want %d (stdout %q, stderr %q)", tt.arguments, exitCode, tt.exitCode,
					stdout, stderr)
			}
			rest := stdout
			for _, part := range tt.stdoutHas {
				_, after, found := strings.Cut(rest, part)
				if !found {
					t.Fatalf("sketch %q stdout =\n%s\nwant %q after what came before", tt.arguments, stdout, part)
				}
				rest = after
			}
			if strings.Contains(stdout, tt.stdoutNot) {
				t.Errorf("sketch %q stdout =\n%s\nwant no %q", tt.arguments, stdout, tt.stdoutNot)
			}
			if !strings.Contains(stderr, tt.stderrHas) || (tt.stderrHas == "" && stderr != "") {
				t.Errorf("sketch %q stderr = %q, want it to say %q", tt.arguments, stderr, tt.stderrHas)
			}
		})
	}
}
