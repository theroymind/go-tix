package rankeval

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"

	"github.com/theroymind/railyard-cli"
)

// claudeCompleter asks a hosted model through claude -p, with no tools, and takes its result text.
type claudeCompleter struct {
	model string
}

type claudeResult struct {
	Type    string `json:"type"`
	IsError bool   `json:"is_error"`
	Result  string `json:"result"`
}

func (completer claudeCompleter) complete(prompt string) (string, error) {
	arguments := append(append([]string{}, claudeArguments...), "--model="+completer.model)
	output, runErr := cli.RunChild(claudeExecutable, arguments, strings.NewReader(prompt))
	result, err := decodeClaudeResult(output)
	if err != nil {
		if runErr != nil {
			return "", runErr
		}
		return "", err
	}
	if result.IsError {
		return "", fmt.Errorf("%w: %s", ErrClaudeFailed, strings.TrimSpace(result.Result))
	}
	return result.Result, nil
}

// decodeClaudeResult reads claude's output leniently, since it is another tool's format that gains fields
// between releases; only the type is held to a known value.
func decodeClaudeResult(text []byte) (claudeResult, error) {
	var result claudeResult
	if err := json.Unmarshal(text, &result); err != nil {
		return claudeResult{}, fmt.Errorf("%w: %w", ErrClaudeFailed, err)
	}
	if result.Type != claudeResultType {
		return claudeResult{}, fmt.Errorf("%w %s", ErrUnknownClaudeType, strconv.Quote(result.Type))
	}
	return result, nil
}
