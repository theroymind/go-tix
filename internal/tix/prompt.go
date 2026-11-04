package tix

import (
	"bufio"
	"fmt"
	"io"
	"strings"
)

type prompter struct {
	lines  *bufio.Scanner
	output io.Writer
}

func newPrompter(input io.Reader, output io.Writer) prompter {
	lines := bufio.NewScanner(input)
	return prompter{lines: lines, output: output}
}

func (setup prompter) ask(question, fallback string, check func(answer string) error) (string, error) {
	label := fmt.Sprintf(promptPlain, question)
	if fallback != "" {
		label = fmt.Sprintf(promptWithDefault, question, fallback)
	}
	for {
		if _, err := fmt.Fprint(setup.output, label); err != nil {
			return "", err
		}
		if !setup.lines.Scan() {
			if err := setup.lines.Err(); err != nil {
				return "", err
			}
			return "", fmt.Errorf("%w at %q", ErrSetupEnded, question)
		}
		line := setup.lines.Text()
		answer := strings.TrimSpace(line)
		if answer == "" {
			answer = fallback
		}
		problem := check(answer)
		if problem == nil {
			return answer, nil
		}
		if _, err := fmt.Fprintf(setup.output, promptProblem, problem); err != nil {
			return "", err
		}
	}
}

func (setup prompter) askList(question, fallback string, check func(items []string) error) ([]string, error) {
	var items []string
	checkList := func(answer string) error {
		items = splitList(answer)
		return check(items)
	}
	if _, err := setup.ask(question, fallback, checkList); err != nil {
		return nil, err
	}
	return items, nil
}

func splitList(answer string) []string {
	var items []string
	for item := range strings.SplitSeq(answer, listItemSeparator) {
		trimmed := strings.TrimSpace(item)
		if trimmed != "" {
			items = append(items, trimmed)
		}
	}
	return items
}
