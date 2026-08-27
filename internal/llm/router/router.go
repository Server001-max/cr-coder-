package router

import (
	"context"
	"fmt"
	"strings"

	"github.com/Server001-max/cr-coder/internal/config"
	"github.com/Server001-max/cr-coder/internal/llm/api"
	"github.com/Server001-max/cr-coder/internal/llm/local"
	"github.com/Server001-max/cr-coder/internal/llm/types"
)

type Router struct {
	config   *config.LLMConfig
	local    *local.Client
	api      *api.Client
	provider types.Provider
}

func New(cfg *config.LLMConfig) *Router {
	r := &Router{config: cfg}
	r.local = local.New(cfg.BaseURL, cfg.ModelsDir)
	r.api = api.New(cfg)
	return r
}

func (r *Router) Initialize() error {
	switch strings.ToLower(r.config.Provider) {
	case "ollama", "llamacpp":
		if err := r.local.Initialize(); err != nil {
			return err
		}
		r.provider = r.local
		return nil
	case "openai", "anthropic", "groq", "openrouter":
		if err := r.api.Initialize(); err != nil {
			return err
		}
		r.provider = r.api
		return nil
	default:
		// Auto-detect: try local first, then API
		if err := r.local.Initialize(); err == nil {
			r.provider = r.local
			r.config.Provider = "ollama"
			return nil
		}
		if err := r.api.Initialize(); err == nil {
			r.provider = r.api
			return nil
		}
		return fmt.Errorf("no LLM provider available. Install ollama or configure API key")
	}
}

func (r *Router) Chat(ctx context.Context, messages []types.Message, opts ...types.Option) (*types.Response, error) {
	if r.provider == nil {
		return nil, fmt.Errorf("provider not initialized")
	}
	return r.provider.Chat(ctx, messages, opts...)
}

func (r *Router) StreamChat(ctx context.Context, messages []types.Message, opts ...types.Option) (<-chan types.StreamChunk, error) {
	if r.provider == nil {
		return nil, fmt.Errorf("provider not initialized")
	}
	return r.provider.StreamChat(ctx, messages, opts...)
}

func (r *Router) Models() ([]string, error) {
	if r.provider == nil {
		return nil, fmt.Errorf("provider not initialized")
	}
	return r.provider.Models()
}

func (r *Router) PullModel(name string) error {
	if r.provider == nil {
		return fmt.Errorf("provider not initialized")
	}
	return r.provider.PullModel(name)
}

func (r *Router) IsModelAvailable(name string) bool {
	if r.provider == nil {
		return false
	}
	return r.provider.IsModelAvailable(name)
}

func (r *Router) Close() error {
	if r.provider != nil {
		return r.provider.Close()
	}
	return nil
}

func (r *Router) GetProvider() string {
	if r.provider == nil {
		return "none"
	}
	switch r.provider.(type) {
	case *local.Client:
		return "local"
	case *api.Client:
		return "api"
	}
	return "unknown"
}