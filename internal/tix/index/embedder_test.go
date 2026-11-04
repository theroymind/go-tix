package index_test

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"testing"

	"github.com/theroymind/railyard/internal/tix/index"
)

type fakeRequest struct {
	Model string   `json:"model"`
	Input []string `json:"input"`
}

// fakeEndpoint serves /v1/embeddings the way an OpenAI-compatible server does: one vector per input, listed in
// reverse so that the order is taken from each datum's index. The vector is the input's length and the model's.
func fakeEndpoint(t *testing.T) (*httptest.Server, *[]fakeRequest) {
	t.Helper()
	var requests []fakeRequest
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		if request.URL.Path != "/v1/embeddings" || request.Method != http.MethodPost {
			http.NotFound(writer, request)
			return
		}
		var received fakeRequest
		if err := json.NewDecoder(request.Body).Decode(&received); err != nil {
			http.Error(writer, err.Error(), http.StatusBadRequest)
			return
		}
		requests = append(requests, received)
		data := []map[string]any{}
		for position := len(received.Input) - 1; position >= 0; position-- {
			data = append(data, map[string]any{
				"object": "embedding", "index": position,
				"embedding": []float32{float32(len(received.Input[position])), float32(len(received.Model))},
			})
		}
		reply := map[string]any{"object": "list", "model": received.Model, "data": data, "usage": map[string]int{"total_tokens": 3}}
		writer.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(writer).Encode(reply); err != nil {
			t.Error(err)
		}
	}))
	t.Cleanup(server.Close)
	return server, &requests
}

func TestOpenAIEmbedder(t *testing.T) {
	server, requests := fakeEndpoint(t)
	embedder := index.OpenAIEmbedder{Endpoint: server.URL + "/v1", Model: "fake"}

	vectors, err := embedder.Embed([]string{"one", "three"})
	if err != nil {
		t.Fatal(err)
	}

	expected := [][]float32{{3, 4}, {5, 4}}
	if !reflect.DeepEqual(vectors, expected) {
		t.Errorf("vectors = %v, expected %v", vectors, expected)
	}
	expectedRequests := []fakeRequest{{Model: "fake", Input: []string{"one", "three"}}}
	if !reflect.DeepEqual(*requests, expectedRequests) {
		t.Errorf("requests = %+v, expected %+v", *requests, expectedRequests)
	}
}

func TestOpenAIEmbedderAcceptsATrailingSlash(t *testing.T) {
	server, _ := fakeEndpoint(t)
	embedder := index.OpenAIEmbedder{Endpoint: server.URL + "/v1/", Model: "fake"}

	if _, err := embedder.Embed([]string{"one"}); err != nil {
		t.Error(err)
	}
}

func TestOpenAIEmbedderNamesADownEndpoint(t *testing.T) {
	server, _ := fakeEndpoint(t)
	endpoint := server.URL + "/v1"
	server.Close()
	embedder := index.OpenAIEmbedder{Endpoint: endpoint, Model: "fake"}

	_, err := embedder.Embed([]string{"one"})

	if !errors.Is(err, index.ErrEndpointUnreachable) || !strings.Contains(err.Error(), endpoint) {
		t.Errorf("error = %v, expected %v naming %s", err, index.ErrEndpointUnreachable, endpoint)
	}
}

func TestOpenAIEmbedderReportsBadReplies(t *testing.T) {
	tests := []struct {
		name        string
		status      int
		body        string
		expectedErr error
		expectedIn  string
	}{
		{"A failing status names the status and the body's first line", http.StatusNotFound, "model missing\nmore", index.ErrEndpointFailed, "404 Not Found: model missing"},
		{"A reply that isn't JSON", http.StatusOK, "<html>", index.ErrEndpointFailed, ""},
		{"Too few vectors", http.StatusOK, `{"data": []}`, index.ErrEmbeddingCount, ""},
		{"A repeated index", http.StatusOK, `{"data": [{"index": 0, "embedding": [1]}, {"index": 0, "embedding": [1]}]}`, index.ErrEmbeddingCount, ""},
		{"An empty vector", http.StatusOK, `{"data": [{"index": 0, "embedding": []}, {"index": 1, "embedding": [1]}]}`, index.ErrEmptyVector, ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, _ *http.Request) {
				writer.WriteHeader(tt.status)
				_, _ = writer.Write([]byte(tt.body))
			}))
			defer server.Close()
			embedder := index.OpenAIEmbedder{Endpoint: server.URL + "/v1", Model: "fake"}

			_, err := embedder.Embed([]string{"one", "two"})

			if !errors.Is(err, tt.expectedErr) || !strings.Contains(err.Error(), tt.expectedIn) {
				t.Errorf("error = %v, expected %v containing %q", err, tt.expectedErr, tt.expectedIn)
			}
		})
	}
}
