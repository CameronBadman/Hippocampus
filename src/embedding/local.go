package embedding

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

// OllamaClient provides local embedding generation via Ollama
type OllamaClient struct {
	BaseURL string
	Model   string
	client  *http.Client
}

// NewOllamaClient creates a new Ollama client
func NewOllamaClient(baseURL, model string) *OllamaClient {
	if baseURL == "" {
		baseURL = "http://localhost:11434"
	}
	if model == "" {
		model = "nomic-embed-text" // Default embedding model
	}

	return &OllamaClient{
		BaseURL: baseURL,
		Model:   model,
		client:  &http.Client{},
	}
}

// ollamaEmbedRequest is the request format for Ollama's embeddings API
type ollamaEmbedRequest struct {
	Model  string `json:"model"`
	Prompt string `json:"prompt"`
}

// ollamaEmbedResponse is the response format
type ollamaEmbedResponse struct {
	Embedding []float64 `json:"embedding"`
}

// GetEmbedding generates an embedding using Ollama
func (o *OllamaClient) GetEmbedding(text string) ([]float32, error) {
	reqBody := ollamaEmbedRequest{
		Model:  o.Model,
		Prompt: text,
	}

	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	resp, err := o.client.Post(
		o.BaseURL+"/api/embeddings",
		"application/json",
		bytes.NewBuffer(jsonData),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to call Ollama API: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("Ollama API error (status %d): %s", resp.StatusCode, string(body))
	}

	var embedResp ollamaEmbedResponse
	if err := json.NewDecoder(resp.Body).Decode(&embedResp); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	// Convert float64 to float32
	embedding := make([]float32, len(embedResp.Embedding))
	for i, v := range embedResp.Embedding {
		embedding[i] = float32(v)
	}

	return embedding, nil
}

// LlamaCppClient provides local embedding generation via llama.cpp server
type LlamaCppClient struct {
	BaseURL string
	client  *http.Client
}

// NewLlamaCppClient creates a new llama.cpp client
func NewLlamaCppClient(baseURL string) *LlamaCppClient {
	if baseURL == "" {
		baseURL = "http://localhost:8080"
	}

	return &LlamaCppClient{
		BaseURL: baseURL,
		client:  &http.Client{},
	}
}

// llamaCppEmbedRequest is the request format for llama.cpp's /embedding API
type llamaCppEmbedRequest struct {
	Content string `json:"content"`
}

// llamaCppEmbedResponse is the response format
type llamaCppEmbedResponse struct {
	Embedding []float32 `json:"embedding"`
}

// GetEmbedding generates an embedding using llama.cpp
func (l *LlamaCppClient) GetEmbedding(text string) ([]float32, error) {
	reqBody := llamaCppEmbedRequest{
		Content: text,
	}

	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	resp, err := l.client.Post(
		l.BaseURL+"/embedding",
		"application/json",
		bytes.NewBuffer(jsonData),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to call llama.cpp API: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("llama.cpp API error (status %d): %s", resp.StatusCode, string(body))
	}

	var embedResp llamaCppEmbedResponse
	if err := json.NewDecoder(resp.Body).Decode(&embedResp); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return embedResp.Embedding, nil
}

// EmbeddingProvider is a generic interface for embedding providers
type EmbeddingProvider interface {
	GetEmbedding(text string) ([]float32, error)
}

// Ensure our types implement the interface
var _ EmbeddingProvider = (*OllamaClient)(nil)
var _ EmbeddingProvider = (*LlamaCppClient)(nil)
