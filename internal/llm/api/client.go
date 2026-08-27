package api

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/Server001-max/cr-coder/internal/config"
	"github.com/Server001-max/cr-coder/internal/llm/types"
)

type Client struct {
	config     *config.LLMConfig
	httpClient *http.Client
	models     []string
}

func New(cfg *config.LLMConfig) *Client {
	return &Client{
		config:     cfg,
		httpClient: &http.Client{Timeout: 5 * time.Minute},
		models:     []string{},
	}
}

func (c *Client) Initialize() error {
	if c.config.APIKey == "" {
		return fmt.Errorf("API key required for provider: %s", c.config.Provider)
	}
	return c.refreshModels()
}

func (c *Client) refreshModels() error {
	// For API providers, we use known model lists
	switch strings.ToLower(c.config.Provider) {
	case "openai":
		c.models = []string{"gpt-4o", "gpt-4o-mini", "gpt-4-turbo", "gpt-3.5-turbo"}
	case "anthropic":
		c.models = []string{"claude-3-5-sonnet-20241022", "claude-3-5-haiku-20241022", "claude-3-opus-20240229"}
	case "groq":
		c.models = []string{"llama-3.1-70b-versatile", "llama-3.1-8b-instant", "mixtral-8x7b-32768"}
	case "openrouter":
		c.models = []string{"qwen/qwen-2.5-coder-32b-instruct", "deepseek/deepseek-coder", "anthropic/claude-3.5-sonnet"}
	default:
		c.models = []string{c.config.Model}
	}
	return nil
}

func (c *Client) Chat(ctx context.Context, messages []types.Message, opts ...types.Option) (*types.Response, error) {
	options := &types.ChatOptions{
		Temperature: c.config.Temperature,
		MaxTokens:   c.config.MaxTokens,
	}
	for _, opt := range opts {
		opt(options)
	}

	var resp *types.Response
	var err error

	switch strings.ToLower(c.config.Provider) {
	case "openai":
		resp, err = c.chatOpenAI(ctx, messages, options)
	case "anthropic":
		resp, err = c.chatAnthropic(ctx, messages, options)
	case "groq":
		resp, err = c.chatOpenAICompatible(ctx, messages, options, "https://api.groq.com/openai/v1/chat/completions")
	case "openrouter":
		resp, err = c.chatOpenAICompatible(ctx, messages, options, "https://openrouter.ai/api/v1/chat/completions")
	default:
		return nil, fmt.Errorf("unsupported API provider: %s", c.config.Provider)
	}

	return resp, err
}

func (c *Client) chatOpenAI(ctx context.Context, messages []types.Message, options *types.ChatOptions) (*types.Response, error) {
	return c.chatOpenAICompatible(ctx, messages, options, c.config.BaseURL+"/chat/completions")
}

func (c *Client) chatOpenAICompatible(ctx context.Context, messages []types.Message, options *types.ChatOptions, endpoint string) (*types.Response, error) {
	reqBody := map[string]interface{}{
		"model":       c.config.Model,
		"messages":    messages,
		"temperature": options.Temperature,
		"max_tokens":  options.MaxTokens,
		"stream":      false,
	}

	if len(options.Tools) > 0 {
		tools := make([]map[string]interface{}, len(options.Tools))
		for i, t := range options.Tools {
			tools[i] = map[string]interface{}{
				"type": "function",
				"function": map[string]interface{}{
					"name":        t.Name,
					"description": t.Description,
					"parameters":  t.Parameters,
				},
			}
		}
		reqBody["tools"] = tools
		if options.ToolChoice != "" {
			reqBody["tool_choice"] = options.ToolChoice
		}
	}

	jsonBody, _ := json.Marshal(reqBody)

	req, err := http.NewRequestWithContext(ctx, "POST", endpoint, bytes.NewReader(jsonBody))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+c.config.APIKey)

	if strings.Contains(endpoint, "openrouter") {
		req.Header.Set("HTTP-Referer", "https://github.com/Server001-max/cr-coder")
		req.Header.Set("X-Title", "CR CODER")
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)

	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("API error %d: %s", resp.StatusCode, string(body))
	}

	var result struct {
		Choices []struct {
			Message struct {
				Content string `json:"content"`
			} `json:"message"`
		} `json:"choices"`
		Usage struct {
			PromptTokens     int `json:"prompt_tokens"`
			CompletionTokens int `json:"completion_tokens"`
			TotalTokens      int `json:"total_tokens"`
		} `json:"usage"`
	}
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("parse response: %w", err)
	}

	if len(result.Choices) == 0 {
		return nil, fmt.Errorf("no response from API")
	}

	return &types.Response{
		Content: result.Choices[0].Message.Content,
		Usage: types.Usage{
			PromptTokens:     result.Usage.PromptTokens,
			CompletionTokens: result.Usage.CompletionTokens,
			TotalTokens:      result.Usage.TotalTokens,
		},
	}, nil
}

func (c *Client) chatAnthropic(ctx context.Context, messages []types.Message, options *types.ChatOptions) (*types.Response, error) {
	// Convert messages to Anthropic format
	var systemPrompt string
	var anthropicMessages []map[string]interface{}

	for _, m := range messages {
		if m.Role == "system" {
			systemPrompt = m.Content
		} else {
			anthropicMessages = append(anthropicMessages, map[string]interface{}{
				"role":    m.Role,
				"content": m.Content,
			})
		}
	}

	reqBody := map[string]interface{}{
		"model":      c.config.Model,
		"messages":   anthropicMessages,
		"max_tokens": options.MaxTokens,
		"temperature": options.Temperature,
		"stream":     false,
	}
	if systemPrompt != "" {
		reqBody["system"] = systemPrompt
	}

	jsonBody, _ := json.Marshal(reqBody)

	endpoint := c.config.BaseURL + "/v1/messages"
	req, err := http.NewRequestWithContext(ctx, "POST", endpoint, bytes.NewReader(jsonBody))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("x-api-key", c.config.APIKey)
	req.Header.Set("anthropic-version", "2023-06-01")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)

	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("Anthropic API error %d: %s", resp.StatusCode, string(body))
	}

	var result struct {
		Content []struct {
			Text string `json:"text"`
		} `json:"content"`
		Usage struct {
			InputTokens  int `json:"input_tokens"`
			OutputTokens int `json:"output_tokens"`
		} `json:"usage"`
	}
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("parse Anthropic response: %w", err)
	}

	var content string
	for _, c := range result.Content {
		content += c.Text
	}

	return &types.Response{
		Content: content,
		Usage: types.Usage{
			PromptTokens:     result.Usage.InputTokens,
			CompletionTokens: result.Usage.OutputTokens,
			TotalTokens:      result.Usage.InputTokens + result.Usage.OutputTokens,
		},
	}, nil
}

func (c *Client) StreamChat(ctx context.Context, messages []types.Message, opts ...types.Option) (<-chan types.StreamChunk, error) {
	// Simplified - not implementing streaming for API providers yet
	ch := make(chan types.StreamChunk, 1)
	go func() {
		resp, err := c.Chat(ctx, messages, opts...)
		if err != nil {
			ch <- types.StreamChunk{Error: err}
		} else {
			ch <- types.StreamChunk{Content: resp.Content, Done: true}
		}
		close(ch)
	}()
	return ch, nil
}

func (c *Client) Models() ([]string, error) {
	return c.models, nil
}

func (c *Client) PullModel(name string) error {
	return fmt.Errorf("model pulling not supported for API providers")
}

func (c *Client) IsModelAvailable(name string) bool {
	for _, m := range c.models {
		if m == name {
			return true
		}
	}
	return false
}

func (c *Client) Close() error {
	return nil
}