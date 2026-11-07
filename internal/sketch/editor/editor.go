// Package editor opens a file for editing: in the parent nvim when sketch runs inside one, otherwise in $EDITOR,
// $VISUAL or vi.
package editor

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
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

	// showExpr calls sketch_show(path, focus, line), the Lua function the LazyVim spec in the README defines, and
	// evaluates to 1 when it's there and 0 when it isn't.
	showExpr          = `luaeval("_G.sketch_show and (_G.sketch_show(_A[1], _A[2], _A[3]) or 1) or 0", ['%s', v:%t, %d])`
	shownBySketchShow = "1"

	// hasShowExpr evaluates to 1 when the nvim has sketch_show, the function the LazyVim spec in the README
	// defines, and 0 when it doesn't.
	hasShowExpr = `luaeval("_G.sketch_show and 1 or 0")`
)

// lineEditors are the editors, by base name, that read +line as the line to open the file on. Others, such as
// code or subl, would open a file named +line.
var lineEditors = []string{"vi", "vim", "nvim", "nano", "emacs", "micro", "kak", "hx", "helix"}

// Paired reports whether sketch runs inside an nvim that has sketch_show, which then shows and edits its notes. It
// asks the nvim once; inside an nvim without the README's LazyVim spec, sketch isn't paired.
func Paired() bool {
	server := os.Getenv(nvimServer)
	if server == "" {
		return false
	}
	output, err := exec.Command("nvim", "--server", server, "--remote-expr", hasShowExpr).Output()
	return err == nil && strings.TrimSpace(string(output)) == shownBySketchShow
}

// Commands returns the argvs that open filePath, run one after another, with the cursor on line when it's above
// 0, reading the environment through getenv, and whether they return at once instead of taking over the terminal.
// With $NVIM set they send the file to that nvim with --remote, so nvim never nests inside nvim, and then send the
// keys that go to the line; otherwise they run $EDITOR, which may hold arguments such as code --wait, else
// $VISUAL, else vi, with +line before the file when the editor is one of lineEditors; any other gets the file
// alone.
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
	if line > 0 && slices.Contains(lineEditors, filepath.Base(argv[0])) {
		argv = append(argv, fmt.Sprintf(lineArgument, line))
	}
	return [][]string{append(argv, filePath)}, false
}

// ShowCommand returns the argv that asks the nvim at server to show filePath in its main window through
// sketch_show, moving focus there when focus is set, with the cursor on line when it's above 0. It prints 1 when
// sketch_show is defined and 0 when it isn't.
func ShowCommand(server, filePath string, focus bool, line int) []string {
	quoted := strings.ReplaceAll(filePath, "'", "''")
	return []string{"nvim", "--server", server, "--remote-expr", fmt.Sprintf(showExpr, quoted, focus, line)}
}

// Open returns the command that opens filePath at line, or at the top for line 0, and then sends done with the
// editor's error. A terminal editor suspends the program until it exits. Inside nvim it runs in the background:
// sketch_show focuses the note in the main window, or, when the LazyVim spec isn't loaded, --remote opens it.
func Open(filePath string, line int, done func(error) tea.Msg) tea.Cmd {
	argvs, remote := Commands(filePath, line, os.Getenv)
	if !remote {
		argv := argvs[0]
		return tea.ExecProcess(exec.Command(argv[0], argv[1:]...), done)
	}
	return func() tea.Msg {
		shown, err := show(filePath, true, line)
		if err != nil || shown {
			return done(err)
		}
		for _, argv := range argvs {
			if err := exec.Command(argv[0], argv[1:]...).Run(); err != nil {
				return done(err)
			}
		}
		return done(nil)
	}
}

// Follow returns the command that shows filePath in the parent nvim's main window, leaving focus where it is. It
// does nothing when the LazyVim spec isn't loaded, since plain --remote would open the note over sketch, and it
// drops errors, since a missed preview isn't worth a message.
func Follow(filePath string) tea.Cmd {
	return func() tea.Msg {
		_, _ = show(filePath, false, 0)
		return nil
	}
}

// show runs ShowCommand against $NVIM and reports whether sketch_show was there to show filePath.
func show(filePath string, focus bool, line int) (bool, error) {
	argv := ShowCommand(os.Getenv(nvimServer), filePath, focus, line)
	output, err := exec.Command(argv[0], argv[1:]...).Output()
	if err != nil {
		return false, err
	}
	return strings.TrimSpace(string(output)) == shownBySketchShow, nil
}
