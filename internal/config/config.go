package config

import (
	"fmt"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

type Config struct {
	LLM       LLMConfig       `yaml:"llm"`
	Agent     AgentConfig     `yaml:"agent"`
	Tools     ToolsConfig     `yaml:"tools"`
	Session   SessionConfig   `yaml:"session"`
	Paths     PathsConfig     `yaml:"paths"`
	configDir string          `yaml:"-"`
}

type LLMConfig struct {
	Provider      string  `yaml:"provider"`        // "ollama", "llamacpp", "openai", "anthropic", "groq", "openrouter"
	Model         string  `yaml:"model"`           // model name
	BaseURL       string  `yaml:"base_url"`        // for ollama/llamacpp/openai-compatible
	APIKey        string  `yaml:"api_key"`         // for API providers
	Temperature   float64 `yaml:"temperature"`     // 0.0-1.0
	MaxTokens     int     `yaml:"max_tokens"`      // max output tokens
	ContextWindow int     `yaml:"context_window"`  // context window size
	AutoDownload  bool    `yaml:"auto_download"`   // auto-download model if missing
	DefaultModel  string  `yaml:"default_model"`   // default model to use
	ModelsDir     string  `yaml:"models_dir"`      // local models directory (for llama.cpp)
}

type AgentConfig struct {
	MaxSteps       int    `yaml:"max_steps"`        // max agent steps
	AutoApprove    bool   `yaml:"auto_approve"`     // auto-approve tool calls
	ShowThinking   bool   `yaml:"show_thinking"`    // show agent reasoning
	EnablePlanning bool   `yaml:"enable_planning"`  // enable planning phase
	DefaultMode    string `yaml:"default_mode"`     // "chat", "agent", "auto"
}

type ToolsConfig struct {
	EnableShell     bool     `yaml:"enable_shell"`      // enable shell tool
	EnableFileOps   bool     `yaml:"enable_file_ops"`   // enable file read/write/edit
	EnableGit       bool     `yaml:"enable_git"`        // enable git tool
	EnableLSP       bool     `yaml:"enable_lsp"`        // enable LSP integration
	EnableWebSearch bool     `yaml:"enable_web_search"` // enable web search
	AllowedPaths    []string `yaml:"allowed_paths"`     // allowed file paths
	BlockedPaths    []string `yaml:"blocked_paths"`     // blocked file paths
}

type SessionConfig struct {
	SaveHistory    bool   `yaml:"save_history"`     // save conversation history
	HistoryPath    string `yaml:"history_path"`     // path to history file
	MaxHistorySize int    `yaml:"max_history_size"` // max history entries
	AutoCheckpoint bool   `yaml:"auto_checkpoint"`  // auto-create checkpoints
	CheckpointDir  string `yaml:"checkpoint_dir"`   // checkpoint directory
}

type PathsConfig struct {
	ConfigDir  string `yaml:"config_dir"`
	DataDir    string `yaml:"data_dir"`
	CacheDir   string `yaml:"cache_dir"`
	ModelsDir  string `yaml:"models_dir"`
}

func DefaultConfig() *Config {
	homeDir, _ := os.UserHomeDir()
	configDir := filepath.Join(homeDir, ".config", "cr-coder")
	dataDir := filepath.Join(homeDir, ".local", "share", "cr-coder")
	cacheDir := filepath.Join(homeDir, ".cache", "cr-coder")
	modelsDir := filepath.Join(dataDir, "models")

	return &Config{
		LLM: LLMConfig{
			Provider:      "ollama",
			Model:         "qwen2.5-coder:7b",
			BaseURL:       "http://localhost:11434",
			Temperature:   0.1,
			MaxTokens:     8192,
			ContextWindow: 32768,
			AutoDownload:  true,
			DefaultModel:  "qwen2.5-coder:7b",
			ModelsDir:     modelsDir,
		},
		Agent: AgentConfig{
			MaxSteps:       50,
			AutoApprove:    false,
			ShowThinking:   true,
			EnablePlanning: true,
			DefaultMode:    "auto",
		},
		Tools: ToolsConfig{
			EnableShell:     true,
			EnableFileOps:   true,
			EnableGit:       true,
			EnableLSP:       true,
			EnableWebSearch: false,
			AllowedPaths:    []string{},
			BlockedPaths:    []string{".git", "node_modules", ".env", "*.key", "*.pem"},
		},
		Session: SessionConfig{
			SaveHistory:    true,
			HistoryPath:    filepath.Join(dataDir, "history.json"),
			MaxHistorySize: 1000,
			AutoCheckpoint: true,
			CheckpointDir:  filepath.Join(dataDir, "checkpoints"),
		},
		Paths: PathsConfig{
			ConfigDir: configDir,
			DataDir:   dataDir,
			CacheDir:  cacheDir,
			ModelsDir: modelsDir,
		},
		configDir: configDir,
	}
}

func Load() (*Config, error) {
	cfg := DefaultConfig()

	configFile := filepath.Join(cfg.configDir, "config.yaml")
	if _, err := os.Stat(configFile); err == nil {
		data, err := os.ReadFile(configFile)
		if err != nil {
			return nil, fmt.Errorf("read config: %w", err)
		}
		if err := yaml.Unmarshal(data, cfg); err != nil {
			return nil, fmt.Errorf("parse config: %w", err)
		}
	}

	// Ensure directories exist
	for _, dir := range []string{cfg.Paths.ConfigDir, cfg.Paths.DataDir, cfg.Paths.CacheDir, cfg.Paths.ModelsDir, cfg.Session.CheckpointDir} {
		if err := os.MkdirAll(dir, 0755); err != nil {
			return nil, fmt.Errorf("create dir %s: %w", dir, err)
		}
	}

	cfg.configDir = cfg.Paths.ConfigDir
	return cfg, nil
}

func (c *Config) Save() error {
	configFile := filepath.Join(c.configDir, "config.yaml")
	data, err := yaml.Marshal(c)
	if err != nil {
		return fmt.Errorf("marshal config: %w", err)
	}
	return os.WriteFile(configFile, data, 0644)
}

func (c *Config) Print() error {
	data, err := yaml.Marshal(c)
	if err != nil {
		return err
	}
	fmt.Println(string(data))
	return nil
}

func (c *Config) Set(key, value string) error {
	// Simple key-value setting for common config options
	switch key {
	case "llm.provider":
		c.LLM.Provider = value
	case "llm.model":
		c.LLM.Model = value
	case "llm.base_url":
		c.LLM.BaseURL = value
	case "llm.api_key":
		c.LLM.APIKey = value
	case "llm.temperature":
		var v float64
		fmt.Sscanf(value, "%f", &v)
		c.LLM.Temperature = v
	case "agent.max_steps":
		var v int
		fmt.Sscanf(value, "%d", &v)
		c.Agent.MaxSteps = v
	case "agent.auto_approve":
		c.Agent.AutoApprove = value == "true"
	default:
		return fmt.Errorf("unknown config key: %s", key)
	}
	return c.Save()
}