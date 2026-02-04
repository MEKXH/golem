# Golem Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Build a lightweight Go AI assistant framework with Eino integration, supporting multi-provider LLM, tool execution, and Telegram channel.

**Architecture:** Message bus decouples channels from agent loop. Agent loop iterates LLM calls and tool executions until completion. Eino provides ChatModel and Tool interfaces.

**Tech Stack:** Go 1.21+, Eino/Eino-ext, Cobra CLI, Viper config, telegram-bot-api

---

## Phase 1: Project Skeleton

### Task 1: Initialize Go Module

**Files:**
- Create: `go.mod`
- Create: `cmd/golem/main.go`

**Step 1: Initialize go module**

Run:
```bash
cd D:\Work\Self_Projects\OpenSource\golem
go mod init github.com/MEKXH/golem
```

**Step 2: Create minimal main.go**

```go
// cmd/golem/main.go
package main

import "fmt"

func main() {
    fmt.Println("Golem v0.1.0")
}
```

**Step 3: Verify it compiles**

Run: `go build ./cmd/golem`
Expected: No errors, produces `golem.exe`

**Step 4: Commit**

```bash
git add go.mod cmd/
git commit -m "init: create go module and main entry"
```

---

### Task 2: Add Core Dependencies

**Files:**
- Modify: `go.mod`

**Step 1: Add dependencies**

Run:
```bash
go get github.com/spf13/cobra@latest
go get github.com/spf13/viper@latest
go get github.com/cloudwego/eino@latest
go get github.com/cloudwego/eino-ext/components/model/openai@latest
go get github.com/cloudwego/eino-ext/components/model/claude@latest
go get github.com/cloudwego/eino-ext/components/model/ollama@latest
go get github.com/go-telegram-bot-api/telegram-bot-api/v5@latest
```

**Step 2: Tidy modules**

Run: `go mod tidy`

**Step 3: Commit**

```bash
git add go.mod go.sum
git commit -m "deps: add core dependencies"
```

---

## Phase 2: Configuration System

### Task 3: Config Structs

**Files:**
- Create: `internal/config/config.go`
- Create: `internal/config/config_test.go`

**Step 1: Write the failing test**

```go
// internal/config/config_test.go
package config

import (
    "testing"
)

func TestDefaultConfig(t *testing.T) {
    cfg := DefaultConfig()

    if cfg.Agents.Defaults.MaxToolIterations != 20 {
        t.Errorf("expected MaxToolIterations=20, got %d", cfg.Agents.Defaults.MaxToolIterations)
    }
    if cfg.Agents.Defaults.Temperature != 0.7 {
        t.Errorf("expected Temperature=0.7, got %f", cfg.Agents.Defaults.Temperature)
    }
    if cfg.Gateway.Port != 18790 {
        t.Errorf("expected Port=18790, got %d", cfg.Gateway.Port)
    }
}
```

**Step 2: Run test to verify it fails**

Run: `go test ./internal/config/... -v`
Expected: FAIL - package not found

**Step 3: Write implementation**

```go
// internal/config/config.go
package config

import (
    "os"
    "path/filepath"
)

// Config root configuration
type Config struct {
    Agents    AgentsConfig    `mapstructure:"agents"`
    Channels  ChannelsConfig  `mapstructure:"channels"`
    Providers ProvidersConfig `mapstructure:"providers"`
    Gateway   GatewayConfig   `mapstructure:"gateway"`
    Tools     ToolsConfig     `mapstructure:"tools"`
}

// AgentsConfig agent settings
type AgentsConfig struct {
    Defaults AgentDefaults `mapstructure:"defaults"`
}

// AgentDefaults default agent parameters
type AgentDefaults struct {
    Workspace         string  `mapstructure:"workspace"`
    Model             string  `mapstructure:"model"`
    MaxTokens         int     `mapstructure:"max_tokens"`
    Temperature       float64 `mapstructure:"temperature"`
    MaxToolIterations int     `mapstructure:"max_tool_iterations"`
}

// ChannelsConfig channel settings
type ChannelsConfig struct {
    Telegram TelegramConfig `mapstructure:"telegram"`
}

// TelegramConfig telegram bot settings
type TelegramConfig struct {
    Enabled   bool     `mapstructure:"enabled"`
    Token     string   `mapstructure:"token"`
    AllowFrom []string `mapstructure:"allow_from"`
}

// ProvidersConfig LLM provider settings
type ProvidersConfig struct {
    OpenRouter ProviderConfig `mapstructure:"openrouter"`
    Claude     ProviderConfig `mapstructure:"claude"`
    OpenAI     ProviderConfig `mapstructure:"openai"`
    DeepSeek   ProviderConfig `mapstructure:"deepseek"`
    Gemini     ProviderConfig `mapstructure:"gemini"`
    Ark        ProviderConfig `mapstructure:"ark"`
    Qianfan    ProviderConfig `mapstructure:"qianfan"`
    Qwen       ProviderConfig `mapstructure:"qwen"`
    Ollama     ProviderConfig `mapstructure:"ollama"`
}

// ProviderConfig single provider settings
type ProviderConfig struct {
    APIKey    string `mapstructure:"api_key"`
    SecretKey string `mapstructure:"secret_key"`
    BaseURL   string `mapstructure:"base_url"`
}

// GatewayConfig server settings
type GatewayConfig struct {
    Host string `mapstructure:"host"`
    Port int    `mapstructure:"port"`
}

// ToolsConfig tool settings
type ToolsConfig struct {
    Web  WebToolsConfig `mapstructure:"web"`
    Exec ExecToolConfig `mapstructure:"exec"`
}

// WebToolsConfig web tool settings
type WebToolsConfig struct {
    Search WebSearchConfig `mapstructure:"search"`
}

// WebSearchConfig brave search settings
type WebSearchConfig struct {
    APIKey     string `mapstructure:"api_key"`
    MaxResults int    `mapstructure:"max_results"`
}

// ExecToolConfig shell exec settings
type ExecToolConfig struct {
    Timeout             int  `mapstructure:"timeout"`
    RestrictToWorkspace bool `mapstructure:"restrict_to_workspace"`
}

// DefaultConfig returns config with sensible defaults
func DefaultConfig() *Config {
    homeDir, _ := os.UserHomeDir()
    return &Config{
        Agents: AgentsConfig{
            Defaults: AgentDefaults{
                Workspace:         filepath.Join(homeDir, ".golem", "workspace"),
                Model:             "anthropic/claude-sonnet-4-5",
                MaxTokens:         8192,
                Temperature:       0.7,
                MaxToolIterations: 20,
            },
        },
        Channels: ChannelsConfig{
            Telegram: TelegramConfig{
                Enabled:   false,
                AllowFrom: []string{},
            },
        },
        Providers: ProvidersConfig{},
        Gateway: GatewayConfig{
            Host: "0.0.0.0",
            Port: 18790,
        },
        Tools: ToolsConfig{
            Web: WebToolsConfig{
                Search: WebSearchConfig{
                    MaxResults: 5,
                },
            },
            Exec: ExecToolConfig{
                Timeout:             60,
                RestrictToWorkspace: false,
            },
        },
    }
}

// ConfigDir returns the golem config directory
func ConfigDir() string {
    homeDir, _ := os.UserHomeDir()
    return filepath.Join(homeDir, ".golem")
}

// ConfigPath returns the config file path
func ConfigPath() string {
    return filepath.Join(ConfigDir(), "config.json")
}

// WorkspacePath returns the expanded workspace path
func (c *Config) WorkspacePath() string {
    if c.Agents.Defaults.Workspace == "" {
        return filepath.Join(ConfigDir(), "workspace")
    }
    // Expand ~ if present
    if len(c.Agents.Defaults.Workspace) > 0 && c.Agents.Defaults.Workspace[0] == '~' {
        homeDir, _ := os.UserHomeDir()
        return filepath.Join(homeDir, c.Agents.Defaults.Workspace[1:])
    }
    return c.Agents.Defaults.Workspace
}
```

**Step 4: Run test to verify it passes**

Run: `go test ./internal/config/... -v`
Expected: PASS

**Step 5: Commit**

```bash
git add internal/config/
git commit -m "feat(config): add configuration structs with defaults"
```

---

### Task 4: Config Loading with Viper

**Files:**
- Modify: `internal/config/config.go`
- Modify: `internal/config/config_test.go`

**Step 1: Write the failing test**

```go
// Add to internal/config/config_test.go

func TestLoadConfig_CreatesDefault(t *testing.T) {
    // Use temp dir
    tmpDir := t.TempDir()
    origHome := os.Getenv("HOME")
    os.Setenv("HOME", tmpDir)
    defer os.Setenv("HOME", origHome)

    cfg, err := Load()
    if err != nil {
        t.Fatalf("Load() error: %v", err)
    }
    if cfg.Agents.Defaults.MaxToolIterations != 20 {
        t.Errorf("expected default MaxToolIterations=20, got %d", cfg.Agents.Defaults.MaxToolIterations)
    }
}
```

**Step 2: Run test to verify it fails**

Run: `go test ./internal/config/... -v`
Expected: FAIL - Load not defined

**Step 3: Add Load function**

```go
// Add to internal/config/config.go

import (
    "encoding/json"
    "os"
    "path/filepath"

    "github.com/spf13/viper"
)

// Load loads config from file or returns defaults
func Load() (*Config, error) {
    cfg := DefaultConfig()

    configPath := ConfigPath()

    // If config doesn't exist, create it
    if _, err := os.Stat(configPath); os.IsNotExist(err) {
        if err := Save(cfg); err != nil {
            return cfg, nil // Return defaults if can't save
        }
        return cfg, nil
    }

    // Load with viper
    v := viper.New()
    v.SetConfigFile(configPath)
    v.SetConfigType("json")

    // Environment variable support
    v.SetEnvPrefix("GOLEM")
    v.AutomaticEnv()

    if err := v.ReadInConfig(); err != nil {
        return cfg, err
    }

    if err := v.Unmarshal(cfg); err != nil {
        return cfg, err
    }

    return cfg, nil
}

// Save saves config to file
func Save(cfg *Config) error {
    configPath := ConfigPath()

    // Ensure directory exists
    if err := os.MkdirAll(filepath.Dir(configPath), 0755); err != nil {
        return err
    }

    data, err := json.MarshalIndent(cfg, "", "  ")
    if err != nil {
        return err
    }

    return os.WriteFile(configPath, data, 0644)
}
```

**Step 4: Run test to verify it passes**

Run: `go test ./internal/config/... -v`
Expected: PASS

**Step 5: Commit**

```bash
git add internal/config/
git commit -m "feat(config): add Load/Save with viper support"
```

---

## Phase 3: Message Bus

### Task 5: Event Types

**Files:**
- Create: `internal/bus/events.go`
- Create: `internal/bus/events_test.go`

**Step 1: Write the failing test**

```go
// internal/bus/events_test.go
package bus

import (
    "testing"
)

func TestInboundMessage_SessionKey(t *testing.T) {
    msg := &InboundMessage{
        Channel: "telegram",
        ChatID:  "12345",
    }

    expected := "telegram:12345"
    if got := msg.SessionKey(); got != expected {
        t.Errorf("SessionKey() = %q, want %q", got, expected)
    }
}
```

**Step 2: Run test to verify it fails**

Run: `go test ./internal/bus/... -v`
Expected: FAIL - package not found

**Step 3: Write implementation**

```go
// internal/bus/events.go
package bus

import "time"

// InboundMessage received from a channel
type InboundMessage struct {
    Channel   string
    SenderID  string
    ChatID    string
    Content   string
    Timestamp time.Time
    Media     []string
    Metadata  map[string]any
}

// SessionKey returns unique session identifier
func (m *InboundMessage) SessionKey() string {
    return m.Channel + ":" + m.ChatID
}

// OutboundMessage to send to a channel
type OutboundMessage struct {
    Channel  string
    ChatID   string
    Content  string
    ReplyTo  string
    Media    []string
    Metadata map[string]any
}
```

**Step 4: Run test to verify it passes**

Run: `go test ./internal/bus/... -v`
Expected: PASS

**Step 5: Commit**

```bash
git add internal/bus/
git commit -m "feat(bus): add message event types"
```

---

### Task 6: Message Bus

**Files:**
- Create: `internal/bus/queue.go`
- Create: `internal/bus/queue_test.go`

**Step 1: Write the failing test**

```go
// internal/bus/queue_test.go
package bus

import (
    "testing"
    "time"
)

func TestMessageBus_PublishConsume(t *testing.T) {
    bus := NewMessageBus(10)

    msg := &InboundMessage{
        Channel: "test",
        Content: "hello",
    }

    bus.PublishInbound(msg)

    select {
    case received := <-bus.Inbound():
        if received.Content != "hello" {
            t.Errorf("got Content=%q, want %q", received.Content, "hello")
        }
    case <-time.After(time.Second):
        t.Fatal("timeout waiting for message")
    }
}
```

**Step 2: Run test to verify it fails**

Run: `go test ./internal/bus/... -v`
Expected: FAIL - NewMessageBus not defined

**Step 3: Write implementation**

```go
// internal/bus/queue.go
package bus

// MessageBus handles message routing between channels and agent
type MessageBus struct {
    inbound  chan *InboundMessage
    outbound chan *OutboundMessage
}

// NewMessageBus creates a new message bus
func NewMessageBus(bufferSize int) *MessageBus {
    return &MessageBus{
        inbound:  make(chan *InboundMessage, bufferSize),
        outbound: make(chan *OutboundMessage, bufferSize),
    }
}

// PublishInbound sends a message to the agent
func (b *MessageBus) PublishInbound(msg *InboundMessage) {
    b.inbound <- msg
}

// Inbound returns the inbound channel for consuming
func (b *MessageBus) Inbound() <-chan *InboundMessage {
    return b.inbound
}

// PublishOutbound sends a message to channels
func (b *MessageBus) PublishOutbound(msg *OutboundMessage) {
    b.outbound <- msg
}

// Outbound returns the outbound channel for consuming
func (b *MessageBus) Outbound() <-chan *OutboundMessage {
    return b.outbound
}

// Close closes both channels
func (b *MessageBus) Close() {
    close(b.inbound)
    close(b.outbound)
}
```

**Step 4: Run test to verify it passes**

Run: `go test ./internal/bus/... -v`
Expected: PASS

**Step 5: Commit**

```bash
git add internal/bus/
git commit -m "feat(bus): add message bus with channels"
```

---

## Phase 4: Session Management

### Task 7: Session Manager

**Files:**
- Create: `internal/session/manager.go`
- Create: `internal/session/manager_test.go`

**Step 1: Write the failing test**

```go
// internal/session/manager_test.go
package session

import (
    "testing"
)

func TestSession_AddMessage(t *testing.T) {
    sess := &Session{Key: "test"}
    sess.AddMessage("user", "hello")
    sess.AddMessage("assistant", "hi there")

    history := sess.GetHistory(10)
    if len(history) != 2 {
        t.Fatalf("expected 2 messages, got %d", len(history))
    }
    if history[0].Role != "user" {
        t.Errorf("expected role=user, got %s", history[0].Role)
    }
}

func TestManager_GetOrCreate(t *testing.T) {
    mgr := NewManager(t.TempDir())

    sess1 := mgr.GetOrCreate("test:123")
    sess2 := mgr.GetOrCreate("test:123")

    if sess1 != sess2 {
        t.Error("expected same session instance")
    }
}
```

**Step 2: Run test to verify it fails**

Run: `go test ./internal/session/... -v`
Expected: FAIL - package not found

**Step 3: Write implementation**

```go
// internal/session/manager.go
package session

import (
    "bufio"
    "encoding/json"
    "os"
    "path/filepath"
    "sync"
    "time"
)

// Message represents a single message in session
type Message struct {
    Role      string    `json:"role"`
    Content   string    `json:"content"`
    Timestamp time.Time `json:"timestamp"`
}

// Session represents a conversation session
type Session struct {
    Key      string
    Messages []*Message
    mu       sync.RWMutex
}

// AddMessage adds a message to the session
func (s *Session) AddMessage(role, content string) {
    s.mu.Lock()
    defer s.mu.Unlock()
    s.Messages = append(s.Messages, &Message{
        Role:      role,
        Content:   content,
        Timestamp: time.Now(),
    })
}

// GetHistory returns the last n messages
func (s *Session) GetHistory(limit int) []*Message {
    s.mu.RLock()
    defer s.mu.RUnlock()

    if limit <= 0 || limit > len(s.Messages) {
        limit = len(s.Messages)
    }
    start := len(s.Messages) - limit
    if start < 0 {
        start = 0
    }

    // Return a copy
    result := make([]*Message, limit)
    copy(result, s.Messages[start:])
    return result
}

// Manager manages sessions
type Manager struct {
    dir      string
    sessions map[string]*Session
    mu       sync.RWMutex
}

// NewManager creates a session manager
func NewManager(baseDir string) *Manager {
    dir := filepath.Join(baseDir, "sessions")
    os.MkdirAll(dir, 0755)
    return &Manager{
        dir:      dir,
        sessions: make(map[string]*Session),
    }
}

// GetOrCreate gets or creates a session
func (m *Manager) GetOrCreate(key string) *Session {
    m.mu.Lock()
    defer m.mu.Unlock()

    if sess, ok := m.sessions[key]; ok {
        return sess
    }

    sess := &Session{Key: key}
    m.loadFromDisk(sess)
    m.sessions[key] = sess
    return sess
}

// Save persists session to disk
func (m *Manager) Save(sess *Session) error {
    sess.mu.RLock()
    defer sess.mu.RUnlock()

    if len(sess.Messages) == 0 {
        return nil
    }

    path := m.sessionPath(sess.Key)
    f, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0644)
    if err != nil {
        return err
    }
    defer f.Close()

    enc := json.NewEncoder(f)
    for _, msg := range sess.Messages {
        if err := enc.Encode(msg); err != nil {
            return err
        }
    }
    return nil
}

func (m *Manager) loadFromDisk(sess *Session) {
    path := m.sessionPath(sess.Key)
    f, err := os.Open(path)
    if err != nil {
        return
    }
    defer f.Close()

    scanner := bufio.NewScanner(f)
    for scanner.Scan() {
        var msg Message
        if err := json.Unmarshal(scanner.Bytes(), &msg); err == nil {
            sess.Messages = append(sess.Messages, &msg)
        }
    }
}

func (m *Manager) sessionPath(key string) string {
    // Replace : with _ for filesystem safety
    safeKey := key
    for _, c := range []string{":", "/", "\\"} {
        safeKey = filepath.Clean(safeKey)
    }
    return filepath.Join(m.dir, key+".jsonl")
}
```

**Step 4: Run test to verify it passes**

Run: `go test ./internal/session/... -v`
Expected: PASS

**Step 5: Commit**

```bash
git add internal/session/
git commit -m "feat(session): add session manager with persistence"
```

---

## Phase 5: Tool System

### Task 8: Tool Registry

**Files:**
- Create: `internal/tools/registry.go`
- Create: `internal/tools/registry_test.go`

**Step 1: Write the failing test**

```go
// internal/tools/registry_test.go
package tools

import (
    "context"
    "testing"

    "github.com/cloudwego/eino/schema"
)

// Mock tool for testing
type mockTool struct{}

func (m *mockTool) Info(ctx context.Context) (*schema.ToolInfo, error) {
    return &schema.ToolInfo{
        Name: "mock_tool",
        Desc: "A mock tool for testing",
    }, nil
}

func (m *mockTool) InvokableRun(ctx context.Context, args string, opts ...any) (string, error) {
    return "mock result", nil
}

func TestRegistry_RegisterAndGet(t *testing.T) {
    reg := NewRegistry()

    err := reg.Register(&mockTool{})
    if err != nil {
        t.Fatalf("Register error: %v", err)
    }

    tool, ok := reg.Get("mock_tool")
    if !ok {
        t.Fatal("expected to find mock_tool")
    }
    if tool == nil {
        t.Fatal("tool is nil")
    }
}
```

**Step 2: Run test to verify it fails**

Run: `go test ./internal/tools/... -v`
Expected: FAIL - package not found

**Step 3: Write implementation**

```go
// internal/tools/registry.go
package tools

import (
    "context"
    "fmt"
    "sync"

    "github.com/cloudwego/eino/components/tool"
    "github.com/cloudwego/eino/schema"
)

// Registry manages tool registration and lookup
type Registry struct {
    tools map[string]tool.InvokableTool
    mu    sync.RWMutex
}

// NewRegistry creates a new tool registry
func NewRegistry() *Registry {
    return &Registry{
        tools: make(map[string]tool.InvokableTool),
    }
}

// Register adds a tool to the registry
func (r *Registry) Register(t tool.InvokableTool) error {
    info, err := t.Info(context.Background())
    if err != nil {
        return fmt.Errorf("failed to get tool info: %w", err)
    }

    r.mu.Lock()
    defer r.mu.Unlock()
    r.tools[info.Name] = t
    return nil
}

// Get retrieves a tool by name
func (r *Registry) Get(name string) (tool.InvokableTool, bool) {
    r.mu.RLock()
    defer r.mu.RUnlock()
    t, ok := r.tools[name]
    return t, ok
}

// GetToolInfos returns all tool schemas for ChatModel binding
func (r *Registry) GetToolInfos(ctx context.Context) ([]*schema.ToolInfo, error) {
    r.mu.RLock()
    defer r.mu.RUnlock()

    infos := make([]*schema.ToolInfo, 0, len(r.tools))
    for _, t := range r.tools {
        info, err := t.Info(ctx)
        if err != nil {
            return nil, err
        }
        infos = append(infos, info)
    }
    return infos, nil
}

// Execute runs a tool by name
func (r *Registry) Execute(ctx context.Context, name string, argsJSON string) (string, error) {
    t, ok := r.Get(name)
    if !ok {
        return "", fmt.Errorf("tool not found: %s", name)
    }
    return t.InvokableRun(ctx, argsJSON)
}

// Names returns all registered tool names
func (r *Registry) Names() []string {
    r.mu.RLock()
    defer r.mu.RUnlock()

    names := make([]string, 0, len(r.tools))
    for name := range r.tools {
        names = append(names, name)
    }
    return names
}
```

**Step 4: Run test to verify it passes**

Run: `go test ./internal/tools/... -v`
Expected: PASS

**Step 5: Commit**

```bash
git add internal/tools/
git commit -m "feat(tools): add tool registry"
```

---

### Task 9: ReadFile Tool

**Files:**
- Create: `internal/tools/filesystem.go`
- Modify: `internal/tools/registry_test.go`

**Step 1: Write the failing test**

```go
// Add to internal/tools/registry_test.go

func TestReadFileTool(t *testing.T) {
    // Create temp file
    tmpDir := t.TempDir()
    testFile := filepath.Join(tmpDir, "test.txt")
    os.WriteFile(testFile, []byte("line1\nline2\nline3"), 0644)

    tool, err := NewReadFileTool()
    if err != nil {
        t.Fatalf("NewReadFileTool error: %v", err)
    }

    ctx := context.Background()
    argsJSON := fmt.Sprintf(`{"path": %q}`, testFile)

    result, err := tool.InvokableRun(ctx, argsJSON)
    if err != nil {
        t.Fatalf("InvokableRun error: %v", err)
    }

    if !strings.Contains(result, "line1") {
        t.Errorf("expected result to contain 'line1', got: %s", result)
    }
}
```

**Step 2: Run test to verify it fails**

Run: `go test ./internal/tools/... -v`
Expected: FAIL - NewReadFileTool not defined

**Step 3: Write implementation**

```go
// internal/tools/filesystem.go
package tools

import (
    "context"
    "os"
    "strings"

    "github.com/cloudwego/eino/components/tool"
    "github.com/cloudwego/eino/components/tool/utils"
)

// ReadFileInput parameters for read_file tool
type ReadFileInput struct {
    Path   string `json:"path" jsonschema:"required,description=Absolute path to the file"`
    Offset int    `json:"offset" jsonschema:"description=Starting line number (0-based)"`
    Limit  int    `json:"limit" jsonschema:"description=Maximum number of lines to read"`
}

// ReadFileOutput result of read_file tool
type ReadFileOutput struct {
    Content    string `json:"content"`
    TotalLines int    `json:"total_lines"`
}

func readFile(ctx context.Context, input *ReadFileInput) (*ReadFileOutput, error) {
    data, err := os.ReadFile(input.Path)
    if err != nil {
        return nil, err
    }

    content := string(data)
    lines := strings.Split(content, "\n")
    totalLines := len(lines)

    // Apply offset and limit
    if input.Offset > 0 {
        if input.Offset >= len(lines) {
            lines = []string{}
        } else {
            lines = lines[input.Offset:]
        }
    }

    if input.Limit > 0 && input.Limit < len(lines) {
        lines = lines[:input.Limit]
    }

    return &ReadFileOutput{
        Content:    strings.Join(lines, "\n"),
        TotalLines: totalLines,
    }, nil
}

// NewReadFileTool creates the read_file tool
func NewReadFileTool() (tool.InvokableTool, error) {
    return utils.InferTool("read_file", "Read the contents of a file", readFile)
}

// WriteFileInput parameters for write_file tool
type WriteFileInput struct {
    Path    string `json:"path" jsonschema:"required,description=Absolute path to the file"`
    Content string `json:"content" jsonschema:"required,description=Content to write"`
}

func writeFile(ctx context.Context, input *WriteFileInput) (string, error) {
    err := os.WriteFile(input.Path, []byte(input.Content), 0644)
    if err != nil {
        return "", err
    }
    return "File written successfully", nil
}

// NewWriteFileTool creates the write_file tool
func NewWriteFileTool() (tool.InvokableTool, error) {
    return utils.InferTool("write_file", "Write content to a file", writeFile)
}

// ListDirInput parameters for list_dir tool
type ListDirInput struct {
    Path string `json:"path" jsonschema:"required,description=Directory path to list"`
}

func listDir(ctx context.Context, input *ListDirInput) ([]string, error) {
    entries, err := os.ReadDir(input.Path)
    if err != nil {
        return nil, err
    }

    var result []string
    for _, entry := range entries {
        name := entry.Name()
        if entry.IsDir() {
            name += "/"
        }
        result = append(result, name)
    }
    return result, nil
}

// NewListDirTool creates the list_dir tool
func NewListDirTool() (tool.InvokableTool, error) {
    return utils.InferTool("list_dir", "List contents of a directory", listDir)
}
```

**Step 4: Run test to verify it passes**

Run: `go test ./internal/tools/... -v`
Expected: PASS

**Step 5: Commit**

```bash
git add internal/tools/
git commit -m "feat(tools): add filesystem tools (read_file, write_file, list_dir)"
```

---

### Task 10: Exec Tool

**Files:**
- Create: `internal/tools/shell.go`
- Modify: `internal/tools/registry_test.go`

**Step 1: Write the failing test**

```go
// Add to internal/tools/registry_test.go

func TestExecTool(t *testing.T) {
    tool, err := NewExecTool(60, false, "")
    if err != nil {
        t.Fatalf("NewExecTool error: %v", err)
    }

    ctx := context.Background()
    // Use a simple cross-platform command
    argsJSON := `{"command": "echo hello"}`

    result, err := tool.InvokableRun(ctx, argsJSON)
    if err != nil {
        t.Fatalf("InvokableRun error: %v", err)
    }

    if !strings.Contains(result, "hello") {
        t.Errorf("expected result to contain 'hello', got: %s", result)
    }
}
```

**Step 2: Run test to verify it fails**

Run: `go test ./internal/tools/... -v`
Expected: FAIL - NewExecTool not defined

**Step 3: Write implementation**

```go
// internal/tools/shell.go
package tools

import (
    "context"
    "fmt"
    "os/exec"
    "runtime"
    "strings"
    "time"

    "github.com/cloudwego/eino/components/tool"
    "github.com/cloudwego/eino/components/tool/utils"
)

// ExecInput parameters for exec tool
type ExecInput struct {
    Command    string `json:"command" jsonschema:"required,description=Shell command to execute"`
    WorkingDir string `json:"working_dir" jsonschema:"description=Working directory for the command"`
}

// ExecOutput result of exec tool
type ExecOutput struct {
    Stdout   string `json:"stdout"`
    Stderr   string `json:"stderr"`
    ExitCode int    `json:"exit_code"`
}

// Dangerous commands to block
var dangerousCommands = []string{
    "rm -rf /",
    "rm -rf ~",
    "mkfs",
    "dd if=",
    ":(){:|:&};:",
    "format c:",
    "del /f /s /q",
}

type execToolImpl struct {
    timeout             time.Duration
    restrictToWorkspace bool
    workspaceDir        string
}

func (e *execToolImpl) execute(ctx context.Context, input *ExecInput) (*ExecOutput, error) {
    // Check for dangerous commands
    cmdLower := strings.ToLower(input.Command)
    for _, dangerous := range dangerousCommands {
        if strings.Contains(cmdLower, dangerous) {
            return &ExecOutput{
                Stderr:   fmt.Sprintf("Blocked dangerous command: %s", dangerous),
                ExitCode: 1,
            }, nil
        }
    }

    // Create command based on OS
    var cmd *exec.Cmd
    if runtime.GOOS == "windows" {
        cmd = exec.CommandContext(ctx, "cmd", "/C", input.Command)
    } else {
        cmd = exec.CommandContext(ctx, "sh", "-c", input.Command)
    }

    // Set working directory
    if input.WorkingDir != "" {
        cmd.Dir = input.WorkingDir
    } else if e.workspaceDir != "" {
        cmd.Dir = e.workspaceDir
    }

    // Set timeout
    timeoutCtx, cancel := context.WithTimeout(ctx, e.timeout)
    defer cancel()
    cmd = exec.CommandContext(timeoutCtx, cmd.Path, cmd.Args[1:]...)
    cmd.Dir = input.WorkingDir

    // Capture output
    var stdout, stderr strings.Builder
    cmd.Stdout = &stdout
    cmd.Stderr = &stderr

    err := cmd.Run()
    exitCode := 0
    if err != nil {
        if exitErr, ok := err.(*exec.ExitError); ok {
            exitCode = exitErr.ExitCode()
        } else {
            return &ExecOutput{
                Stderr:   err.Error(),
                ExitCode: 1,
            }, nil
        }
    }

    return &ExecOutput{
        Stdout:   stdout.String(),
        Stderr:   stderr.String(),
        ExitCode: exitCode,
    }, nil
}

// NewExecTool creates the exec tool
func NewExecTool(timeoutSec int, restrictToWorkspace bool, workspaceDir string) (tool.InvokableTool, error) {
    impl := &execToolImpl{
        timeout:             time.Duration(timeoutSec) * time.Second,
        restrictToWorkspace: restrictToWorkspace,
        workspaceDir:        workspaceDir,
    }
    return utils.InferTool("exec", "Execute a shell command", impl.execute)
}
```

**Step 4: Run test to verify it passes**

Run: `go test ./internal/tools/... -v`
Expected: PASS

**Step 5: Commit**

```bash
git add internal/tools/
git commit -m "feat(tools): add exec shell tool with safety checks"
```

---

## Phase 6: Provider System

### Task 11: Provider Factory

**Files:**
- Create: `internal/provider/provider.go`
- Create: `internal/provider/provider_test.go`

**Step 1: Write the failing test**

```go
// internal/provider/provider_test.go
package provider

import (
    "testing"

    "github.com/MEKXH/golem/internal/config"
)

func TestNewChatModel_NoProvider(t *testing.T) {
    cfg := config.DefaultConfig()

    _, err := NewChatModel(nil, cfg)
    if err == nil {
        t.Error("expected error when no provider configured")
    }
}
```

**Step 2: Run test to verify it fails**

Run: `go test ./internal/provider/... -v`
Expected: FAIL - package not found

**Step 3: Write implementation**

```go
// internal/provider/provider.go
package provider

import (
    "context"
    "fmt"

    "github.com/cloudwego/eino/components/model"
    "github.com/cloudwego/eino-ext/components/model/openai"
    "github.com/MEKXH/golem/internal/config"
)

// NewChatModel creates a ChatModel based on configuration
func NewChatModel(ctx context.Context, cfg *config.Config) (model.ChatModel, error) {
    p := cfg.Providers
    d := cfg.Agents.Defaults

    // Check providers in priority order
    switch {
    case p.OpenRouter.APIKey != "":
        return newOpenRouterModel(ctx, p.OpenRouter, d)
    case p.Claude.APIKey != "":
        return newClaudeModel(ctx, p.Claude, d)
    case p.OpenAI.APIKey != "":
        return newOpenAIModel(ctx, p.OpenAI, d)
    case p.DeepSeek.APIKey != "":
        return newDeepSeekModel(ctx, p.DeepSeek, d)
    case p.Ollama.BaseURL != "":
        return newOllamaModel(ctx, p.Ollama, d)
    default:
        return nil, fmt.Errorf("no provider configured: set api_key for at least one provider")
    }
}

func newOpenRouterModel(ctx context.Context, p config.ProviderConfig, d config.AgentDefaults) (model.ChatModel, error) {
    return openai.NewChatModel(ctx, &openai.ChatModelConfig{
        Model:       d.Model,
        APIKey:      p.APIKey,
        BaseURL:     "https://openrouter.ai/api/v1",
        Temperature: toFloat32Ptr(d.Temperature),
        MaxTokens:   toIntPtr(d.MaxTokens),
    })
}

func newClaudeModel(ctx context.Context, p config.ProviderConfig, d config.AgentDefaults) (model.ChatModel, error) {
    // Claude uses OpenAI-compatible API via eino-ext
    return openai.NewChatModel(ctx, &openai.ChatModelConfig{
        Model:       d.Model,
        APIKey:      p.APIKey,
        BaseURL:     "https://api.anthropic.com/v1",
        Temperature: toFloat32Ptr(d.Temperature),
        MaxTokens:   toIntPtr(d.MaxTokens),
    })
}

func newOpenAIModel(ctx context.Context, p config.ProviderConfig, d config.AgentDefaults) (model.ChatModel, error) {
    cfg := &openai.ChatModelConfig{
        Model:       d.Model,
        APIKey:      p.APIKey,
        Temperature: toFloat32Ptr(d.Temperature),
        MaxTokens:   toIntPtr(d.MaxTokens),
    }
    if p.BaseURL != "" {
        cfg.BaseURL = p.BaseURL
    }
    return openai.NewChatModel(ctx, cfg)
}

func newDeepSeekModel(ctx context.Context, p config.ProviderConfig, d config.AgentDefaults) (model.ChatModel, error) {
    return openai.NewChatModel(ctx, &openai.ChatModelConfig{
        Model:       d.Model,
        APIKey:      p.APIKey,
        BaseURL:     "https://api.deepseek.com/v1",
        Temperature: toFloat32Ptr(d.Temperature),
        MaxTokens:   toIntPtr(d.MaxTokens),
    })
}

func newOllamaModel(ctx context.Context, p config.ProviderConfig, d config.AgentDefaults) (model.ChatModel, error) {
    baseURL := p.BaseURL
    if baseURL == "" {
        baseURL = "http://localhost:11434"
    }
    return openai.NewChatModel(ctx, &openai.ChatModelConfig{
        Model:       d.Model,
        BaseURL:     baseURL + "/v1",
        Temperature: toFloat32Ptr(d.Temperature),
        MaxTokens:   toIntPtr(d.MaxTokens),
    })
}

func toFloat32Ptr(f float64) *float32 {
    v := float32(f)
    return &v
}

func toIntPtr(i int) *int {
    return &i
}
```

**Step 4: Run test to verify it passes**

Run: `go test ./internal/provider/... -v`
Expected: PASS

**Step 5: Commit**

```bash
git add internal/provider/
git commit -m "feat(provider): add ChatModel factory with multi-provider support"
```

---

## Phase 7: Agent Loop

### Task 12: Agent Loop Core

**Files:**
- Create: `internal/agent/loop.go`
- Create: `internal/agent/loop_test.go`

**Step 1: Write the failing test**

```go
// internal/agent/loop_test.go
package agent

import (
    "context"
    "testing"

    "github.com/MEKXH/golem/internal/bus"
    "github.com/MEKXH/golem/internal/config"
)

func TestNewLoop(t *testing.T) {
    cfg := config.DefaultConfig()
    msgBus := bus.NewMessageBus(10)

    loop := NewLoop(cfg, msgBus, nil) // nil model for now
    if loop == nil {
        t.Fatal("expected non-nil Loop")
    }
    if loop.maxIterations != 20 {
        t.Errorf("expected maxIterations=20, got %d", loop.maxIterations)
    }
}
```

**Step 2: Run test to verify it fails**

Run: `go test ./internal/agent/... -v`
Expected: FAIL - package not found

**Step 3: Write implementation**

```go
// internal/agent/loop.go
package agent

import (
    "context"
    "encoding/json"
    "log/slog"

    "github.com/cloudwego/eino/components/model"
    "github.com/cloudwego/eino/schema"
    "github.com/MEKXH/golem/internal/bus"
    "github.com/MEKXH/golem/internal/config"
    "github.com/MEKXH/golem/internal/session"
    "github.com/MEKXH/golem/internal/tools"
)

// Loop is the main agent processing loop
type Loop struct {
    bus           *bus.MessageBus
    model         model.ChatModel
    tools         *tools.Registry
    sessions      *session.Manager
    context       *ContextBuilder
    maxIterations int
    workspacePath string
}

// NewLoop creates a new agent loop
func NewLoop(cfg *config.Config, msgBus *bus.MessageBus, chatModel model.ChatModel) *Loop {
    workspacePath := cfg.WorkspacePath()
    return &Loop{
        bus:           msgBus,
        model:         chatModel,
        tools:         tools.NewRegistry(),
        sessions:      session.NewManager(workspacePath),
        context:       NewContextBuilder(workspacePath),
        maxIterations: cfg.Agents.Defaults.MaxToolIterations,
        workspacePath: workspacePath,
    }
}

// RegisterDefaultTools registers all built-in tools
func (l *Loop) RegisterDefaultTools(cfg *config.Config) error {
    toolFns := []func() (interface{}, error){
        func() (interface{}, error) { return tools.NewReadFileTool() },
        func() (interface{}, error) { return tools.NewWriteFileTool() },
        func() (interface{}, error) { return tools.NewListDirTool() },
        func() (interface{}, error) {
            return tools.NewExecTool(
                cfg.Tools.Exec.Timeout,
                cfg.Tools.Exec.RestrictToWorkspace,
                l.workspacePath,
            )
        },
    }

    for _, fn := range toolFns {
        t, err := fn()
        if err != nil {
            return err
        }
        if invokable, ok := t.(interface {
            Info(context.Context) (*schema.ToolInfo, error)
            InvokableRun(context.Context, string, ...any) (string, error)
        }); ok {
            if err := l.tools.Register(invokable); err != nil {
                return err
            }
        }
    }
    return nil
}

// Run starts the agent loop
func (l *Loop) Run(ctx context.Context) error {
    // Bind tools to model
    if l.model != nil {
        toolInfos, err := l.tools.GetToolInfos(ctx)
        if err != nil {
            return err
        }
        if binder, ok := l.model.(interface{ BindTools([]*schema.ToolInfo) error }); ok {
            if err := binder.BindTools(toolInfos); err != nil {
                return err
            }
        }
    }

    slog.Info("agent loop started")

    for {
        select {
        case <-ctx.Done():
            return ctx.Err()
        case msg := <-l.bus.Inbound():
            resp, err := l.processMessage(ctx, msg)
            if err != nil {
                slog.Error("process message failed", "error", err)
                l.bus.PublishOutbound(&bus.OutboundMessage{
                    Channel: msg.Channel,
                    ChatID:  msg.ChatID,
                    Content: "Error: " + err.Error(),
                })
                continue
            }
            if resp != nil {
                l.bus.PublishOutbound(resp)
            }
        }
    }
}

func (l *Loop) processMessage(ctx context.Context, msg *bus.InboundMessage) (*bus.OutboundMessage, error) {
    slog.Info("processing message", "channel", msg.Channel, "sender", msg.SenderID)

    // Get or create session
    sess := l.sessions.GetOrCreate(msg.SessionKey())

    // Build messages
    messages := l.context.BuildMessages(sess.GetHistory(50), msg.Content, msg.Media)

    // Agent loop
    var finalContent string

    for i := 0; i < l.maxIterations; i++ {
        if l.model == nil {
            finalContent = "No model configured"
            break
        }

        resp, err := l.model.Generate(ctx, messages)
        if err != nil {
            return nil, err
        }

        // No tool calls - we're done
        if len(resp.ToolCalls) == 0 {
            finalContent = resp.Content
            break
        }

        // Add assistant message
        messages = append(messages, resp)

        // Execute tools
        for _, tc := range resp.ToolCalls {
            slog.Debug("executing tool", "name", tc.Function.Name)

            result, err := l.tools.Execute(ctx, tc.Function.Name, tc.Function.Arguments)
            if err != nil {
                result = "Error: " + err.Error()
            }

            messages = append(messages, &schema.Message{
                Role:       schema.Tool,
                Content:    result,
                ToolCallID: tc.ID,
            })
        }
    }

    if finalContent == "" {
        finalContent = "Processing complete."
    }

    // Save session
    sess.AddMessage("user", msg.Content)
    sess.AddMessage("assistant", finalContent)
    l.sessions.Save(sess)

    return &bus.OutboundMessage{
        Channel: msg.Channel,
        ChatID:  msg.ChatID,
        Content: finalContent,
    }, nil
}

// ProcessDirect processes a message directly (for CLI)
func (l *Loop) ProcessDirect(ctx context.Context, content string) (string, error) {
    msg := &bus.InboundMessage{
        Channel:  "cli",
        SenderID: "user",
        ChatID:   "direct",
        Content:  content,
    }

    resp, err := l.processMessage(ctx, msg)
    if err != nil {
        return "", err
    }
    return resp.Content, nil
}
```

**Step 4: Run test to verify it passes**

Run: `go test ./internal/agent/... -v`
Expected: PASS

**Step 5: Commit**

```bash
git add internal/agent/
git commit -m "feat(agent): add agent loop core"
```

---

### Task 13: Context Builder

**Files:**
- Create: `internal/agent/context.go`
- Modify: `internal/agent/loop_test.go`

**Step 1: Write the failing test**

```go
// Add to internal/agent/loop_test.go

func TestContextBuilder_BuildSystemPrompt(t *testing.T) {
    tmpDir := t.TempDir()
    cb := NewContextBuilder(tmpDir)

    prompt := cb.BuildSystemPrompt()
    if !strings.Contains(prompt, "Golem") {
        t.Error("expected system prompt to contain 'Golem'")
    }
}
```

**Step 2: Run test to verify it fails**

Run: `go test ./internal/agent/... -v`
Expected: FAIL - NewContextBuilder not defined

**Step 3: Write implementation**

```go
// internal/agent/context.go
package agent

import (
    "os"
    "path/filepath"
    "strings"

    "github.com/cloudwego/eino/schema"
    "github.com/MEKXH/golem/internal/session"
)

// ContextBuilder builds LLM context
type ContextBuilder struct {
    workspacePath string
}

// NewContextBuilder creates a context builder
func NewContextBuilder(workspacePath string) *ContextBuilder {
    return &ContextBuilder{workspacePath: workspacePath}
}

// BuildSystemPrompt assembles the system prompt
func (c *ContextBuilder) BuildSystemPrompt() string {
    var parts []string

    // Core identity
    parts = append(parts, c.coreIdentity())

    // Bootstrap files
    bootstrapFiles := []string{"IDENTITY.md", "SOUL.md", "USER.md", "TOOLS.md", "AGENTS.md"}
    for _, name := range bootstrapFiles {
        if content := c.readWorkspaceFile(name); content != "" {
            parts = append(parts, "## "+strings.TrimSuffix(name, ".md")+"\n"+content)
        }
    }

    // Long-term memory
    if mem := c.readWorkspaceFile(filepath.Join("memory", "MEMORY.md")); mem != "" {
        parts = append(parts, "## Long-term Memory\n"+mem)
    }

    return strings.Join(parts, "\n\n")
}

func (c *ContextBuilder) coreIdentity() string {
    return `You are Golem, a personal AI assistant.
You have access to tools for file operations, shell commands, and more.
Be helpful, concise, and proactive. Use tools when needed to accomplish tasks.`
}

func (c *ContextBuilder) readWorkspaceFile(name string) string {
    path := filepath.Join(c.workspacePath, name)
    data, err := os.ReadFile(path)
    if err != nil {
        return ""
    }
    return strings.TrimSpace(string(data))
}

// BuildMessages constructs the full message list
func (c *ContextBuilder) BuildMessages(history []*session.Message, current string, media []string) []*schema.Message {
    messages := make([]*schema.Message, 0, len(history)+2)

    // System prompt
    messages = append(messages, &schema.Message{
        Role:    schema.System,
        Content: c.BuildSystemPrompt(),
    })

    // History
    for _, h := range history {
        role := schema.User
        if h.Role == "assistant" {
            role = schema.Assistant
        }
        messages = append(messages, &schema.Message{
            Role:    role,
            Content: h.Content,
        })
    }

    // Current message
    messages = append(messages, &schema.Message{
        Role:    schema.User,
        Content: current,
    })

    return messages
}
```

**Step 4: Run test to verify it passes**

Run: `go test ./internal/agent/... -v`
Expected: PASS

**Step 5: Commit**

```bash
git add internal/agent/
git commit -m "feat(agent): add context builder"
```

---

## Phase 8: CLI Commands

### Task 14: Root Command with Cobra

**Files:**
- Modify: `cmd/golem/main.go`
- Create: `cmd/golem/commands/root.go`

**Step 1: Create root command**

```go
// cmd/golem/commands/root.go
package commands

import (
    "github.com/spf13/cobra"
)

// NewRootCmd creates the root command
func NewRootCmd() *cobra.Command {
    cmd := &cobra.Command{
        Use:   "golem",
        Short: "Golem - Lightweight AI Assistant",
        Long:  `Golem is a lightweight personal AI assistant built with Go and Eino.`,
    }

    cmd.AddCommand(
        NewInitCmd(),
        NewChatCmd(),
        NewRunCmd(),
        NewStatusCmd(),
    )

    return cmd
}
```

**Step 2: Update main.go**

```go
// cmd/golem/main.go
package main

import (
    "os"

    "github.com/MEKXH/golem/cmd/golem/commands"
)

func main() {
    if err := commands.NewRootCmd().Execute(); err != nil {
        os.Exit(1)
    }
}
```

**Step 3: Verify it compiles**

Run: `go build ./cmd/golem`
Expected: Compilation error (commands not yet implemented)

**Step 4: Create stub commands**

```go
// cmd/golem/commands/init.go
package commands

import "github.com/spf13/cobra"

func NewInitCmd() *cobra.Command {
    return &cobra.Command{
        Use:   "init",
        Short: "Initialize Golem configuration",
        RunE: func(cmd *cobra.Command, args []string) error {
            // TODO: implement
            return nil
        },
    }
}
```

```go
// cmd/golem/commands/chat.go
package commands

import "github.com/spf13/cobra"

func NewChatCmd() *cobra.Command {
    return &cobra.Command{
        Use:   "chat [message]",
        Short: "Chat with Golem",
        RunE: func(cmd *cobra.Command, args []string) error {
            // TODO: implement
            return nil
        },
    }
}
```

```go
// cmd/golem/commands/run.go
package commands

import "github.com/spf13/cobra"

func NewRunCmd() *cobra.Command {
    return &cobra.Command{
        Use:   "run",
        Short: "Start Golem server (Telegram + scheduled tasks)",
        RunE: func(cmd *cobra.Command, args []string) error {
            // TODO: implement
            return nil
        },
    }
}
```

```go
// cmd/golem/commands/status.go
package commands

import "github.com/spf13/cobra"

func NewStatusCmd() *cobra.Command {
    return &cobra.Command{
        Use:   "status",
        Short: "Show Golem configuration status",
        RunE: func(cmd *cobra.Command, args []string) error {
            // TODO: implement
            return nil
        },
    }
}
```

**Step 5: Verify it compiles**

Run: `go build ./cmd/golem && ./golem --help`
Expected: Shows help with init, chat, run, status commands

**Step 6: Commit**

```bash
git add cmd/
git commit -m "feat(cli): add cobra root command and stubs"
```

---

### Task 15: Init Command

**Files:**
- Modify: `cmd/golem/commands/init.go`

**Step 1: Implement init command**

```go
// cmd/golem/commands/init.go
package commands

import (
    "fmt"
    "os"
    "path/filepath"

    "github.com/spf13/cobra"
    "github.com/MEKXH/golem/internal/config"
)

func NewInitCmd() *cobra.Command {
    return &cobra.Command{
        Use:   "init",
        Short: "Initialize Golem configuration",
        RunE:  runInit,
    }
}

func runInit(cmd *cobra.Command, args []string) error {
    configPath := config.ConfigPath()

    // Check if already initialized
    if _, err := os.Stat(configPath); err == nil {
        fmt.Printf("Config already exists: %s\n", configPath)
        return nil
    }

    // Create default config
    cfg := config.DefaultConfig()

    // Ensure directories exist
    dirs := []string{
        config.ConfigDir(),
        cfg.WorkspacePath(),
        filepath.Join(cfg.WorkspacePath(), "memory"),
        filepath.Join(cfg.WorkspacePath(), "skills"),
        filepath.Join(config.ConfigDir(), "sessions"),
    }

    for _, dir := range dirs {
        if err := os.MkdirAll(dir, 0755); err != nil {
            return fmt.Errorf("failed to create directory %s: %w", dir, err)
        }
    }

    // Save config
    if err := config.Save(cfg); err != nil {
        return fmt.Errorf("failed to save config: %w", err)
    }

    // Create default workspace files
    workspaceFiles := map[string]string{
        "IDENTITY.md": "# Identity\n\nYou are Golem, a helpful AI assistant.",
        "SOUL.md":     "# Soul\n\nBe helpful, concise, and proactive.",
        "USER.md":     "# User\n\nInformation about the user goes here.",
        "AGENTS.md":   "# Agents\n\nAgent-specific instructions go here.",
    }

    for name, content := range workspaceFiles {
        path := filepath.Join(cfg.WorkspacePath(), name)
        if _, err := os.Stat(path); os.IsNotExist(err) {
            os.WriteFile(path, []byte(content), 0644)
        }
    }

    fmt.Printf("Golem initialized!\n")
    fmt.Printf("Config: %s\n", configPath)
    fmt.Printf("Workspace: %s\n", cfg.WorkspacePath())
    fmt.Printf("\nNext steps:\n")
    fmt.Printf("1. Edit %s to add your API keys\n", configPath)
    fmt.Printf("2. Run 'golem chat' to start chatting\n")

    return nil
}
```

**Step 2: Test manually**

Run: `go build ./cmd/golem && ./golem init`
Expected: Creates config and workspace directories

**Step 3: Commit**

```bash
git add cmd/golem/commands/init.go
git commit -m "feat(cli): implement init command"
```

---

### Task 16: Chat Command

**Files:**
- Modify: `cmd/golem/commands/chat.go`

**Step 1: Implement chat command**

```go
// cmd/golem/commands/chat.go
package commands

import (
    "bufio"
    "context"
    "fmt"
    "os"
    "strings"

    "github.com/spf13/cobra"
    "github.com/MEKXH/golem/internal/agent"
    "github.com/MEKXH/golem/internal/bus"
    "github.com/MEKXH/golem/internal/config"
    "github.com/MEKXH/golem/internal/provider"
)

func NewChatCmd() *cobra.Command {
    return &cobra.Command{
        Use:   "chat [message]",
        Short: "Chat with Golem",
        RunE:  runChat,
    }
}

func runChat(cmd *cobra.Command, args []string) error {
    ctx := context.Background()

    // Load config
    cfg, err := config.Load()
    if err != nil {
        return fmt.Errorf("failed to load config: %w", err)
    }

    // Create chat model
    model, err := provider.NewChatModel(ctx, cfg)
    if err != nil {
        fmt.Printf("Warning: %v\n", err)
        fmt.Println("Running without LLM (tools only mode)")
        model = nil
    }

    // Create message bus and agent
    msgBus := bus.NewMessageBus(10)
    loop := agent.NewLoop(cfg, msgBus, model)

    // Register tools
    if err := loop.RegisterDefaultTools(cfg); err != nil {
        return fmt.Errorf("failed to register tools: %w", err)
    }

    // Single message mode
    if len(args) > 0 {
        message := strings.Join(args, " ")
        resp, err := loop.ProcessDirect(ctx, message)
        if err != nil {
            return err
        }
        fmt.Println(resp)
        return nil
    }

    // Interactive mode
    fmt.Println("Golem ready. Type 'exit' to quit.")
    scanner := bufio.NewScanner(os.Stdin)

    for {
        fmt.Print("\n> ")
        if !scanner.Scan() {
            break
        }

        input := strings.TrimSpace(scanner.Text())
        if input == "exit" || input == "quit" {
            break
        }
        if input == "" {
            continue
        }

        resp, err := loop.ProcessDirect(ctx, input)
        if err != nil {
            fmt.Printf("Error: %v\n", err)
            continue
        }
        fmt.Println(resp)
    }

    return nil
}
```

**Step 2: Test manually**

Run: `go build ./cmd/golem && ./golem chat "hello"`
Expected: Returns response (or "No model configured" if no API key)

**Step 3: Commit**

```bash
git add cmd/golem/commands/chat.go
git commit -m "feat(cli): implement chat command"
```

---

### Task 17: Status Command

**Files:**
- Modify: `cmd/golem/commands/status.go`

**Step 1: Implement status command**

```go
// cmd/golem/commands/status.go
package commands

import (
    "fmt"
    "os"

    "github.com/spf13/cobra"
    "github.com/MEKXH/golem/internal/config"
)

func NewStatusCmd() *cobra.Command {
    return &cobra.Command{
        Use:   "status",
        Short: "Show Golem configuration status",
        RunE:  runStatus,
    }
}

func runStatus(cmd *cobra.Command, args []string) error {
    cfg, err := config.Load()
    if err != nil {
        return fmt.Errorf("failed to load config: %w", err)
    }

    fmt.Println("=== Golem Status ===")
    fmt.Println()

    // Config
    fmt.Printf("Config: %s\n", config.ConfigPath())
    if _, err := os.Stat(config.ConfigPath()); err == nil {
        fmt.Println("  Status: OK")
    } else {
        fmt.Println("  Status: Not found (run 'golem init')")
    }

    // Workspace
    fmt.Printf("\nWorkspace: %s\n", cfg.WorkspacePath())
    if _, err := os.Stat(cfg.WorkspacePath()); err == nil {
        fmt.Println("  Status: OK")
    } else {
        fmt.Println("  Status: Not found")
    }

    // Model
    fmt.Printf("\nModel: %s\n", cfg.Agents.Defaults.Model)

    // Providers
    fmt.Println("\nProviders:")
    providers := map[string]string{
        "OpenRouter": cfg.Providers.OpenRouter.APIKey,
        "Claude":     cfg.Providers.Claude.APIKey,
        "OpenAI":     cfg.Providers.OpenAI.APIKey,
        "DeepSeek":   cfg.Providers.DeepSeek.APIKey,
        "Gemini":     cfg.Providers.Gemini.APIKey,
        "Ollama":     cfg.Providers.Ollama.BaseURL,
    }

    for name, key := range providers {
        status := "Not configured"
        if key != "" {
            status = "Configured"
        }
        fmt.Printf("  %s: %s\n", name, status)
    }

    // Channels
    fmt.Println("\nChannels:")
    fmt.Printf("  Telegram: %v\n", cfg.Channels.Telegram.Enabled)

    return nil
}
```

**Step 2: Test manually**

Run: `go build ./cmd/golem && ./golem status`
Expected: Shows configuration status

**Step 3: Commit**

```bash
git add cmd/golem/commands/status.go
git commit -m "feat(cli): implement status command"
```

---

## Phase 9: Telegram Channel

### Task 18: Channel Interface

**Files:**
- Create: `internal/channel/channel.go`

**Step 1: Write interface**

```go
// internal/channel/channel.go
package channel

import (
    "context"

    "github.com/MEKXH/golem/internal/bus"
)

// Channel interface for chat platforms
type Channel interface {
    Name() string
    Start(ctx context.Context) error
    Stop(ctx context.Context) error
    Send(ctx context.Context, msg *bus.OutboundMessage) error
    IsAllowed(senderID string) bool
}

// BaseChannel provides common functionality
type BaseChannel struct {
    Bus       *bus.MessageBus
    AllowList map[string]bool
}

// IsAllowed checks if sender is permitted
func (b *BaseChannel) IsAllowed(senderID string) bool {
    if len(b.AllowList) == 0 {
        return true
    }
    return b.AllowList[senderID]
}

// PublishInbound sends message to bus
func (b *BaseChannel) PublishInbound(msg *bus.InboundMessage) {
    b.Bus.PublishInbound(msg)
}
```

**Step 2: Commit**

```bash
git add internal/channel/
git commit -m "feat(channel): add channel interface"
```

---

### Task 19: Telegram Implementation

**Files:**
- Create: `internal/channel/telegram/telegram.go`

**Step 1: Write implementation**

```go
// internal/channel/telegram/telegram.go
package telegram

import (
    "context"
    "fmt"
    "log/slog"
    "regexp"
    "strings"
    "time"

    tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
    "github.com/MEKXH/golem/internal/bus"
    "github.com/MEKXH/golem/internal/channel"
    "github.com/MEKXH/golem/internal/config"
)

// Channel implements Telegram bot
type Channel struct {
    channel.BaseChannel
    cfg *config.TelegramConfig
    bot *tgbotapi.BotAPI
}

// New creates a Telegram channel
func New(cfg *config.TelegramConfig, msgBus *bus.MessageBus) *Channel {
    allowList := make(map[string]bool)
    for _, id := range cfg.AllowFrom {
        allowList[id] = true
    }
    return &Channel{
        BaseChannel: channel.BaseChannel{
            Bus:       msgBus,
            AllowList: allowList,
        },
        cfg: cfg,
    }
}

func (c *Channel) Name() string { return "telegram" }

func (c *Channel) Start(ctx context.Context) error {
    bot, err := tgbotapi.NewBotAPI(c.cfg.Token)
    if err != nil {
        return fmt.Errorf("telegram init failed: %w", err)
    }
    c.bot = bot

    slog.Info("telegram bot connected", "username", bot.Self.UserName)

    u := tgbotapi.NewUpdate(0)
    u.Timeout = 60
    updates := bot.GetUpdatesChan(u)

    for {
        select {
        case <-ctx.Done():
            return nil
        case update := <-updates:
            if update.Message == nil {
                continue
            }
            go c.handleMessage(update.Message)
        }
    }
}

func (c *Channel) handleMessage(msg *tgbotapi.Message) {
    senderID := fmt.Sprintf("%d", msg.From.ID)

    if !c.IsAllowed(senderID) {
        slog.Debug("unauthorized sender", "id", senderID)
        return
    }

    content := msg.Text
    if content == "" {
        content = msg.Caption
    }
    if content == "" {
        return
    }

    c.PublishInbound(&bus.InboundMessage{
        Channel:   "telegram",
        SenderID:  senderID,
        ChatID:    fmt.Sprintf("%d", msg.Chat.ID),
        Content:   content,
        Timestamp: time.Now(),
        Metadata: map[string]any{
            "message_id": msg.MessageID,
            "username":   msg.From.UserName,
        },
    })
}

func (c *Channel) Send(ctx context.Context, msg *bus.OutboundMessage) error {
    if c.bot == nil {
        return fmt.Errorf("bot not initialized")
    }

    chatID := parseInt64(msg.ChatID)
    html := markdownToHTML(msg.Content)

    tgMsg := tgbotapi.NewMessage(chatID, html)
    tgMsg.ParseMode = "HTML"

    _, err := c.bot.Send(tgMsg)
    if err != nil {
        // Fallback to plain text
        tgMsg.ParseMode = ""
        tgMsg.Text = msg.Content
        _, err = c.bot.Send(tgMsg)
    }
    return err
}

func (c *Channel) Stop(ctx context.Context) error {
    if c.bot != nil {
        c.bot.StopReceivingUpdates()
    }
    return nil
}

func parseInt64(s string) int64 {
    var n int64
    fmt.Sscanf(s, "%d", &n)
    return n
}

func markdownToHTML(text string) string {
    // Basic markdown to Telegram HTML conversion
    text = regexp.MustCompile(`\*\*(.+?)\*\*`).ReplaceAllString(text, "<b>$1</b>")
    text = regexp.MustCompile(`__(.+?)__`).ReplaceAllString(text, "<b>$1</b>")
    text = regexp.MustCompile("`([^`]+)`").ReplaceAllString(text, "<code>$1</code>")
    text = strings.ReplaceAll(text, "&", "&amp;")
    text = strings.ReplaceAll(text, "<", "&lt;")
    text = strings.ReplaceAll(text, ">", "&gt;")
    return text
}
```

**Step 2: Commit**

```bash
git add internal/channel/telegram/
git commit -m "feat(channel): add telegram implementation"
```

---

### Task 20: Channel Manager & Run Command

**Files:**
- Create: `internal/channel/manager.go`
- Modify: `cmd/golem/commands/run.go`

**Step 1: Create manager**

```go
// internal/channel/manager.go
package channel

import (
    "context"
    "log/slog"
    "sync"

    "github.com/MEKXH/golem/internal/bus"
)

// Manager coordinates all channels
type Manager struct {
    channels map[string]Channel
    bus      *bus.MessageBus
    mu       sync.RWMutex
}

// NewManager creates a channel manager
func NewManager(msgBus *bus.MessageBus) *Manager {
    return &Manager{
        channels: make(map[string]Channel),
        bus:      msgBus,
    }
}

// Register adds a channel
func (m *Manager) Register(ch Channel) {
    m.mu.Lock()
    defer m.mu.Unlock()
    m.channels[ch.Name()] = ch
}

// StartAll starts all channels
func (m *Manager) StartAll(ctx context.Context) {
    m.mu.RLock()
    defer m.mu.RUnlock()

    for name, ch := range m.channels {
        go func(n string, c Channel) {
            slog.Info("starting channel", "name", n)
            if err := c.Start(ctx); err != nil {
                slog.Error("channel error", "name", n, "error", err)
            }
        }(name, ch)
    }
}

// RouteOutbound sends outbound messages to appropriate channels
func (m *Manager) RouteOutbound(ctx context.Context) {
    for {
        select {
        case <-ctx.Done():
            return
        case msg := <-m.bus.Outbound():
            m.mu.RLock()
            if ch, ok := m.channels[msg.Channel]; ok {
                go ch.Send(ctx, msg)
            }
            m.mu.RUnlock()
        }
    }
}

// StopAll stops all channels
func (m *Manager) StopAll(ctx context.Context) {
    m.mu.RLock()
    defer m.mu.RUnlock()

    for _, ch := range m.channels {
        ch.Stop(ctx)
    }
}
```

**Step 2: Update run command**

```go
// cmd/golem/commands/run.go
package commands

import (
    "context"
    "fmt"
    "log/slog"
    "os"
    "os/signal"
    "syscall"

    "github.com/spf13/cobra"
    "github.com/MEKXH/golem/internal/agent"
    "github.com/MEKXH/golem/internal/bus"
    "github.com/MEKXH/golem/internal/channel"
    "github.com/MEKXH/golem/internal/channel/telegram"
    "github.com/MEKXH/golem/internal/config"
    "github.com/MEKXH/golem/internal/provider"
)

func NewRunCmd() *cobra.Command {
    var port int

    cmd := &cobra.Command{
        Use:   "run",
        Short: "Start Golem server",
        RunE:  runServer,
    }

    cmd.Flags().IntVarP(&port, "port", "p", 18790, "Server port")
    return cmd
}

func runServer(cmd *cobra.Command, args []string) error {
    ctx, cancel := signal.NotifyContext(context.Background(),
        syscall.SIGINT, syscall.SIGTERM)
    defer cancel()

    // Load config
    cfg, err := config.Load()
    if err != nil {
        return fmt.Errorf("failed to load config: %w", err)
    }

    // Create components
    msgBus := bus.NewMessageBus(100)

    model, err := provider.NewChatModel(ctx, cfg)
    if err != nil {
        slog.Warn("no model configured", "error", err)
    }

    // Create and start agent loop
    loop := agent.NewLoop(cfg, msgBus, model)
    if err := loop.RegisterDefaultTools(cfg); err != nil {
        return err
    }
    go loop.Run(ctx)

    // Create channel manager
    chanMgr := channel.NewManager(msgBus)

    // Register Telegram if enabled
    if cfg.Channels.Telegram.Enabled {
        tg := telegram.New(&cfg.Channels.Telegram, msgBus)
        chanMgr.Register(tg)
    }

    // Start channels
    chanMgr.StartAll(ctx)
    go chanMgr.RouteOutbound(ctx)

    fmt.Printf("Golem server running. Press Ctrl+C to stop.\n")

    <-ctx.Done()

    slog.Info("shutting down")
    chanMgr.StopAll(context.Background())

    return nil
}
```

**Step 3: Commit**

```bash
git add internal/channel/manager.go cmd/golem/commands/run.go
git commit -m "feat(cli): implement run command with channel manager"
```

---

## Phase 10: Final Integration

### Task 21: Fix Import Paths

**Files:**
- All files with imports

**Step 1: Update all imports to use correct module path**

Replace `github.com/MEKXH/golem` with `github.com/MEKXH/golem` (or your actual module name) in all files.

**Step 2: Run tests**

Run: `go test ./... -v`
Expected: All tests pass

**Step 3: Build and test**

Run:
```bash
go build ./cmd/golem
./golem init
./golem status
./golem chat "hello"
```

**Step 4: Commit**

```bash
git add -A
git commit -m "fix: update import paths"
```

---

### Task 22: Add README

**Files:**
- Create: `README.md`

**Step 1: Write README**

```markdown
# Golem

Lightweight personal AI assistant built with Go and Eino.

## Installation

```bash
go install github.com/MEKXH/golem/cmd/golem@latest
```

## Quick Start

```bash
# Initialize configuration
golem init

# Edit config to add API key
# ~/.golem/config.json

# Chat interactively
golem chat

# Send single message
golem chat "What is 2+2?"

# Start server with Telegram
golem run
```

## Configuration

Config file: `~/.golem/config.json`

### Providers

Supports: OpenRouter, Claude, OpenAI, DeepSeek, Gemini, Ollama, and more.

### Channels

- Telegram (implemented)
- More coming soon

## License

MIT
```

**Step 2: Commit**

```bash
git add README.md
git commit -m "docs: add README"
```

---

## Summary

**Total Tasks: 22**

**Phases:**
1. Project Skeleton (Tasks 1-2)
2. Configuration System (Tasks 3-4)
3. Message Bus (Tasks 5-6)
4. Session Management (Task 7)
5. Tool System (Tasks 8-10)
6. Provider System (Task 11)
7. Agent Loop (Tasks 12-13)
8. CLI Commands (Tasks 14-17)
9. Telegram Channel (Tasks 18-20)
10. Final Integration (Tasks 21-22)

**Key Dependencies:**
- github.com/cloudwego/eino
- github.com/cloudwego/eino-ext
- github.com/spf13/cobra
- github.com/spf13/viper
- github.com/go-telegram-bot-api/telegram-bot-api/v5
