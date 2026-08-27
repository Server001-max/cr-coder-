# CR CODER

<p align="center">
  <strong>AI Coding Agent for Terminal — Cross-platform, Local-first, Free</strong>
</p>

<p align="center">
  <a href="https://github.com/Server001-max/cr-coder/releases"><img src="https://img.shields.io/github/v/release/Server001-max/cr-coder" alt="Latest Release"></a>
  <a href="https://github.com/Server001-max/cr-coder/actions"><img src="https://github.com/Server001-max/cr-coder/actions/workflows/build.yml/badge.svg" alt="Build Status"></a>
  <a href="https://golang.org/dl/"><img src="https://img.shields.io/badge/Go-1.23+-00ADD8?logo=go" alt="Go Version"></a>
  <a href="https://opensource.org/licenses/MIT"><img src="https://img.shields.io/badge/License-MIT-green" alt="License"></a>
</p>

---

## ✨ Features

| Feature | Description |
|---------|-------------|
| **🤖 Local AI First** | Runs models locally via Ollama/llama.cpp — your code never leaves your machine |
| **💰 Completely Free** | No API keys, no subscriptions, no cloud costs |
| **🖥️ Cross-Platform** | Single binary for Linux, Windows, macOS (Intel & Apple Silicon) |
| **🛠️ Full Tool Access** | Read/write/edit files, grep, glob, shell, git, LSP integration |
| **💬 Chat & Agent Modes** | Quick questions or multi-step coding tasks |
| **📦 Zero Config** | Works out of the box, customizable when needed |
| **🔄 Auto-Updates** | Built-in update mechanism via GitHub Releases |

---

## 🚀 Quick Start

### Install (one-liner)

```bash
curl -fsSL https://raw.githubusercontent.com/Server001-max/cr-coder/main/scripts/install.sh | sh
```

### Manual Install

Download the latest release for your platform from [Releases](https://github.com/Server001-max/cr-coder/releases):

| Platform | File |
|----------|------|
| Linux x64 | `cr-coder_<version>_linux_amd64.tar.gz` |
| Linux ARM64 | `cr-coder_<version>_linux_arm64.tar.gz` |
| Windows x64 | `cr-coder_<version>_windows_amd64.zip` |
| macOS Intel | `cr-coder_<version>_darwin_amd64.tar.gz` |
| macOS Apple Silicon | `cr-coder_<version>_darwin_arm64.tar.gz` |

Extract and move `cr-coder` (or `cr-coder.exe`) to a directory in your PATH.

---

## 📋 Prerequisites

**For Local AI (Recommended):**

Install [Ollama](https://ollama.ai) — it's free, open-source, and runs locally:

```bash
# Linux/macOS
curl -fsSL https://ollama.ai/install.sh | sh

# Windows: Download from https://ollama.ai/download
```

Then pull a coding model (done automatically on first run):
```bash
ollama pull qwen2.5-coder:7b
```

**Optional: API Providers**  
Configure in `~/.config/cr-coder/config.yaml`:
- OpenAI / Azure OpenAI
- Anthropic (Claude)
- Groq (fast, free tier)
- OpenRouter (access to many models)

---

## 💡 Usage

### Initialize (first time)
```bash
cr-coder init
```
Downloads the default model (`qwen2.5-coder:7b`) via Ollama.

### Chat Mode
```bash
# Single question
cr-coder chat "How do I reverse a string in Go?"

# Interactive chat
cr-coder chat
```

### Agent Mode (Multi-step tasks)
```bash
cr-coder agent "Create a REST API in Go with user CRUD operations"
cr-coder agent "Refactor the auth module to use JWT"
cr-coder agent "Fix the memory leak in the worker pool"
```

### Configuration
```bash
# Show current config
cr-coder config show

# Set options
cr-coder config set llm.model qwen2.5-coder:32b
cr-coder config set llm.provider ollama
cr-coder config set agent.auto_approve true
```

### Version Info
```bash
cr-coder version
```

---

## 🎯 Recommended Models (Free, Local)

| Model | Size | RAM Required | Best For |
|-------|------|--------------|----------|
| `qwen2.5-coder:7b` | 4.7 GB | 8 GB | **Default** — Excellent balance |
| `qwen2.5-coder:32b` | 19 GB | 32 GB | Maximum code quality |
| `deepseek-coder-v2:16b` | 9 GB | 16 GB | Strong alternative |
| `codellama:34b` | 19 GB | 32 GB | Meta's code model |
| `qwen2.5-coder:1.5b` | 1 GB | 4 GB | Low-end machines |

Install any model: `ollama pull <model-name>`

---

## ⚙️ Configuration

Config file: `~/.config/cr-coder/config.yaml`

```yaml
llm:
  provider: "ollama"          # ollama, llamacpp, openai, anthropic, groq, openrouter
  model: "qwen2.5-coder:7b"   # Model name
  base_url: "http://localhost:11434"
  api_key: ""                 # For API providers
  temperature: 0.1
  max_tokens: 8192
  context_window: 32768
  auto_download: true

agent:
  max_steps: 50
  auto_approve: false
  show_thinking: true
  enable_planning: true
  default_mode: "auto"

tools:
  enable_shell: true
  enable_file_ops: true
  enable_git: true
  enable_lsp: true
  enable_web_search: false
  blocked_paths: [".git", "node_modules", ".env", "*.key", "*.pem"]

session:
  save_history: true
  history_path: "~/.local/share/cr-coder/history.json"
  max_history_size: 1000
  auto_checkpoint: true
  checkpoint_dir: "~/.local/share/cr-coder/checkpoints"
```

---

## 🛠️ Available Tools

| Tool | Description |
|------|-------------|
| `read` | Read file contents |
| `write` | Create/write files |
| `edit` | Edit files (string replace) |
| `list` | List directory contents |
| `glob` | Find files by pattern |
| `grep` | Search code/text in files |
| `bash` | Execute shell commands |
| `git` | Git operations (status, diff, log, commit, push, etc.) |

---

## 🏗️ Building from Source

```bash
git clone https://github.com/Server001-max/cr-coder.git
cd cr-coder
go mod download
go build -ldflags="-s -w" -o cr-coder ./cmd/cr-coder
```

### Cross-compile
```bash
# Linux
GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build -o cr-coder-linux-amd64 ./cmd/cr-coder

# Windows
GOOS=windows GOARCH=amd64 CGO_ENABLED=0 go build -o cr-coder.exe ./cmd/cr-coder

# macOS Intel
GOOS=darwin GOARCH=amd64 CGO_ENABLED=0 go build -o cr-coder-darwin-amd64 ./cmd/cr-coder

# macOS Apple Silicon
GOOS=darwin GOARCH=arm64 CGO_ENABLED=0 go build -o cr-coder-darwin-arm64 ./cmd/cr-coder
```

---

## 🤝 Contributing

1. Fork the repository
2. Create a feature branch: `git checkout -b feature/amazing-feature`
3. Commit changes: `git commit -m 'feat: add amazing feature'`
4. Push to branch: `git push origin feature/amazing-feature`
5. Open a Pull Request

### Development Setup
```bash
# Install development tools
go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest
go install github.com/goreleaser/goreleaser@latest

# Run tests
go test -v -race ./...

# Run linter
golangci-lint run

# Build release locally
goreleaser release --snapshot --clean
```

---

## 📄 License

MIT License — see [LICENSE](LICENSE) for details.

---

## 🙏 Acknowledgments

- **Ollama** — Making local LLMs accessible
- **Qwen Team** — Excellent open-source coding models
- **DeepSeek** — Powerful open-source code models
- **Cobra** — CLI framework
- **GoReleaser** — Automated releases

---

## 📞 Support

- **Issues**: [GitHub Issues](https://github.com/Server001-max/cr-coder/issues)
- **Discussions**: [GitHub Discussions](https://github.com/Server001-max/cr-coder/discussions)

---

<p align="center">
  Made with ❤️ for developers who value privacy and freedom
</p>