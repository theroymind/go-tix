package editor

import (
	"reflect"
	"testing"
)

func TestCommands(t *testing.T) {
	const file = "/repo/a.ts.sketch.md"
	tests := []struct {
		name       string
		env        map[string]string
		line       int
		wantArgvs  [][]string
		wantRemote bool
	}{
		{"The parent nvim wins", map[string]string{"NVIM": "/tmp/nvim.sock", "EDITOR": "vim"}, 0,
			[][]string{{"nvim", "--server", "/tmp/nvim.sock", "--remote", file}}, true},
		{"The parent nvim goes to the line after opening", map[string]string{"NVIM": "/tmp/nvim.sock"}, 7,
			[][]string{{"nvim", "--server", "/tmp/nvim.sock", "--remote", file},
				{"nvim", "--server", "/tmp/nvim.sock", "--remote-send", `<C-\><C-N>:7<CR>`}}, true},
		{"EDITOR with arguments", map[string]string{"EDITOR": "code --wait", "VISUAL": "vim"}, 0,
			[][]string{{"code", "--wait", file}}, false},
		{"EDITOR with a line", map[string]string{"EDITOR": "vim"}, 3, [][]string{{"vim", "+3", file}}, false},
		{"VISUAL when EDITOR is blank", map[string]string{"EDITOR": "  ", "VISUAL": "hx"}, 0,
			[][]string{{"hx", file}}, false},
		{"vi when neither is set", nil, 0, [][]string{{"vi", file}}, false},
		{"vi with a line", nil, 2, [][]string{{"vi", "+2", file}}, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			getenv := func(name string) string { return tt.env[name] }

			argvs, remote := Commands(file, tt.line, getenv)

			if !reflect.DeepEqual(argvs, tt.wantArgvs) || remote != tt.wantRemote {
				t.Errorf("Commands() = %q, %v, want %q, %v", argvs, remote, tt.wantArgvs, tt.wantRemote)
			}
		})
	}
}
