package local

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/Server001-max/cr-coder/internal/llm/types"
)

type Client struct {
	baseURL   string
	modelsDir string
	httpClient *http.Client
	availableModels map[string]bool
	useLlamaCpp bool
}

func New(baseURL, modelsDir string) *Client {
	return &Client{
		baseURL:   strings.TrimSuffix(baseURL, "/"),
		modelsDir: modelsDir,
		httpClient: &http.Client{Timeout: 5 * time.Minute},
		availableModels: make(map[string]bool),
		useLlamaCpp: false,
	}
}

func (c *Client) Initialize() error {
	// Check if Ollama is running
	if c.checkOllama() {
		c.useLlamaCpp = false
		return c.refreshModels()
	}

	// Check if llama.cpp is available
	if c.checkLlamaCpp() {
		c.useLlamaCpp = true
		c.baseURL = "http://localhost:8080" // llama.cpp server default
		return c.refreshModels()
	}

	// Try to start Ollama if installed
	if c.tryStartOllama() {
		time.Sleep(2 * time.Second)
		if c.checkOllama() {
			c.useLlamaCpp = false
			return c.refreshModels()
		}
	}

	return fmt.Errorf("no local LLM runtime found. Install ollama (https://ollama.ai) or llama.cpp")
}

func (c *Client) checkOllama() bool {
	resp, err := c.httpClient.Get(c.baseURL + "/api/tags")
	if err != nil {
		return false
	}
	defer resp.Body.Close()
	return resp.StatusCode == 200
}

func (c *Client) checkLlamaCpp() bool {
	// Check if llama-server is running on port 8080
	resp, err := http.Get("http://localhost:8080/health")
	if err != nil {
		return false
	}
	defer resp.Body.Close()
	return resp.StatusCode == 200
}

func (c *Client) tryStartOllama() bool {
	// Check if ollama binary exists
	_, err := exec.LookPath("ollama")
	if err != nil {
		return false
	}
	// Try to start ollama serve in background
	cmd := exec.Command("ollama", "serve")
	cmd.Stdout = nil
	cmd.Stderr = nil
	return cmd.Start() == nil
}

func (c *Client) refreshModels() error {
	var models []string
	var err error

	if c.useLlamaCpp {
		models, err = c.listLlamaCppModels()
	} else {
		models, err = c.listOllamaModels()
	}

	if err != nil {
		return err
	}

	c.availableModels = make(map[string]bool)
	for _, m := range models {
		c.availableModels[m] = true
	}
	return nil
}

func (c *Client) listOllamaModels() ([]string, error) {
	resp, err := c.httpClient.Get(c.baseURL + "/api/tags")
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var result struct {
		Models []struct {
			Name string `json:"name"`
		} `json:"models"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}

	models := make([]string, len(result.Models))
	for i, m := range result.Models {
		models[i] = m.Name
	}
	return models, nil
}

func (c *Client) listLlamaCppModels() ([]string, error) {
	// llama.cpp doesn't have a standard models endpoint
	// Check models directory for GGUF files
	entries, err := os.ReadDir(c.modelsDir)
	if err != nil {
		return []string{}, nil
	}

	var models []string
	for _, e := range entries {
		if !e.IsDir() && (strings.HasSuffix(e.Name(), ".gguf") || strings.HasSuffix(e.Name(), ".bin")) {
			models = append(models, strings.TrimSuffix(e.Name(), filepath.Ext(e.Name())))
		}
	}
	return models, nil
}

func (c *Client) Chat(ctx context.Context, messages []types.Message, opts ...types.Option) (*types.Response, error) {
	options := &types.ChatOptions{}
	for _, opt := range opts {
		opt(options)
	}

	reqBody := map[string]interface{}{
		"model":    c.getModelName(),
		"messages": messages,
		"stream":   false,
		"options": map[string]interface{}{
			"temperature": options.Temperature,
			"num_predict": options.MaxTokens,
		},
	}

	if options.Temperature == 0 {
		reqBody["options"].(map[string]interface{})["temperature"] = 0.1
	}
	if options.MaxTokens == 0 {
		reqBody["options"].(map[string]interface{})["num_predict"] = 8192
	}

	jsonBody, _ := json.Marshal(reqBody)

	req, err := http.NewRequestWithContext(ctx, "POST", c.baseURL+"/api/chat", bytes.NewReader(jsonBody))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)

	var result struct {
		Message struct {
			Content string `json:"content"`
		} `json:"message"`
		Done bool `json:"done"`
	}
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("parse response: %w, body: %s", err, string(body))
	}

	return &types.Response{
		Content: result.Message.Content,
		Usage: types.Usage{
			TotalTokens: len(result.Message.Content) / 4, // rough estimate
		},
	}, nil
}

func (c *Client) StreamChat(ctx context.Context, messages []types.Message, opts ...types.Option) (<-chan types.StreamChunk, error) {
	options := &types.ChatOptions{}
	for _, opt := range opts {
		opt(options)
	}

	reqBody := map[string]interface{}{
		"model":    c.getModelName(),
		"messages": messages,
		"stream":   true,
		"options": map[string]interface{}{
			"temperature": options.Temperature,
			"num_predict": options.MaxTokens,
		},
	}

	if options.Temperature == 0 {
		reqBody["options"].(map[string]interface{})["temperature"] = 0.1
	}
	if options.MaxTokens == 0 {
		reqBody["options"].(map[string]interface{})["num_predict"] = 8192
	}

	jsonBody, _ := json.Marshal(reqBody)

	req, err := http.NewRequestWithContext(ctx, "POST", c.baseURL+"/api/chat", bytes.NewReader(jsonBody))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}

	ch := make(chan types.StreamChunk, 10)

	go func() {
		defer resp.Body.Close()
		defer close(ch)

		decoder := json.NewDecoder(resp.Body)
		for {
			var chunk struct {
				Message struct {
					Content string `json:"content"`
				} `json:"message"`
				Done bool `json:"done"`
			}
			if err := decoder.Decode(&chunk); err != nil {
				if err != io.EOF {
					ch <- types.StreamChunk{Error: err}
				}
				break
			}
			ch <- types.StreamChunk{
				Content: chunk.Message.Content,
				Done:    chunk.Done,
			}
			if chunk.Done {
				break
			}
		}
	}()

	return ch, nil
}

func (c *Client) Models() ([]string, error) {
	models := make([]string, 0, len(c.availableModels))
	for m := range c.availableModels {
		models = append(models, m)
	}
	return models, nil
}

func (c *Client) PullModel(name string) error {
	if c.useLlamaCpp {
		return fmt.Errorf("model pulling not supported for llama.cpp. Place GGUF files in %s", c.modelsDir)
	}

	reqBody := map[string]string{"name": name, "stream": "false"}
	jsonBody, _ := json.Marshal(reqBody)

	resp, err := c.httpClient.Post(c.baseURL+"/api/pull", "application/json", bytes.NewReader(jsonBody))
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	_, _ = io.ReadAll(resp.Body)
	return c.refreshModels()
}

func (c *Client) IsModelAvailable(name string) bool {
	return c.availableModels[name]
}

func (c *Client) Close() error {
	return nil
}

func (c *Client) getModelName() string {
	// Return first available model or default
	for m := range c.availableModels {
		return m
	}
	return "qwen2.5-coder:7b"
}