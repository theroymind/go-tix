package rankeval

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

const endpointTimeout = 10 * time.Minute

// endpointCompleter asks an OpenAI-compatible chat server for one completion of one user message.
type endpointCompleter struct {
	base     string
	model    string
	thinking bool
	client   *http.Client
}

func newEndpointCompleter(base, model string, thinking bool) endpointCompleter {
	return endpointCompleter{base: base, model: model, thinking: thinking, client: &http.Client{Timeout: endpointTimeout}}
}

// chatRequest is one chat completion. TemplateArguments carries the MLX server's enable_thinking switch, left
// out when the model may think; the answer is read from the content, apart from any reasoning returned.
type chatRequest struct {
	Model             string             `json:"model"`
	Messages          []chatMessage      `json:"messages"`
	Temperature       float64            `json:"temperature"`
	MaximumTokens     int                `json:"max_tokens"`
	TemplateArguments *templateArguments `json:"chat_template_kwargs,omitempty"`
}

type templateArguments struct {
	EnableThinking bool `json:"enable_thinking"`
}

type chatMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type chatResponse struct {
	Choices []chatChoice `json:"choices"`
}

type chatChoice struct {
	Message chatMessage `json:"message"`
}

func (endpoint endpointCompleter) complete(prompt string) (string, error) {
	request := chatRequest{
		Model:         endpoint.model,
		Messages:      []chatMessage{{Role: chatRoleUser, Content: prompt}},
		Temperature:   chatTemperature,
		MaximumTokens: chatMaximumTokens,
	}
	if !endpoint.thinking {
		request.TemplateArguments = &templateArguments{EnableThinking: false}
	}
	body, err := json.Marshal(request)
	if err != nil {
		return "", err
	}
	response, err := endpoint.client.Post(endpoint.base+chatCompletionsPath, "application/json", bytes.NewReader(body))
	if err != nil {
		return "", fmt.Errorf("%w: %w", ErrEndpointFailed, err)
	}
	defer func() { _ = response.Body.Close() }()
	text, err := io.ReadAll(response.Body)
	if err != nil {
		return "", fmt.Errorf("%w: %w", ErrEndpointFailed, err)
	}
	if response.StatusCode != http.StatusOK {
		return "", fmt.Errorf("%w: status %d: %s", ErrEndpointFailed, response.StatusCode, bytes.TrimSpace(text))
	}
	var decoded chatResponse
	if err := json.Unmarshal(text, &decoded); err != nil {
		return "", fmt.Errorf("%w: %w", ErrEndpointFailed, err)
	}
	if len(decoded.Choices) == 0 {
		return "", ErrEmptyCompletion
	}
	return decoded.Choices[0].Message.Content, nil
}
