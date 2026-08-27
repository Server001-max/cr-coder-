package session

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/Server001-max/cr-coder/internal/config"
)

type Session struct {
	config     *config.SessionConfig
	history    []HistoryEntry
	checkpoints map[string]*Checkpoint
	mu         sync.RWMutex
}

type HistoryEntry struct {
	Timestamp time.Time `json:"timestamp"`
	Role      string    `json:"role"`
	Content   string    `json:"content"`
	Tokens    int       `json:"tokens,omitempty"`
}

type Checkpoint struct {
	ID        string            `json:"id"`
	Timestamp time.Time         `json:"timestamp"`
	Messages  []HistoryEntry    `json:"messages"`
	WorkingDir string           `json:"working_dir"`
	Metadata  map[string]string `json:"metadata,omitempty"`
}

func New(cfg *config.SessionConfig) *Session {
	s := &Session{
		config:      cfg,
		history:     []HistoryEntry{},
		checkpoints: make(map[string]*Checkpoint),
	}
	s.loadHistory()
	return s
}

func (s *Session) loadHistory() {
	if !s.config.SaveHistory {
		return
	}

	data, err := os.ReadFile(s.config.HistoryPath)
	if err != nil {
		return
	}

	var entries []HistoryEntry
	if err := json.Unmarshal(data, &entries); err != nil {
		return
	}

	// Limit history size
	if len(entries) > s.config.MaxHistorySize {
		entries = entries[len(entries)-s.config.MaxHistorySize:]
	}
	s.history = entries
}

func (s *Session) Save() error {
	if !s.config.SaveHistory {
		return nil
	}

	data, err := json.MarshalIndent(s.history, "", "  ")
	if err != nil {
		return err
	}

	// Ensure directory exists
	dir := filepath.Dir(s.config.HistoryPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}

	return os.WriteFile(s.config.HistoryPath, data, 0644)
}

func (s *Session) AddEntry(role, content string, tokens int) {
	s.mu.Lock()
	defer s.mu.Unlock()

	entry := HistoryEntry{
		Timestamp: time.Now(),
		Role:      role,
		Content:   content,
		Tokens:    tokens,
	}

	s.history = append(s.history, entry)

	// Trim if needed
	if len(s.history) > s.config.MaxHistorySize {
		s.history = s.history[len(s.history)-s.config.MaxHistorySize:]
	}

	// Auto-save periodically
	if len(s.history)%10 == 0 {
		go s.Save()
	}
}

func (s *Session) GetHistory(limit int) []HistoryEntry {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if limit <= 0 || limit > len(s.history) {
		limit = len(s.history)
	}

	start := len(s.history) - limit
	result := make([]HistoryEntry, limit)
	copy(result, s.history[start:])
	return result
}

func (s *Session) ClearHistory() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.history = []HistoryEntry{}
	s.Save()
}

func (s *Session) CreateCheckpoint(id string, metadata map[string]string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	wd, _ := os.Getwd()

	cp := &Checkpoint{
		ID:         id,
		Timestamp:  time.Now(),
		Messages:   make([]HistoryEntry, len(s.history)),
		WorkingDir: wd,
		Metadata:   metadata,
	}
	copy(cp.Messages, s.history)

	s.checkpoints[id] = cp
	return s.saveCheckpoint(cp)
}

func (s *Session) saveCheckpoint(cp *Checkpoint) error {
	if !s.config.AutoCheckpoint {
		return nil
	}

	data, err := json.MarshalIndent(cp, "", "  ")
	if err != nil {
		return err
	}

	dir := s.config.CheckpointDir
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}

	path := filepath.Join(dir, cp.ID+".json")
	return os.WriteFile(path, data, 0644)
}

func (s *Session) LoadCheckpoint(id string) (*Checkpoint, error) {
	s.mu.RLock()
	if cp, ok := s.checkpoints[id]; ok {
		s.mu.RUnlock()
		return cp, nil
	}
	s.mu.RUnlock()

	// Try loading from disk
	path := filepath.Join(s.config.CheckpointDir, id+".json")
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var cp Checkpoint
	if err := json.Unmarshal(data, &cp); err != nil {
		return nil, err
	}

	s.mu.Lock()
	s.checkpoints[id] = &cp
	s.mu.Unlock()

	return &cp, nil
}

func (s *Session) ListCheckpoints() []string {
	s.mu.RLock()
	defer s.mu.RUnlock()

	ids := make([]string, 0, len(s.checkpoints))
	for id := range s.checkpoints {
		ids = append(ids, id)
	}
	return ids
}

func (s *Session) RestoreCheckpoint(id string) error {
	cp, err := s.LoadCheckpoint(id)
	if err != nil {
		return err
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	s.history = make([]HistoryEntry, len(cp.Messages))
	copy(s.history, cp.Messages)

	return s.Save()
}

func (s *Session) GetStats() map[string]interface{} {
	s.mu.RLock()
	defer s.mu.RUnlock()

	totalTokens := 0
	for _, e := range s.history {
		totalTokens += e.Tokens
	}

	return map[string]interface{}{
		"total_entries":  len(s.history),
		"total_tokens":   totalTokens,
		"checkpoints":    len(s.checkpoints),
		"history_file":   s.config.HistoryPath,
		"checkpoint_dir": s.config.CheckpointDir,
	}
}

func (s *Session) Export(format string) (string, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	switch format {
	case "json":
		data, _ := json.MarshalIndent(s.history, "", "  ")
		return string(data), nil
	case "markdown":
		var result string
		for _, e := range s.history {
			result += fmt.Sprintf("## %s (%s)\n\n%s\n\n---\n\n", e.Role, e.Timestamp.Format(time.RFC3339), e.Content)
		}
		return result, nil
	default:
		return "", fmt.Errorf("unsupported format: %s", format)
	}
}