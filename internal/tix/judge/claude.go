package judge

import (
	"bytes"
	"encoding/json"
	"fmt"
	"slices"
	"strconv"
	"strings"

	"github.com/theroymind/railyard/internal/cli"
	"github.com/theroymind/railyard/internal/contracts/verdict"
)

// Claude is the Judge that asks Model through Command, which takes claude -p's arguments, run in Directory.
type Claude struct {
	Command   string
	Model     string
	Directory string
}

var _ Judge = Claude{}

// claudeOutput is claude -p's JSON result. StructuredOutput holds the answer when a JSON Schema shaped it.
type claudeOutput struct {
	Type             string          `json:"type"`
	IsError          bool            `json:"is_error"`
	Result           string          `json:"result"`
	StructuredOutput json.RawMessage `json:"structured_output"`
}

// Judge sends the prompt on stdin with the Verdict schema and returns the verdict once it has decoded strictly and
// validated.
func (judge Claude) Judge(prompt string) (verdict.Verdict, error) {
	arguments := append(slices.Clone(claudeArguments), "--model="+judge.Model, "--json-schema="+verdict.Schema)
	output, runErr := cli.RunChildIn(judge.Directory, judge.Command, arguments, strings.NewReader(prompt))
	answer, err := decodeOutput(output)
	if err != nil {
		if runErr != nil {
			return verdict.Verdict{}, runErr
		}
		return verdict.Verdict{}, err
	}
	return cli.ReadJSON[verdict.Verdict](bytes.NewReader(answer))
}

// decodeOutput reads claude's envelope leniently, since it is another tool's format that gains fields between
// releases, holding only its type to a known value, and returns the answer: the structured output when there is
// one, else the result text.
func decodeOutput(text []byte) ([]byte, error) {
	var output claudeOutput
	if err := json.Unmarshal(text, &output); err != nil {
		return nil, fmt.Errorf("%w: %w", ErrClaudeFailed, err)
	}
	if output.Type != claudeResultType {
		return nil, fmt.Errorf("%w %s", ErrUnknownClaudeType, strconv.Quote(output.Type))
	}
	if output.IsError {
		return nil, fmt.Errorf("%w: %s", ErrClaudeFailed, strings.TrimSpace(output.Result))
	}
	if hasValue(output.StructuredOutput) {
		return output.StructuredOutput, nil
	}
	return []byte(output.Result), nil
}

func hasValue(raw json.RawMessage) bool {
	trimmed := bytes.TrimSpace(raw)
	return len(trimmed) > 0 && !bytes.Equal(trimmed, []byte(jsonNull))
}
