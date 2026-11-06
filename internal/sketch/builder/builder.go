// Package builder provides the sketch builder: a prompt that shows where in a feature's sketch you are, runs each
// line through the grammar the one-shot commands share and saves it, and adds .., /, undo and edit, with history
// and tab completion.
package builder

import (
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"

	"github.com/peterh/liner"
)

// Run reads lines at the session's prompt until end of input, exit or quit, printing each line's output, or its
// error, to stdout. Ctrl-C clears the line being typed. Up-arrow steps back through the lines typed this session.
func Run(session *Session, completer Completer, stdout io.Writer) error {
	line := liner.NewLiner()
	defer func() { _ = line.Close() }()
	line.SetCtrlCAborts(true)
	line.SetTabCompletionStyle(liner.TabPrints)
	line.SetWordCompleter(completer.Complete)
	for {
		text, err := line.Prompt(session.Prompt())
		if errors.Is(err, liner.ErrPromptAborted) {
			continue
		}
		if errors.Is(err, io.EOF) {
			_, err = fmt.Fprintln(stdout)
			return err
		}
		if err != nil {
			return err
		}
		if strings.TrimSpace(text) != "" {
			line.AppendHistory(text)
		}
		output, err := session.Run(text)
		if errors.Is(err, ErrQuit) {
			return nil
		}
		if err != nil {
			output = err.Error()
		}
		if output != "" {
			if _, err := fmt.Fprintln(stdout, output); err != nil {
				return err
			}
		}
	}
}

// ExternalEditor returns an Editor that writes the text to a temporary file, opens it in $EDITOR, vi when that's
// unset, and returns the file as the editor left it. $EDITOR may hold arguments, such as code --wait.
func ExternalEditor() Editor {
	return func(text []byte) ([]byte, error) {
		file, err := os.CreateTemp("", editPattern)
		if err != nil {
			return nil, err
		}
		defer func() { _ = os.Remove(file.Name()) }()
		if _, err := file.Write(text); err != nil {
			_ = file.Close()
			return nil, err
		}
		if err := file.Close(); err != nil {
			return nil, err
		}
		command := strings.Fields(os.Getenv(editorVariable))
		if len(command) == 0 {
			command = []string{defaultEditor}
		}
		editor := exec.Command(command[0], append(command[1:], file.Name())...)
		editor.Stdin, editor.Stdout, editor.Stderr = os.Stdin, os.Stdout, os.Stderr
		if err := editor.Run(); err != nil {
			return nil, fmt.Errorf("%s: %w", command[0], err)
		}
		return os.ReadFile(file.Name())
	}
}
