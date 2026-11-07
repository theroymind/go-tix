// Package editor opens a file for editing: in the parent nvim when sketch runs inside one, otherwise in $EDITOR,
// $VISUAL or vi.
package editor

import (
	"fmt"
	"os"
	"os/exec"
	"strings"

	tea "charm.land/bubbletea/v2"
)

const (
	nvimServer    = "NVIM"
	editorVar     = "EDITOR"
	visualVar     = "VISUAL"
	defaultEditor = "vi"
	lineArgument  = "+%d"
	nvimGoToLine  = `<C-\><C-N>:%d<CR>`
)

// Commands returns the argvs that open filePath, run one after another, with the cursor on line when it's above
// 0, reading the environment through getenv, and whether they return at once instead of taking over the terminal.
// With $NVIM set they send the file to that nvim with --remote, so nvim never nests inside nvim, and then send the
// keys that go to the line; otherwise they run $EDITOR, which may hold arguments such as code --wait, else
// $VISUAL, else vi, with +line before the file.
func Commands(filePath string, line int, getenv func(string) string) (argvs [][]string, remote bool) {
	if server := getenv(nvimServer); server != "" {
		argvs = [][]string{{"nvim", "--server", server, "--remote", filePath}}
		if line > 0 {
			argvs = append(argvs, []string{"nvim", "--server", server, "--remote-send", fmt.Sprintf(nvimGoToLine, line)})
		}
		return argvs, true
	}
	argv := []string{defaultEditor}
	for _, name := range []string{editorVar, visualVar} {
		if fields := strings.Fields(getenv(name)); len(fields) > 0 {
			argv = fields
			break
		}
	}
	if line > 0 {
		argv = append(argv, fmt.Sprintf(lineArgument, line))
	}
	return [][]string{append(argv, filePath)}, false
}

// Open returns the command that opens filePath at line, or at the top for line 0, and then sends done with the
// editor's error. A terminal editor suspends the program until it exits; remote nvim calls run in the background.
func Open(filePath string, line int, done func(error) tea.Msg) tea.Cmd {
	argvs, remote := Commands(filePath, line, os.Getenv)
	if !remote {
		argv := argvs[0]
		return tea.ExecProcess(exec.Command(argv[0], argv[1:]...), done)
	}
	return func() tea.Msg {
		for _, argv := range argvs {
			if err := exec.Command(argv[0], argv[1:]...).Run(); err != nil {
				return done(err)
			}
		}
		return done(nil)
	}
}
