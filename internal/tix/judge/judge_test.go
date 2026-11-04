package judge_test

import (
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/theroymind/railyard/internal/cli"
	"github.com/theroymind/railyard/internal/contracts/verdict"
	"github.com/theroymind/railyard/internal/tix/judge"
)

func TestClaudeJudge(t *testing.T) {
	tests := []struct {
		name            string
		output          string
		exitCode        string
		expectedVerdict verdict.Verdict
		expectedError   error
	}{
		{"The structured output is the answer",
			`{"type":"result","is_error":false,"result":"ignored","session_id":"s","structured_output":{"verdict":"refuse","duplicate_of":"a1","conflicts_with":"","reason":"Same."}}`,
			"0", verdict.Verdict{Verdict: verdict.Refuse, DuplicateOf: "a1", Reason: "Same."}, nil},
		{"Without structured output the result text is the answer",
			`{"type":"result","is_error":false,"result":"{\"verdict\":\"accept\",\"duplicate_of\":\"\",\"conflicts_with\":\"\",\"reason\":\"New.\"}"}`,
			"0", verdict.Verdict{Verdict: verdict.Accept, Reason: "New."}, nil},
		{"An answer with an unknown field is invalid input",
			`{"type":"result","is_error":false,"structured_output":{"verdict":"accept","duplicate_of":"","conflicts_with":"","reason":"New.","score":1}}`,
			"0", verdict.Verdict{}, cli.ErrInvalidInput},
		{"An answer that fails validation is invalid input",
			`{"type":"result","is_error":false,"structured_output":{"verdict":"refuse","duplicate_of":"","conflicts_with":"","reason":"Weak."}}`,
			"0", verdict.Verdict{}, cli.ErrInvalidInput},
		{"A result marked as an error fails",
			`{"type":"result","is_error":true,"result":"rate limited"}`, "0", verdict.Verdict{}, judge.ErrClaudeFailed},
		{"An unknown output type fails",
			`{"type":"progress"}`, "0", verdict.Verdict{}, judge.ErrUnknownClaudeType},
		{"A command that fails with no output fails as a child",
			``, "3", verdict.Verdict{}, cli.ErrChildFailed},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			directory := t.TempDir()
			command := filepath.Join(directory, "fake-claude")
			script := "#!/bin/sh\ncat >/dev/null\nprintf '%s' '" + tt.output + "'\nexit " + tt.exitCode + "\n"
			if err := os.WriteFile(command, []byte(script), 0o755); err != nil {
				t.Fatal(err)
			}

			value, err := judge.Claude{Command: command, Model: "sonnet", Directory: directory}.Judge("prompt")

			if !errors.Is(err, tt.expectedError) {
				t.Fatalf("Judge() error = %v, expected %v", err, tt.expectedError)
			}
			if err == nil && value != tt.expectedVerdict {
				t.Errorf("Judge() = %+v, expected %+v", value, tt.expectedVerdict)
			}
		})
	}
}

func TestCheckNames(t *testing.T) {
	neighbours := []judge.Note{{ID: "turn-order-is-a-wait-counter"}, {ID: "combat-loop"}}
	tests := []struct {
		name          string
		value         verdict.Verdict
		expectedError error
	}{
		{"An acceptance names nothing to check", verdict.Verdict{Verdict: verdict.Accept, Reason: "New."}, nil},
		{"A duplicate of a neighbour passes", verdict.Verdict{Verdict: verdict.Refuse, DuplicateOf: "combat-loop", Reason: "Same."}, nil},
		{"A conflict with a neighbour passes", verdict.Verdict{Verdict: verdict.Refuse, ConflictsWith: "turn-order-is-a-wait-counter", Reason: "Contradicts."}, nil},
		{"A note the judge wasn't shown fails", verdict.Verdict{Verdict: verdict.Refuse, DuplicateOf: "mana-pool", Reason: "Same."}, judge.ErrUnknownNote},
		{"The draft itself fails", verdict.Verdict{Verdict: verdict.Refuse, ConflictsWith: "draft", Reason: "Itself."}, judge.ErrNamesDraft},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := judge.CheckNames(tt.value, "draft", neighbours)

			if !errors.Is(err, tt.expectedError) {
				t.Errorf("CheckNames() error = %v, expected %v", err, tt.expectedError)
			}
		})
	}
}
