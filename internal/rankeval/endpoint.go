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
	base   string
	model  string
	client *http.Client
}

func newEndpointCompleter(base, model string) endpointCompleter {
	return endpointCompleter{base: base, model: model, client: &http.Client{Timeout: endpointTimeout}}
}

type chatRequest struct {
	Model       string        `json:"model"`
	Messages    []chatMessage `json:"messages"`
	Temperature float64       `json:"temperature"`
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
	body, err := json.Marshal(chatRequest{
		Model:       endpoint.model,
		Messages:    []chatMessage{{Role: chatRoleUser, Content: prompt}},
		Temperature: chatTemperature,
	})
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
