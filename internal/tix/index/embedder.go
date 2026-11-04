package index

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
)

// Embedder turns texts into vectors, one per text, all of the same length.
type Embedder interface {
	Embed(texts []string) ([][]float32, error)
}

// OpenAIEmbedder embeds through an OpenAI-compatible /v1/embeddings endpoint, such as a local ollama or llama.cpp
// server. Endpoint is the base URL that /embeddings is appended to, such as http://localhost:11434/v1.
type OpenAIEmbedder struct {
	Endpoint string
	Model    string
	Client   *http.Client
}

type embeddingsRequest struct {
	Model string   `json:"model"`
	Input []string `json:"input"`
}

// embeddingsResponse is the part of the endpoint's reply that matters. The reply carries fields of the server's
// own too, so it is decoded leniently rather than through strictjson.
type embeddingsResponse struct {
	Data []embeddingsDatum `json:"data"`
}

type embeddingsDatum struct {
	Index     int       `json:"index"`
	Embedding []float32 `json:"embedding"`
}

func (embedder OpenAIEmbedder) Embed(texts []string) ([][]float32, error) {
	body, err := json.Marshal(embeddingsRequest{Model: embedder.Model, Input: texts})
	if err != nil {
		return nil, err
	}
	client := embedder.Client
	if client == nil {
		client = http.DefaultClient
	}
	url := strings.TrimSuffix(embedder.Endpoint, "/") + embeddingsPath
	response, err := client.Post(url, contentTypeJSON, bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("%w %s: %w", ErrEndpointUnreachable, embedder.Endpoint, err)
	}
	defer func() { _ = response.Body.Close() }()
	reply, err := io.ReadAll(response.Body)
	if err != nil {
		return nil, fmt.Errorf("%w %s: %w", ErrEndpointFailed, embedder.Endpoint, err)
	}
	if response.StatusCode < http.StatusOK || response.StatusCode >= http.StatusMultipleChoices {
		return nil, fmt.Errorf("%w %s: %s: %s", ErrEndpointFailed, embedder.Endpoint, response.Status, firstLine(reply))
	}
	return decodeVectors(reply, len(texts), embedder.Endpoint)
}

// decodeVectors reads the reply's vectors in the order of the texts, whatever order the reply lists them in.
func decodeVectors(reply []byte, count int, endpoint string) ([][]float32, error) {
	var decoded embeddingsResponse
	if err := json.Unmarshal(reply, &decoded); err != nil {
		return nil, fmt.Errorf("%w %s: %w", ErrEndpointFailed, endpoint, err)
	}
	if len(decoded.Data) != count {
		return nil, fmt.Errorf("%w: %d for %d texts", ErrEmbeddingCount, len(decoded.Data), count)
	}
	vectors := make([][]float32, count)
	for _, datum := range decoded.Data {
		if datum.Index < 0 || datum.Index >= count || vectors[datum.Index] != nil {
			return nil, fmt.Errorf("%w: index %d", ErrEmbeddingCount, datum.Index)
		}
		if len(datum.Embedding) == 0 {
			return nil, fmt.Errorf("%w: index %d", ErrEmptyVector, datum.Index)
		}
		vectors[datum.Index] = datum.Embedding
	}
	return vectors, nil
}

func firstLine(text []byte) string {
	line, _, _ := strings.Cut(strings.TrimSpace(string(text)), "\n")
	return line
}
