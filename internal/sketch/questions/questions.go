// Package questions lists a sketch's questions, the ask lines, each with a number that stays put as others are
// answered, and records an answer to one.
package questions

import (
	"errors"
	"fmt"
	"strings"

	"github.com/theroymind/go-tix/contracts/sketch"
	"github.com/theroymind/go-tix/internal/sketch/grammar"
)

const firstNumber = 1

var (
	ErrNoQuestion    = errors.New("no such question")
	ErrMissingAnswer = errors.New("an answer is required")
)

// Question is one ask line: its number among every question in the sketch, counted from 1 in the order they
// appear, the context of the item it asks about, its text, and its answer, empty while it's open.
type Question struct {
	Number   int    `json:"number"`
	Context  string `json:"context"`
	Question string `json:"question"`
	Answer   string `json:"answer,omitempty"`
}

// Open returns the unanswered questions of the feature's sketch. Their numbers count the answered ones too, so
// answering one leaves the others' numbers as they were.
func Open(feature string, value sketch.Sketch) []Question {
	open := []Question{}
	for _, question := range all(feature, value) {
		if question.Answer == "" {
			open = append(open, question)
		}
	}
	return open
}

// Answer records answer on question number, replacing any answer it had, and returns the edited sketch. value is
// not changed.
func Answer(value sketch.Sketch, number int, answer string) (sketch.Sketch, error) {
	if strings.TrimSpace(answer) == "" {
		return value, ErrMissingAnswer
	}
	edited := value.Trimmed()
	count := 0
	for taskIndex := range edited.Tasks {
		for fileIndex := range edited.Tasks[taskIndex].Files {
			for itemIndex := range edited.Tasks[taskIndex].Files[fileIndex].Items {
				lines := edited.Tasks[taskIndex].Files[fileIndex].Items[itemIndex].Lines
				for lineIndex := range lines {
					if lines[lineIndex].Type != sketch.LineTypeAsk {
						continue
					}
					count++
					if count == number {
						lines[lineIndex].Answer = answer
						return edited.Trimmed(), nil
					}
				}
			}
		}
	}
	return value, fmt.Errorf("%w %d: the sketch has %d", ErrNoQuestion, number, count)
}

func all(feature string, value sketch.Sketch) []Question {
	var found []Question
	for _, task := range value.Trimmed().Tasks {
		for _, file := range task.Files {
			for _, item := range file.Items {
				at := grammar.Context{Feature: feature, Task: task.Name, File: file.Path, Item: item.Name}
				for _, line := range item.Lines {
					if line.Type != sketch.LineTypeAsk {
						continue
					}
					found = append(found, Question{Number: len(found) + firstNumber, Context: at.String(),
						Question: line.Text, Answer: line.Answer})
				}
			}
		}
	}
	return found
}
