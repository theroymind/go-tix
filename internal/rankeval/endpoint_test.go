package rankeval

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestEndpointCompleterComplete(t *testing.T) {
	tests := []struct {
		name          string
		status        int
		body          string
		expected      string
		expectedError error
	}{
		{"The first choice's content", http.StatusOK, `{"choices":[{"message":{"role":"assistant","content":"{\"scores\":[]}"}}],"usage":{"total_tokens":3}}`, `{"scores":[]}`, nil},
		{"No choices", http.StatusOK, `{"choices":[]}`, "", ErrEmptyCompletion},
		{"A failing status", http.StatusNotFound, `{"error": "generation thread died"}`, "", ErrEndpointFailed},
		{"A body that is not JSON", http.StatusOK, `<html>`, "", ErrEndpointFailed},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var received chatRequest
			server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
				if request.URL.Path != chatCompletionsPath {
					t.Errorf("path = %s, want %s", request.URL.Path, chatCompletionsPath)
				}
				if err := json.NewDecoder(request.Body).Decode(&received); err != nil {
					t.Errorf("request body: %v", err)
				}
				writer.WriteHeader(tt.status)
				_, _ = writer.Write([]byte(tt.body))
			}))
			defer server.Close()
			completer := newEndpointCompleter(server.URL, "a-model", false)

			answer, err := completer.complete("rank these")

			if !errors.Is(err, tt.expectedError) {
				t.Fatalf("complete() error = %v, want %v", err, tt.expectedError)
			}
			if answer != tt.expected {
				t.Errorf("complete() = %q, want %q", answer, tt.expected)
			}
			if received.Model != "a-model" || len(received.Messages) != 1 || received.Messages[0].Content != "rank these" || received.Messages[0].Role != chatRoleUser {
				t.Errorf("request = %+v", received)
			}
		})
	}
}

func TestEndpointCompleterUnreachable(t *testing.T) {
	server := httptest.NewServer(http.NotFoundHandler())
	server.Close()
	completer := newEndpointCompleter(server.URL, "a-model", false)

	_, err := completer.complete("rank these")

	if !errors.Is(err, ErrEndpointFailed) {
		t.Errorf("complete() error = %v, want %v", err, ErrEndpointFailed)
	}
}

func TestDecodeClaudeResult(t *testing.T) {
	tests := []struct {
		name          string
		output        string
		expected      string
		expectedError error
	}{
		{"A result with fields this tool doesn't know", `{"type":"result","subtype":"success","is_error":false,"result":"{\"scores\":[]}","total_cost_usd":0.001}`, `{"scores":[]}`, nil},
		{"Another type", `{"type":"assistant","result":"x"}`, "", ErrUnknownClaudeType},
		{"Not JSON", `oops`, "", ErrClaudeFailed},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := decodeClaudeResult([]byte(tt.output))

			if !errors.Is(err, tt.expectedError) {
				t.Fatalf("decodeClaudeResult() error = %v, want %v", err, tt.expectedError)
			}
			if result.Result != tt.expected {
				t.Errorf("decodeClaudeResult() result = %q, want %q", result.Result, tt.expected)
			}
		})
	}
}

func TestEndpointCompleterThinkingSwitch(t *testing.T) {
	tests := []struct {
		name             string
		thinking         bool
		expectedSwitch   bool
		expectedArgument *templateArguments
	}{
		{"Thinking off asks the server to disable it", false, true, &templateArguments{EnableThinking: false}},
		{"Thinking on leaves the template alone", true, false, nil},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var received map[string]json.RawMessage
			server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
				if err := json.NewDecoder(request.Body).Decode(&received); err != nil {
					t.Errorf("request body: %v", err)
				}
				_, _ = writer.Write([]byte(`{"choices":[{"message":{"role":"assistant","content":"ok"}}]}`))
			}))
			defer server.Close()
			completer := newEndpointCompleter(server.URL, "a-model", tt.thinking)

			if _, err := completer.complete("rank these"); err != nil {
				t.Fatal(err)
			}

			raw, hasSwitch := received["chat_template_kwargs"]
			if hasSwitch != tt.expectedSwitch {
				t.Fatalf("chat_template_kwargs sent = %v, want %v", hasSwitch, tt.expectedSwitch)
			}
			if hasSwitch && string(raw) != `{"enable_thinking":false}` {
				t.Errorf("chat_template_kwargs = %s", raw)
			}
			if _, hasMaximum := received["max_tokens"]; !hasMaximum {
				t.Error("max_tokens not sent")
			}
		})
	}
}
