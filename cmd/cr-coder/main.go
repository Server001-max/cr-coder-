package main

import (
	"fmt"
	"os"

	"github.com/Server001-max/cr-coder/internal/agent"
	"github.com/Server001-max/cr-coder/internal/config"
	"github.com/Server001-max/cr-coder/internal/llm/router"
	"github.com/spf13/cobra"
)

var (
	version = "dev"
	commit  = "none"
	date    = "unknown"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Config error: %v\n", err)
		os.Exit(1)
	}

	llmRouter := router.New(cfg.LLM)

	rootCmd := &cobra.Command{
		Use:     "cr-coder",
		Short:   "CR CODER - AI Coding Agent for Terminal",
		Version: fmt.Sprintf("%s (commit: %s, date: %s)", version, commit, date),
		PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
			return llmRouter.Initialize()
		},
	}

	rootCmd.AddCommand(
		newInitCmd(cfg, llmRouter),
		newChatCmd(cfg, llmRouter),
		newAgentCmd(cfg, llmRouter),
		newConfigCmd(cfg),
		newVersionCmd(),
	)

	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func newInitCmd(cfg *config.Config, llmRouter *router.Router) *cobra.Command {
	return &cobra.Command{
		Use:   "init",
		Short: "Initialize cr-coder and download default model",
		RunE: func(cmd *cobra.Command, args []string) error {
			return agent.Initialize(cfg, llmRouter)
		},
	}
}

func newChatCmd(cfg *config.Config, llmRouter *router.Router) *cobra.Command {
	return &cobra.Command{
		Use:   "chat [message]",
		Short: "Start a chat session with the AI",
		Args:  cobra.ArbitraryArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			message := ""
			if len(args) > 0 {
				message = args[0]
			}
			return agent.Chat(cfg, llmRouter, message)
		},
	}
}

func newAgentCmd(cfg *config.Config, llmRouter *router.Router) *cobra.Command {
	return &cobra.Command{
		Use:   "agent [task]",
		Short: "Run coding agent on a task",
		Args:  cobra.ArbitraryArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			task := ""
			if len(args) > 0 {
				task = args[0]
			}
			return agent.Run(cfg, llmRouter, task)
		},
	}
}

func newConfigCmd(cfg *config.Config) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "config",
		Short: "Manage configuration",
	}
	cmd.AddCommand(&cobra.Command{
		Use:   "show",
		Short: "Show current configuration",
		RunE: func(cmd *cobra.Command, args []string) error {
			return cfg.Print()
		},
	})
	cmd.AddCommand(&cobra.Command{
		Use:   "set [key] [value]",
		Short: "Set a configuration value",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			return cfg.Set(args[0], args[1])
		},
	})
	return cmd
}

func newVersionCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "version",
		Short: "Print version information",
		Run: func(cmd *cobra.Command, args []string) {
			fmt.Printf("cr-coder %s\n", version)
			fmt.Printf("Commit: %s\n", commit)
			fmt.Printf("Date: %s\n", date)
		},
	}
}