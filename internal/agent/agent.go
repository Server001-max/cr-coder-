package agent

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/Server001-max/cr-coder/internal/config"
	"github.com/Server001-max/cr-coder/internal/llm/router"
	"github.com/Server001-max/cr-coder/internal/llm/types"
	"github.com/Server001-max/cr-coder/internal/session"
	"github.com/Server001-max/cr-coder/internal/tools"
)

const systemPrompt = `You are CR CODER, an expert AI coding agent that helps users with software development tasks.

You have access to a set of tools to interact with the user's codebase and system. Use them to:
- Read, write, and edit files
- Search code and files
- Run shell commands
- Work with git
- Navigate and understand codebases

Guidelines:
- Be concise and direct
- Prefer using tools over guessing
- Explain what you're doing before complex operations
- Ask for clarification when needed
- Follow the user's instructions precisely
- Write clean, idiomatic code
- Don't add unnecessary comments or documentation

When the user asks you to do something:
1. Understand the task
2. Plan your approach if complex
3. Execute using tools
4. Verify the result
5. Report back concisely`

type Agent struct {
	config     *config.Config
	router     *router.Router
	tools      *tools.Registry
	session    *session.Session
	workingDir string
}

func Initialize(cfg *config.Config, r *router.Router) error {
	fmt.Println("🔧 Initializing CR CODER...")

	// Check if model is available
	if !r.IsModelAvailable(cfg.LLM.Model) {
		fmt.Printf("📥 Model '%s' not found. Downloading...\n", cfg.LLM.Model)
		if err := r.PullModel(cfg.LLM.Model); err != nil {
			return fmt.Errorf("failed to download model: %w", err)
		}
		fmt.Println("✅ Model downloaded successfully")
	} else {
		fmt.Printf("✅ Model '%s' is available\n", cfg.LLM.Model)
	}

	// Initialize directories
	dirs := []string{cfg.Paths.DataDir, cfg.Paths.CacheDir, cfg.Paths.ModelsDir, cfg.Session.CheckpointDir}
	for _, dir := range dirs {
		os.MkdirAll(dir, 0755)
	}

	fmt.Println("✅ CR CODER initialized successfully!")
	fmt.Println("\nUsage:")
	fmt.Println("  cr-coder chat \"your message\"     - Chat with AI")
	fmt.Println("  cr-coder agent \"task description\" - Run coding agent")
	fmt.Println("  cr-coder config show              - Show configuration")
	return nil
}

func Chat(cfg *config.Config, r *router.Router, message string) error {
	agent := newAgent(cfg, r)
	defer agent.Close()

	ctx := context.Background()

	if message == "" {
		return agent.interactiveChat(ctx)
	}

	messages := []types.Message{
		{Role: "system", Content: systemPrompt},
		{Role: "user", Content: message},
	}

	resp, err := r.Chat(ctx, messages)
	if err != nil {
		return err
	}

	fmt.Println(resp.Content)
	return nil
}

func Run(cfg *config.Config, r *router.Router, task string) error {
	agent := newAgent(cfg, r)
	defer agent.Close()

	ctx := context.Background()

	if task == "" {
		fmt.Print("Enter task: ")
		fmt.Scanln(&task)
	}

	fmt.Printf("🤖 Starting agent task: %s\n\n", task)
	return agent.runTask(ctx, task)
}

func newAgent(cfg *config.Config, r *router.Router) *Agent {
	wd, _ := os.Getwd()
	toolReg := tools.NewRegistry(&cfg.Tools)
	sess := session.New(&cfg.Session)

	return &Agent{
		config:     cfg,
		router:     r,
		tools:      toolReg,
		session:    sess,
		workingDir: wd,
	}
}

func (a *Agent) Close() error {
	return a.router.Close()
}

func (a *Agent) interactiveChat(ctx context.Context) error {
	fmt.Println("💬 Interactive chat mode (type 'exit' to quit)")
	fmt.Println()

	scanner := newScanner()
	messages := []types.Message{{Role: "system", Content: systemPrompt}}

	for {
		fmt.Print("You: ")
		input, ok := scanner()
		if !ok || strings.ToLower(strings.TrimSpace(input)) == "exit" {
			break
		}

		messages = append(messages, types.Message{Role: "user", Content: input})

		if a.config.Agent.ShowThinking {
			fmt.Print("CR CODER: ")
		}

		resp, err := a.router.Chat(ctx, messages)
		if err != nil {
			fmt.Printf("Error: %v\n", err)
			continue
		}

		fmt.Println(resp.Content)
		messages = append(messages, types.Message{Role: "assistant", Content: resp.Content})

		// Keep history manageable
		if len(messages) > 20 {
			messages = append([]types.Message{messages[0]}, messages[len(messages)-19:]...)
		}
	}
	return nil
}

func (a *Agent) runTask(ctx context.Context, task string) error {
	messages := []types.Message{
		{Role: "system", Content: systemPrompt + "\n\nYou are in agent mode. Use tools to complete the task."},
		{Role: "user", Content: task},
	}

	toolsDef := a.tools.GetDefinitions()

	for step := 0; step < a.config.Agent.MaxSteps; step++ {
		if a.config.Agent.ShowThinking {
			fmt.Printf("🔄 Step %d/%d\n", step+1, a.config.Agent.MaxSteps)
		}

		resp, err := a.router.Chat(ctx, messages, types.WithTools(toolsDef))
		if err != nil {
			return fmt.Errorf("LLM error: %w", err)
		}

		// Check if the model wants to use tools
		if resp.Content != "" && a.config.Agent.ShowThinking {
			fmt.Println(resp.Content)
		}

		// For now, we'll parse tool calls from the response
		// In a full implementation, the model would return structured tool calls
		// This is a simplified version - the model responds with text and we check for tool calls

		messages = append(messages, types.Message{Role: "assistant", Content: resp.Content})

		// Simple heuristic: if response contains "DONE" or similar, finish
		if strings.Contains(strings.ToUpper(resp.Content), "TASK COMPLETE") ||
			strings.Contains(strings.ToUpper(resp.Content), "DONE") {
			fmt.Println("\n✅ Task completed!")
			break
		}

		// If no tools were called and we're not done, ask model to continue
		if step == a.config.Agent.MaxSteps-1 {
			fmt.Println("\n⚠️ Max steps reached")
		}
	}

	return nil
}

func newScanner() func() (string, bool) {
	return func() (string, bool) {
		var input string
		_, err := fmt.Scanln(&input)
		return input, err == nil
	}
}

// Tool result handling would go here in a full implementation