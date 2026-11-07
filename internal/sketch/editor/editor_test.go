package editor

import (
	"slices"
	"testing"
)

func TestCommand(t *testing.T) {
	tests := []struct {
		name       string
		env        map[string]string
		wantArgv   []string
		wantRemote bool
	}{
		{"The parent nvim wins", map[string]string{"NVIM": "/tmp/nvim.sock", "EDITOR": "vim"},
			[]string{"nvim", "--server", "/tmp/nvim.sock", "--remote", "/repo/a.ts.sketch.md"}, true},
		{"EDITOR with arguments", map[string]string{"EDITOR": "code --wait", "VISUAL": "vim"},
			[]string{"code", "--wait", "/repo/a.ts.sketch.md"}, false},
		{"VISUAL when EDITOR is blank", map[string]string{"EDITOR": "  ", "VISUAL": "hx"},
			[]string{"hx", "/repo/a.ts.sketch.md"}, false},
		{"vi when neither is set", nil, []string{"vi", "/repo/a.ts.sketch.md"}, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			getenv := func(name string) string { return tt.env[name] }

			argv, remote := Command("/repo/a.ts.sketch.md", getenv)

			if !slices.Equal(argv, tt.wantArgv) || remote != tt.wantRemote {
				t.Errorf("Command() = %q, %v, want %q, %v", argv, remote, tt.wantArgv, tt.wantRemote)
			}
		})
	}
}
