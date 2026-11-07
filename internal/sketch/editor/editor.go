// Package editor opens a file for editing: in the parent nvim when sketch runs inside one, otherwise in $EDITOR,
// $VISUAL or vi.
package editor

import (
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
)

// Command returns the argv that opens filePath, reading the environment through getenv, and whether it returns at
// once instead of taking over the terminal. With $NVIM set it sends the file to that nvim with --remote, so nvim
// never nests inside nvim; otherwise it runs $EDITOR, which may hold arguments such as code --wait, else $VISUAL,
// else vi.
func Command(filePath string, getenv func(string) string) (argv []string, remote bool) {
	if server := getenv(nvimServer); server != "" {
		return []string{"nvim", "--server", server, "--remote", filePath}, true
	}
	for _, name := range []string{editorVar, visualVar} {
		if fields := strings.Fields(getenv(name)); len(fields) > 0 {
			return append(fields, filePath), false
		}
	}
	return []string{defaultEditor, filePath}, false
}

// Open returns the command that opens filePath and then sends done with the editor's error. A terminal editor
// suspends the program until it exits; a remote nvim call runs in the background.
func Open(filePath string, done func(error) tea.Msg) tea.Cmd {
	argv, remote := Command(filePath, os.Getenv)
	process := exec.Command(argv[0], argv[1:]...)
	if remote {
		return func() tea.Msg { return done(process.Run()) }
	}
	return tea.ExecProcess(process, done)
}
