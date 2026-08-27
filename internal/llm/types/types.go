package types

import "context"

type Message struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type Response struct {
	Content string
	Usage   Usage
}

type Usage struct {
	PromptTokens     int
	CompletionTokens int
	TotalTokens      int
}

type StreamChunk struct {
	Content string
	Done    bool
	Error   error
}

type Option func(*ChatOptions)

type ChatOptions struct {
	Temperature   float64
	MaxTokens     int
	StopSequences []string
	Tools         []Tool
	ToolChoice    string
}

type Tool struct {
	Name        string
	Description string
	Parameters  map[string]interface{}
}

type Provider interface {
	Chat(ctx context.Context, messages []Message, opts ...Option) (*Response, error)
	StreamChat(ctx context.Context, messages []Message, opts ...Option) (<-chan StreamChunk, error)
	Models() ([]string, error)
	PullModel(name string) error
	IsModelAvailable(name string) bool
	Close() error
}

func WithTemperature(t float64) Option {
	return func(o *ChatOptions) { o.Temperature = t }
}

func WithMaxTokens(t int) Option {
	return func(o *ChatOptions) { o.MaxTokens = t }
}

func WithTools(tools []Tool) Option {
	return func(o *ChatOptions) { o.Tools = tools }
}