package tools

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/Server001-max/cr-coder/internal/config"
	"github.com/Server001-max/cr-coder/internal/llm/types"
)

type Registry struct {
	config     *config.ToolsConfig
	tools      map[string]Tool
	workingDir string
}

type Tool interface {
	Name() string
	Description() string
	Parameters() map[string]interface{}
	Execute(ctx context.Context, args map[string]interface{}) (string, error)
}

type ToolDef struct {
	Name        string
	Description string
	Parameters  map[string]interface{}
}

func NewRegistry(cfg *config.ToolsConfig) *Registry {
	wd, _ := os.Getwd()
	r := &Registry{
		config:     cfg,
		tools:      make(map[string]Tool),
		workingDir: wd,
	}

	// Register built-in tools
	if cfg.EnableFileOps {
		r.register(&ReadTool{base: r})
		r.register(&WriteTool{base: r})
		r.register(&EditTool{base: r})
		r.register(&ListTool{base: r})
		r.register(&GlobTool{base: r})
		r.register(&GrepTool{base: r})
	}

	if cfg.EnableShell {
		r.register(&BashTool{base: r})
	}

	if cfg.EnableGit {
		r.register(&GitTool{base: r})
	}

	return r
}

func (r *Registry) register(t Tool) {
	r.tools[t.Name()] = t
}

func (r *Registry) GetDefinitions() []types.Tool {
	defs := make([]types.Tool, 0, len(r.tools))
	for _, t := range r.tools {
		defs = append(defs, types.Tool{
			Name:        t.Name(),
			Description: t.Description(),
			Parameters:  t.Parameters(),
		})
	}
	return defs
}

func (r *Registry) Execute(ctx context.Context, name string, args map[string]interface{}) (string, error) {
	tool, ok := r.tools[name]
	if !ok {
		return "", fmt.Errorf("tool not found: %s", name)
	}
	return tool.Execute(ctx, args)
}

func (r *Registry) validatePath(path string) (string, error) {
	// Convert to absolute path
	absPath := path
	if !filepath.IsAbs(absPath) {
		absPath = filepath.Join(r.workingDir, absPath)
	}

	// Clean path
	absPath = filepath.Clean(absPath)

	// Check blocked paths
	for _, blocked := range r.config.BlockedPaths {
		if matched, _ := filepath.Match(blocked, filepath.Base(absPath)); matched {
			return "", fmt.Errorf("path blocked: %s", blocked)
		}
		if strings.Contains(absPath, blocked) {
			return "", fmt.Errorf("path blocked: %s", blocked)
		}
	}

	// Check allowed paths (if specified)
	if len(r.config.AllowedPaths) > 0 {
		allowed := false
		for _, allowedPath := range r.config.AllowedPaths {
			if strings.HasPrefix(absPath, allowedPath) {
				allowed = true
				break
			}
		}
		if !allowed {
			return "", fmt.Errorf("path not in allowed list: %s", absPath)
		}
	}

	return absPath, nil
}

// ReadTool
type ReadTool struct {
	base *Registry
}

func (t *ReadTool) Name() string        { return "read" }
func (t *ReadTool) Description() string { return "Read a file from the filesystem" }
func (t *ReadTool) Parameters() map[string]interface{} {
	return map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"path": map[string]interface{}{
				"type":        "string",
				"description": "Path to the file to read",
			},
			"offset": map[string]interface{}{
				"type":        "integer",
				"description": "Line number to start reading from (1-indexed)",
			},
			"limit": map[string]interface{}{
				"type":        "integer",
				"description": "Maximum number of lines to read",
			},
		},
		"required": []string{"path"},
	}
}

func (t *ReadTool) Execute(ctx context.Context, args map[string]interface{}) (string, error) {
	path := args["path"].(string)
	absPath, err := t.base.validatePath(path)
	if err != nil {
		return "", err
	}

	data, err := os.ReadFile(absPath)
	if err != nil {
		return "", err
	}

	content := string(data)
	lines := strings.Split(content, "\n")

	offset := 0
	if o, ok := args["offset"].(float64); ok {
		offset = int(o) - 1
	}
	if offset < 0 {
		offset = 0
	}
	if offset >= len(lines) {
		offset = 0
	}

	limit := len(lines)
	if l, ok := args["limit"].(float64); ok && int(l) > 0 {
		limit = int(l)
	}

	end := offset + limit
	if end > len(lines) {
		end = len(lines)
	}

	result := strings.Join(lines[offset:end], "\n")
	return fmt.Sprintf("File: %s (lines %d-%d of %d)\n%s", path, offset+1, end, len(lines), result), nil
}

// WriteTool
type WriteTool struct {
	base *Registry
}

func (t *WriteTool) Name() string        { return "write" }
func (t *WriteTool) Description() string { return "Write content to a file" }
func (t *WriteTool) Parameters() map[string]interface{} {
	return map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"path": map[string]interface{}{
				"type":        "string",
				"description": "Path to the file to write",
			},
			"content": map[string]interface{}{
				"type":        "string",
				"description": "Content to write",
			},
		},
		"required": []string{"path", "content"},
	}
}

func (t *WriteTool) Execute(ctx context.Context, args map[string]interface{}) (string, error) {
	path := args["path"].(string)
	content := args["content"].(string)

	absPath, err := t.base.validatePath(path)
	if err != nil {
		return "", err
	}

	// Create directory if needed
	dir := filepath.Dir(absPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return "", err
	}

	if err := os.WriteFile(absPath, []byte(content), 0644); err != nil {
		return "", err
	}

	return fmt.Sprintf("Successfully wrote %d bytes to %s", len(content), path), nil
}

// EditTool
type EditTool struct {
	base *Registry
}

func (t *EditTool) Name() string        { return "edit" }
func (t *EditTool) Description() string { return "Edit a file by replacing a string" }
func (t *EditTool) Parameters() map[string]interface{} {
	return map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"path": map[string]interface{}{
				"type":        "string",
				"description": "Path to the file to edit",
			},
			"old_string": map[string]interface{}{
				"type":        "string",
				"description": "Text to replace",
			},
			"new_string": map[string]interface{}{
				"type":        "string",
				"description": "Text to replace with",
			},
			"replace_all": map[string]interface{}{
				"type":        "boolean",
				"description": "Replace all occurrences",
			},
		},
		"required": []string{"path", "old_string", "new_string"},
	}
}

func (t *EditTool) Execute(ctx context.Context, args map[string]interface{}) (string, error) {
	path := args["path"].(string)
	oldStr := args["old_string"].(string)
	newStr := args["new_string"].(string)
	replaceAll := false
	if ra, ok := args["replace_all"].(bool); ok {
		replaceAll = ra
	}

	absPath, err := t.base.validatePath(path)
	if err != nil {
		return "", err
	}

	data, err := os.ReadFile(absPath)
	if err != nil {
		return "", err
	}

	content := string(data)
	if !strings.Contains(content, oldStr) {
		return "", fmt.Errorf("old_string not found in file")
	}

	var newContent string
	if replaceAll {
		newContent = strings.ReplaceAll(content, oldStr, newStr)
	} else {
		newContent = strings.Replace(content, oldStr, newStr, 1)
	}

	if err := os.WriteFile(absPath, []byte(newContent), 0644); err != nil {
		return "", err
	}

	return fmt.Sprintf("Successfully edited %s", path), nil
}

// ListTool
type ListTool struct {
	base *Registry
}

func (t *ListTool) Name() string        { return "list" }
func (t *ListTool) Description() string { return "List files in a directory" }
func (t *ListTool) Parameters() map[string]interface{} {
	return map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"path": map[string]interface{}{
				"type":        "string",
				"description": "Directory path to list",
			},
		},
		"required": []string{"path"},
	}
}

func (t *ListTool) Execute(ctx context.Context, args map[string]interface{}) (string, error) {
	path := args["path"].(string)
	absPath, err := t.base.validatePath(path)
	if err != nil {
		return "", err
	}

	entries, err := os.ReadDir(absPath)
	if err != nil {
		return "", err
	}

	var result []string
	for _, e := range entries {
		prefix := "  "
		if e.IsDir() {
			prefix += "📁 "
		} else {
			prefix += "📄 "
		}
		result = append(result, prefix+e.Name())
	}

	return strings.Join(result, "\n"), nil
}

// GlobTool
type GlobTool struct {
	base *Registry
}

func (t *GlobTool) Name() string        { return "glob" }
func (t *GlobTool) Description() string { return "Find files matching a glob pattern" }
func (t *GlobTool) Parameters() map[string]interface{} {
	return map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"pattern": map[string]interface{}{
				"type":        "string",
				"description": "Glob pattern (e.g., **/*.go)",
			},
			"path": map[string]interface{}{
				"type":        "string",
				"description": "Base directory to search in",
			},
		},
		"required": []string{"pattern"},
	}
}

func (t *GlobTool) Execute(ctx context.Context, args map[string]interface{}) (string, error) {
	pattern := args["pattern"].(string)
	basePath := t.base.workingDir
	if p, ok := args["path"].(string); ok && p != "" {
		var err error
		basePath, err = t.base.validatePath(p)
		if err != nil {
			return "", err
		}
	}

	matches, err := filepath.Glob(filepath.Join(basePath, pattern))
	if err != nil {
		return "", err
	}

	// Make paths relative to working dir
	var result []string
	for _, m := range matches {
		rel, _ := filepath.Rel(t.base.workingDir, m)
		result = append(result, rel)
	}

	return strings.Join(result, "\n"), nil
}

// GrepTool
type GrepTool struct {
	base *Registry
}

func (t *GrepTool) Name() string        { return "grep" }
func (t *GrepTool) Description() string { return "Search for a pattern in files" }
func (t *GrepTool) Parameters() map[string]interface{} {
	return map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"pattern": map[string]interface{}{
				"type":        "string",
				"description": "Regex pattern to search for",
			},
			"path": map[string]interface{}{
				"type":        "string",
				"description": "Directory to search in",
			},
			"include": map[string]interface{}{
				"type":        "string",
				"description": "File pattern to include (e.g., *.go)",
			},
		},
		"required": []string{"pattern"},
	}
}

func (t *GrepTool) Execute(ctx context.Context, args map[string]interface{}) (string, error) {
	pattern := args["pattern"].(string)
	basePath := t.base.workingDir
	if p, ok := args["path"].(string); ok && p != "" {
		var err error
		basePath, err = t.base.validatePath(p)
		if err != nil {
			return "", err
		}
	}

	include := ""
	if inc, ok := args["include"].(string); ok {
		include = inc
	}

	// Simple grep implementation using filepath.Walk
	var result []string
	err := filepath.Walk(basePath, func(path string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() {
			return nil
		}

		if include != "" {
			matched, _ := filepath.Match(include, info.Name())
			if !matched {
				return nil
			}
		}

		data, err := os.ReadFile(path)
		if err != nil {
			return nil
		}

		content := string(data)
		if strings.Contains(content, pattern) {
			rel, _ := filepath.Rel(t.base.workingDir, path)
			lines := strings.Split(content, "\n")
			for i, line := range lines {
				if strings.Contains(line, pattern) {
					result = append(result, fmt.Sprintf("%s:%d: %s", rel, i+1, strings.TrimSpace(line)))
					break // Just first match per file
				}
			}
		}
		return nil
	})

	if err != nil {
		return "", err
	}

	if len(result) == 0 {
		return "No matches found", nil
	}

	return strings.Join(result, "\n"), nil
}

// BashTool
type BashTool struct {
	base *Registry
}

func (t *BashTool) Name() string        { return "bash" }
func (t *BashTool) Description() string { return "Execute a shell command" }
func (t *BashTool) Parameters() map[string]interface{} {
	return map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"command": map[string]interface{}{
				"type":        "string",
				"description": "Command to execute",
			},
			"timeout": map[string]interface{}{
				"type":        "integer",
				"description": "Timeout in seconds",
			},
		},
		"required": []string{"command"},
	}
}

func (t *BashTool) Execute(ctx context.Context, args map[string]interface{}) (string, error) {
	command := args["command"].(string)
	timeout := 120
	if to, ok := args["timeout"].(float64); ok {
		timeout = int(to)
	}

	// Simple execution - in production, use proper shell parsing
	ctx, cancel := context.WithTimeout(ctx, time.Duration(timeout)*time.Second)
	defer cancel()

	cmd := exec.CommandContext(ctx, "sh", "-c", command)
	cmd.Dir = t.base.workingDir

	output, err := cmd.CombinedOutput()
	result := string(output)

	if err != nil {
		return result, fmt.Errorf("command failed: %w", err)
	}

	return result, nil
}

// GitTool
type GitTool struct {
	base *Registry
}

func (t *GitTool) Name() string        { return "git" }
func (t *GitTool) Description() string { return "Execute git commands" }
func (t *GitTool) Parameters() map[string]interface{} {
	return map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"command": map[string]interface{}{
				"type":        "string",
				"description": "Git subcommand (status, diff, log, add, commit, push, pull, etc.)",
			},
			"args": map[string]interface{}{
				"type":        "array",
				"items":       map[string]interface{}{"type": "string"},
				"description": "Additional arguments",
			},
		},
		"required": []string{"command"},
	}
}

func (t *GitTool) Execute(ctx context.Context, args map[string]interface{}) (string, error) {
	command := args["command"].(string)
	var gitArgs []string
	if a, ok := args["args"].([]interface{}); ok {
		for _, arg := range a {
			if s, ok := arg.(string); ok {
				gitArgs = append(gitArgs, s)
			}
		}
	}

	cmd := exec.CommandContext(ctx, "git", append([]string{command}, gitArgs...)...)
	cmd.Dir = t.base.workingDir

	output, err := cmd.CombinedOutput()
	result := string(output)

	if err != nil {
		return result, fmt.Errorf("git command failed: %w", err)
	}

	return result, nil
}