```
# 设计文档

## 1. 系统架构概览

### 1.1 架构图

```
┌─────────────────────────────────────────────────────────────────────────┐
│                           外部服务                                       │
├─────────────┬─────────────┬─────────────┬─────────────┬─────────────────┤
│  Telegram   │  Discord    │   OpenAI    │  Anthropic  │    Ollama       │
│  Bot API    │  Bot API    │    API      │    API      │    (本地)       │
└──────┬──────┴──────┬──────┴──────┬──────┴──────┬──────┴────────┬────────┘
       │             │             │             │               │
       ▼             ▼             ▼             ▼               ▼
┌─────────────────────────────────────────────────────────────────────────┐
│                         Gateway 网关层                                   │
├─────────────────────────────────────────────────────────────────────────┤
│  ┌───────────────┐  ┌───────────────┐  ┌───────────────────────────┐   │
│  │ Channel       │  │ Auth          │  │ Health Check              │   │
│  │ Manager       │  │ Middleware    │  │ Server (HTTP :8080)       │   │
│  └───────┬───────┘  └───────┬───────┘  └───────────────────────────┘   │
│          │                  │                                          │
│          ▼                  ▼                                          │
│  ┌─────────────────────────────────────────────────────────────────┐   │
│  │                    Message Router                                │   │
│  │         (按发送者/群组/渠道路由消息)                              │   │
│  └─────────────────────────────┬───────────────────────────────────┘   │
└────────────────────────────────┼────────────────────────────────────────┘
                                 │
                                 ▼
┌─────────────────────────────────────────────────────────────────────────┐
│                         Agent 智能体层                                   │
├─────────────────────────────────────────────────────────────────────────┤
│  ┌─────────────────────────────────────────────────────────────────┐   │
│  │                    Agent Router                                  │   │
│  │              (多智能体路由和隔离)                                 │   │
│  └─────────────────────────────┬───────────────────────────────────┘   │
│                                │                                        │
│          ┌─────────────────────┼─────────────────────┐                 │
│          ▼                     ▼                     ▼                 │
│  ┌───────────────┐     ┌───────────────┐     ┌───────────────┐        │
│  │   Agent A     │     │   Agent B     │     │   Agent C     │        │
│  │  (默认助手)    │     │  (专用智能体)  │     │  (群组智能体)  │        │
│  └───────┬───────┘     └───────┬───────┘     └───────┬───────┘        │
│          │                     │                     │                 │
└──────────┼─────────────────────┼─────────────────────┼─────────────────┘
           │                     │                     │
           ▼                     ▼                     ▼
┌─────────────────────────────────────────────────────────────────────────┐
│                         Core 核心层                                      │
├─────────────────────────────────────────────────────────────────────────┤
│  ┌───────────────┐  ┌───────────────┐  ┌───────────────────────────┐   │
│  │ Session       │  │ LLM           │  │ Tool                      │   │
│  │ Manager       │  │ Client        │  │ Executor                  │   │
│  └───────────────┘  └───────────────┘  └───────────────────────────┘   │
│                                                                          │
│  ┌───────────────┐  ┌───────────────┐  ┌───────────────────────────┐   │
│  │ Memory        │  │ Config        │  │ Logger                    │   │
│  │ Store         │  │ Manager       │  │ (slog/zerolog)            │   │
│  └───────────────┘  └───────────────┘  └───────────────────────────┘   │
└─────────────────────────────────────────────────────────────────────────┘
                                 │
                                 ▼
┌─────────────────────────────────────────────────────────────────────────┐
│                         Storage 存储层                                   │
├─────────────────────────────────────────────────────────────────────────┤
│  ┌───────────────────────────────────────────────────────────────────┐ │
│  │                    File System (文件系统)                          │ │
│  │  ~/.mujibot/                                                       │ │
│  │  ├── config.json5      # 配置文件                                  │ │
│  │  ├── sessions/         # 会话持久化 (可选)                         │ │
│  │  ├── memory/           # 记忆存储                                  │ │
│  │  │   ├── YYYY-MM-DD.md # 每日笔记                                  │ │
│  │  │   └── MEMORY.md     # 长期记忆                                  │ │
│  │  └── logs/             # 日志文件                                  │ │
│  └───────────────────────────────────────────────────────────────────┘ │
└─────────────────────────────────────────────────────────────────────────┘
```

### 1.2 核心设计原则

1. **单进程架构**: 所有组件运行在单个Go进程中，通过channel和goroutine实现并发
2. **事件驱动**: 消息处理采用事件驱动模型，减少轮询开销
3. **内存优先**: 优先使用内存缓存，按需持久化到文件系统
4. **优雅降级**: 组件故障不影响其他组件运行
5. **零依赖启动**: 单二进制文件，无需外部运行时

## 2. 核心组件设计

### 2.1 Gateway 网关

#### 2.1.1 Channel Manager

负责管理所有消息渠道的连接生命周期。

```go
package channel

type Manager struct {
    channels  map[string]Channel
    msgChan   chan *Message
    errChan   chan error
    mu        sync.RWMutex
}

type Channel interface {
    Name() string
    Connect(ctx context.Context) error
    Disconnect() error
    Send(ctx context.Context, recipient string, message string) error
    Messages() <-chan *Message
    IsConnected() bool
}

type Message struct {
    ID        string
    Channel   string
    Sender    Sender
    Content   string
    Timestamp time.Time
    Metadata  map[string]any
}

type Sender struct {
    ID       string
    Name     string
    IsGroup  bool
    GroupID  string
}
```

#### 2.1.2 Telegram Channel

```go
package channel

type TelegramChannel struct {
    token       string
    api         *tgbotapi.BotAPI
    updates     tgbotapi.UpdatesChannel
    msgChan     chan *Message
    allowedUsers map[int64]bool
}

func NewTelegramChannel(token string, allowedUsers []int64) *TelegramChannel

func (t *TelegramChannel) Connect(ctx context.Context) error {
    bot, err := tgbotapi.NewBotAPI(t.token)
    if err != nil {
        return err
    }
    t.api = bot
    
    u := tgbotapi.NewUpdate(0)
    u.Timeout = 60
    t.updates = bot.GetUpdatesChan(u)
    
    go t.processUpdates(ctx)
    return nil
}

func (t *TelegramChannel) processUpdates(ctx context.Context) {
    for {
        select {
        case <-ctx.Done():
            return
        case update := <-t.updates:
            if update.Message == nil {
                continue
            }
            msg := t.convertMessage(update.Message)
            if t.isAllowed(msg.Sender.ID) {
                t.msgChan <- msg
            }
        }
    }
}
```

#### 2.1.3 Discord Channel

```go
package channel

type DiscordChannel struct {
    token    string
    session  *discordgo.Session
    msgChan  chan *Message
    allowedGuilds map[string]bool
}

func NewDiscordChannel(token string, allowedGuilds []string) *DiscordChannel

func (d *DiscordChannel) Connect(ctx context.Context) error {
    s, err := discordgo.New("Bot " + d.token)
    if err != nil {
        return err
    }
    d.session = s
    
    s.AddHandler(d.onMessageCreate)
    return s.Open()
}

func (d *DiscordChannel) onMessageCreate(s *discordgo.Session, m *discordgo.MessageCreate) {
    if m.Author.Bot {
        return
    }
    msg := d.convertMessage(m)
    if d.isAllowed(msg) {
        d.msgChan <- msg
    }
}
```

### 2.2 Agent Router 智能体路由器

```go
package agent

type Router struct {
    agents      map[string]*Agent
    defaultID   string
    sessionMgr  *session.Manager
    llmClient   *llm.Client
    toolExec    *tool.Executor
    mu          sync.RWMutex
}

type Agent struct {
    ID           string
    Name         string
    SystemPrompt string
    Tools        []string
    MaxTokens    int
}

func NewRouter(cfg *config.Config, sessionMgr *session.Manager, llmClient *llm.Client, toolExec *tool.Executor) *Router

func (r *Router) Route(msg *channel.Message) *Agent {
    routingKey := r.getRoutingKey(msg)
    
    if agentID, ok := r.routeMap[routingKey]; ok {
        return r.agents[agentID]
    }
    return r.agents[r.defaultID]
}

func (r *Router) Process(ctx context.Context, msg *channel.Message) (*Response, error) {
    agent := r.Route(msg)
    
    sess := r.sessionMgr.GetOrCreate(msg.Sender.ID, msg.Channel)
    
    sess.AddMessage("user", msg.Content)
    
    resp, err := r.llmClient.Chat(ctx, &llm.Request{
        Model:       agent.Model,
        System:      agent.SystemPrompt,
        Messages:    sess.GetMessages(),
        Tools:       r.getToolDefinitions(agent.Tools),
        MaxTokens:   agent.MaxTokens,
        Stream:      true,
    })
    if err != nil {
        return nil, err
    }
    
    return r.handleResponse(ctx, resp, sess, agent)
}

func (r *Router) handleResponse(ctx context.Context, resp *llm.Response, sess *session.Session, agent *Agent) (*Response, error) {
    if len(resp.ToolCalls) > 0 {
        for _, tc := range resp.ToolCalls {
            result, err := r.toolExec.Execute(ctx, tc.Name, tc.Arguments)
            if err != nil {
                result = fmt.Sprintf("Error: %v", err)
            }
            sess.AddMessage("tool", result)
        }
        return r.Process(ctx, sess.LastUserMessage())
    }
    
    sess.AddMessage("assistant", resp.Content)
    return &Response{Content: resp.Content}, nil
}
```

### 2.3 Session Manager 会话管理器

```go
package session

type Manager struct {
    sessions    map[string]*Session
    maxMessages int
    idleTimeout time.Duration
    mu          sync.RWMutex
    cleanupTicker *time.Ticker
}

type Session struct {
    ID          string
    SenderID    string
    Channel     string
    Messages    []Message
    LastActive  time.Time
    Metadata    map[string]any
}

type Message struct {
    Role    string
    Content string
    Time    time.Time
}

func NewManager(maxMessages int, idleTimeout time.Duration) *Manager {
    m := &Manager{
        sessions:    make(map[string]*Session),
        maxMessages: maxMessages,
        idleTimeout: idleTimeout,
    }
    go m.cleanup()
    return m
}

func (m *Manager) GetOrCreate(senderID, channel string) *Session {
    key := fmt.Sprintf("%s:%s", channel, senderID)
    
    m.mu.RLock()
    sess, ok := m.sessions[key]
    m.mu.RUnlock()
    
    if ok {
        sess.Touch()
        return sess
    }
    
    m.mu.Lock()
    defer m.mu.Unlock()
    
    sess = &Session{
        ID:         key,
        SenderID:   senderID,
        Channel:    channel,
        Messages:   make([]Message, 0, m.maxMessages),
        LastActive: time.Now(),
    }
    m.sessions[key] = sess
    return sess
}

func (s *Session) AddMessage(role, content string) {
    if len(s.Messages) >= m.maxMessages {
        s.Messages = s.Messages[1:]
    }
    s.Messages = append(s.Messages, Message{
        Role:    role,
        Content: content,
        Time:    time.Now(),
    })
}

func (m *Manager) cleanup() {
    ticker := time.NewTicker(5 * time.Minute)
    for range ticker.C {
        m.mu.Lock()
        now := time.Now()
        for key, sess := range m.sessions {
            if now.Sub(sess.LastActive) > m.idleTimeout {
                delete(m.sessions, key)
            }
        }
        m.mu.Unlock()
    }
}
```

### 2.4 LLM Client 大语言模型客户端

```go
package llm

type Client struct {
    providers map[string]Provider
    defaultProvider string
    timeout   time.Duration
    maxRetries int
    httpClient *http.Client
}

type Provider interface {
    Name() string
    Chat(ctx context.Context, req *Request) (*Response, error)
    ChatStream(ctx context.Context, req *Request) (<-chan StreamChunk, error)
}

type Request struct {
    Model       string
    System      string
    Messages    []Message
    Tools       []Tool
    MaxTokens   int
    Temperature float64
    Stream      bool
}

type Response struct {
    Content    string
    ToolCalls  []ToolCall
    Usage      Usage
    FinishReason string
}

type ToolCall struct {
    ID        string
    Name      string
    Arguments string
}

func NewClient(cfg *config.LLMConfig) *Client {
    c := &Client{
        providers: make(map[string]Provider),
        timeout:   time.Duration(cfg.Timeout) * time.Second,
        maxRetries: cfg.MaxRetries,
        httpClient: &http.Client{
            Timeout: time.Duration(cfg.Timeout) * time.Second,
        },
    }
    
    if cfg.OpenAI.APIKey != "" {
        c.providers["openai"] = NewOpenAIProvider(cfg.OpenAI)
    }
    if cfg.Anthropic.APIKey != "" {
        c.providers["anthropic"] = NewAnthropicProvider(cfg.Anthropic)
    }
    if cfg.Ollama.BaseURL != "" {
        c.providers["ollama"] = NewOllamaProvider(cfg.Ollama)
    }
    
    c.defaultProvider = cfg.DefaultProvider
    return c
}

func (c *Client) Chat(ctx context.Context, req *Request) (*Response, error) {
    provider := c.providers[req.Provider]
    if provider == nil {
        provider = c.providers[c.defaultProvider]
    }
    
    var resp *Response
    var err error
    
    for i := 0; i < c.maxRetries; i++ {
        resp, err = provider.Chat(ctx, req)
        if err == nil {
            return resp, nil
        }
        
        if !c.isRetryable(err) {
            return nil, err
        }
        
        delay := c.backoff(i)
        select {
        case <-ctx.Done():
            return nil, ctx.Err()
        case <-time.After(delay):
        }
    }
    
    return nil, err
}

func (c *Client) backoff(attempt int) time.Duration {
    return time.Duration(100*math.Pow(2, float64(attempt))) * time.Millisecond
}
```

#### 2.4.1 OpenAI Provider

```go
package llm

type OpenAIProvider struct {
    apiKey  string
    baseURL string
    client  *http.Client
}

func NewOpenAIProvider(cfg config.OpenAIConfig) *OpenAIProvider {
    baseURL := cfg.BaseURL
    if baseURL == "" {
        baseURL = "https://api.openai.com/v1"
    }
    return &OpenAIProvider{
        apiKey:  cfg.APIKey,
        baseURL: baseURL,
        client:  &http.Client{Timeout: 60 * time.Second},
    }
}

func (p *OpenAIProvider) Chat(ctx context.Context, req *Request) (*Response, error) {
    body := p.buildRequestBody(req)
    
    httpReq, err := http.NewRequestWithContext(ctx, "POST", 
        p.baseURL+"/chat/completions", 
        bytes.NewReader(body))
    if err != nil {
        return nil, err
    }
    
    httpReq.Header.Set("Authorization", "Bearer "+p.apiKey)
    httpReq.Header.Set("Content-Type", "application/json")
    
    resp, err := p.client.Do(httpReq)
    if err != nil {
        return nil, err
    }
    defer resp.Body.Close()
    
    return p.parseResponse(resp.Body)
}

func (p *OpenAIProvider) ChatStream(ctx context.Context, req *Request) (<-chan StreamChunk, error) {
    req.Stream = true
    body := p.buildRequestBody(req)
    
    httpReq, err := http.NewRequestWithContext(ctx, "POST",
        p.baseURL+"/chat/completions",
        bytes.NewReader(body))
    if err != nil {
        return nil, err
    }
    
    httpReq.Header.Set("Authorization", "Bearer "+p.apiKey)
    httpReq.Header.Set("Content-Type", "application/json")
    
    resp, err := p.client.Do(httpReq)
    if err != nil {
        return nil, err
    }
    
    ch := make(chan StreamChunk, 10)
    go p.processStream(resp.Body, ch)
    return ch, nil
}
```

### 2.5 Tool Executor 工具执行器

```go
package tool

type Executor struct {
    tools      map[string]Tool
    workDir    string
    timeout    time.Duration
    confirmFn  func(string) bool
}

type Tool interface {
    Name() string
    Description() string
    Parameters() map[string]Parameter
    Execute(ctx context.Context, args map[string]any) (string, error)
}

type Parameter struct {
    Type        string
    Description string
    Required    bool
    Enum        []string
}

func NewExecutor(cfg *config.ToolsConfig) *Executor {
    e := &Executor{
        tools:   make(map[string]Tool),
        workDir: cfg.WorkDir,
        timeout: time.Duration(cfg.Timeout) * time.Second,
    }
    
    e.registerBuiltinTools()
    return e
}

func (e *Executor) registerBuiltinTools() {
    e.Register(NewReadFileTool(e.workDir))
    e.Register(NewWriteFileTool(e.workDir))
    e.Register(NewEditFileTool(e.workDir))
    e.Register(NewListDirTool(e.workDir))
    e.Register(NewExecTool(e.workDir, e.timeout, e.confirmFn))
    e.Register(NewGrepTool(e.workDir))
    e.Register(NewFindTool(e.workDir))
}

func (e *Executor) Register(t Tool) {
    e.tools[t.Name()] = t
}

func (e *Executor) Execute(ctx context.Context, name string, argsJSON string) (string, error) {
    tool, ok := e.tools[name]
    if !ok {
        return "", fmt.Errorf("unknown tool: %s", name)
    }
    
    var args map[string]any
    if err := json.Unmarshal([]byte(argsJSON), &args); err != nil {
        return "", fmt.Errorf("invalid arguments: %w", err)
    }
    
    if e.needsConfirmation(name, args) {
        if !e.confirmFn(name) {
            return "Operation cancelled by user", nil
        }
    }
    
    ctx, cancel := context.WithTimeout(ctx, e.timeout)
    defer cancel()
    
    return tool.Execute(ctx, args)
}

func (e *Executor) GetToolDefinitions() []llm.Tool {
    var defs []llm.Tool
    for _, t := range e.tools {
        defs = append(defs, llm.Tool{
            Type: "function",
            Function: llm.FunctionDef{
                Name:        t.Name(),
                Description: t.Description(),
                Parameters:  t.Parameters(),
            },
        })
    }
    return defs
}
```

#### 2.5.1 内置工具实现

```go
package tool

type ReadFileTool struct {
    workDir string
}

func (t *ReadFileTool) Name() string { return "read" }

func (t *ReadFileTool) Description() string {
    return "Read the contents of a file"
}

func (t *ReadFileTool) Parameters() map[string]Parameter {
    return map[string]Parameter{
        "path": {
            Type:        "string",
            Description: "The path to the file to read",
            Required:    true,
        },
    }
}

func (t *ReadFileTool) Execute(ctx context.Context, args map[string]any) (string, error) {
    path := args["path"].(string)
    
    fullPath := filepath.Join(t.workDir, path)
    if !strings.HasPrefix(fullPath, t.workDir) {
        return "", fmt.Errorf("access denied: path outside work directory")
    }
    
    info, err := os.Stat(fullPath)
    if err != nil {
        return "", err
    }
    if info.Size() > 1*1024*1024 {
        return "", fmt.Errorf("file too large (max 1MB)")
    }
    
    content, err := os.ReadFile(fullPath)
    if err != nil {
        return "", err
    }
    
    return string(content), nil
}

type ExecTool struct {
    workDir    string
    timeout    time.Duration
    confirmFn  func(string) bool
    dangerous  map[string]bool
}

func NewExecTool(workDir string, timeout time.Duration, confirmFn func(string) bool) *ExecTool {
    return &ExecTool{
        workDir:   workDir,
        timeout:   timeout,
        confirmFn: confirmFn,
        dangerous: map[string]bool{
            "rm -rf":  true,
            "dd":      true,
            "mkfs":    true,
            "chmod 777": true,
            "reboot":  true,
            "shutdown": true,
            "init":    true,
        },
    }
}

func (t *ExecTool) Name() string { return "exec" }

func (t *ExecTool) Execute(ctx context.Context, args map[string]any) (string, error) {
    cmd := args["command"].(string)
    
    if t.isDangerous(cmd) {
        if !t.confirmFn(cmd) {
            return "Command cancelled by user", nil
        }
    }
    
    ctx, cancel := context.WithTimeout(ctx, t.timeout)
    defer cancel()
    
    execCmd := exec.CommandContext(ctx, "sh", "-c", cmd)
    execCmd.Dir = t.workDir
    
    output, err := execCmd.CombinedOutput()
    if err != nil {
        return string(output) + "\nError: " + err.Error(), nil
    }
    
    return string(output), nil
}

func (t *ExecTool) isDangerous(cmd string) bool {
    for dangerous := range t.dangerous {
        if strings.Contains(cmd, dangerous) {
            return true
        }
    }
    return false
}
```

### 2.6 Config Manager 配置管理器

```go
package config

import (
    "os"
    "path/filepath"
    "github.com/fsnotify/fsnotify"
)

type Manager struct {
    path     string
    config   *Config
    watcher  *fsnotify.Watcher
    onChange func(*Config)
    mu       sync.RWMutex
}

type Config struct {
    Server   ServerConfig   `json:"server"`
    Channels ChannelsConfig `json:"channels"`
    LLM      LLMConfig      `json:"llm"`
    Agents   AgentsConfig   `json:"agents"`
    Tools    ToolsConfig    `json:"tools"`
    Session  SessionConfig  `json:"session"`
    Logging  LoggingConfig  `json:"logging"`
}

func NewManager(path string) (*Manager, error) {
    m := &Manager{path: path}
    
    if _, err := os.Stat(path); os.IsNotExist(err) {
        if err := m.createDefault(); err != nil {
            return nil, err
        }
    }
    
    if err := m.load(); err != nil {
        return nil, err
    }
    
    if err := m.watch(); err != nil {
        return nil, err
    }
    
    return m, nil
}

func (m *Manager) load() error {
    data, err := os.ReadFile(m.path)
    if err != nil {
        return err
    }
    
    data = m.expandEnvVars(data)
    
    var cfg Config
    if err := json5.Unmarshal(data, &cfg); err != nil {
        return fmt.Errorf("parse config: %w", err)
    }
    
    if err := m.validate(&cfg); err != nil {
        return err
    }
    
    m.mu.Lock()
    m.config = &cfg
    m.mu.Unlock()
    
    return nil
}

func (m *Manager) expandEnvVars(data []byte) []byte {
    re := regexp.MustCompile(`\$\{([^}]+)\}`)
    return re.ReplaceAllFunc(data, func(match []byte) []byte {
        envVar := string(re.FindSubmatch(match)[1])
        return []byte(os.Getenv(envVar))
    })
}

func (m *Manager) watch() error {
    watcher, err := fsnotify.NewWatcher()
    if err != nil {
        return err
    }
    m.watcher = watcher
    
    dir := filepath.Dir(m.path)
    if err := watcher.Add(dir); err != nil {
        return err
    }
    
    go func() {
        for {
            select {
            case event, ok := <-watcher.Events:
                if !ok {
                    return
                }
                if event.Name == m.path && event.Op&fsnotify.Write == fsnotify.Write {
                    if err := m.load(); err == nil && m.onChange != nil {
                        m.onChange(m.Get())
                    }
                }
            case <-watcher.Errors:
                return
            }
        }
    }()
    
    return nil
}

func (m *Manager) OnChange(fn func(*Config)) {
    m.onChange = fn
}

func (m *Manager) Get() *Config {
    m.mu.RLock()
    defer m.mu.RUnlock()
    return m.config
}
```

### 2.7 Memory Store 记忆存储

```go
package memory

type Store struct {
    baseDir      string
    maxFileSize  int64
    maxDays      int
}

func NewStore(baseDir string, maxFileSize int64) *Store {
    return &Store{
        baseDir:     baseDir,
        maxFileSize: maxFileSize,
        maxDays:     2,
    }
}

func (s *Store) LoadDailyNotes() (string, error) {
    var notes []string
    
    for i := 0; i < s.maxDays; i++ {
        date := time.Now().AddDate(0, 0, -i).Format("2006-01-02")
        path := filepath.Join(s.baseDir, "memory", date+".md")
        
        content, err := os.ReadFile(path)
        if err != nil {
            continue
        }
        notes = append(notes, fmt.Sprintf("## %s\n%s", date, string(content)))
    }
    
    return strings.Join(notes, "\n\n"), nil
}

func (s *Store) LoadLongTermMemory() (string, error) {
    path := filepath.Join(s.baseDir, "MEMORY.md")
    content, err := os.ReadFile(path)
    if err != nil {
        return "", nil
    }
    return string(content), nil
}

func (s *Store) SaveDailyNote(content string) error {
    dir := filepath.Join(s.baseDir, "memory")
    if err := os.MkdirAll(dir, 0755); err != nil {
        return err
    }
    
    path := filepath.Join(dir, time.Now().Format("2006-01-02")+".md")
    return os.WriteFile(path, []byte(content), 0644)
}

func (s *Store) Search(keyword string) ([]string, error) {
    var results []string
    
    files, err := filepath.Glob(filepath.Join(s.baseDir, "memory", "*.md"))
    if err != nil {
        return nil, err
    }
    
    longTermPath := filepath.Join(s.baseDir, "MEMORY.md")
    if _, err := os.Stat(longTermPath); err == nil {
        files = append(files, longTermPath)
    }
    
    for _, file := range files {
        content, err := os.ReadFile(file)
        if err != nil {
            continue
        }
        
        if strings.Contains(string(content), keyword) {
            results = append(results, file)
        }
    }
    
    return results, nil
}
```

## 3. 数据流

### 3.1 消息处理流程

```
用户消息
    │
    ▼
┌─────────────────┐
│ Channel (TG/Discord) │
└────────┬────────┘
         │ Message
         ▼
┌─────────────────┐
│ Auth Middleware │ ─── 检查用户白名单
└────────┬────────┘
         │
         ▼
┌─────────────────┐
│ Message Router  │ ─── 按渠道/群组路由
└────────┬────────┘
         │
         ▼
┌─────────────────┐
│ Agent Router    │ ─── 选择智能体
└────────┬────────┘
         │
         ▼
┌─────────────────┐
│ Session Manager │ ─── 获取/创建会话
└────────┬────────┘
         │
         ▼
┌─────────────────┐
│ LLM Client      │ ─── 调用LLM API
└────────┬────────┘
         │
         ├─── 有工具调用? ──→ ┌─────────────┐
         │                    │ Tool Executor│
         │                    └──────┬──────┘
         │                           │
         │                           ▼
         │                    ┌─────────────┐
         │                    │ LLM Client  │ ←── 再次调用LLM
         │                    └─────────────┘
         │
         ▼
┌─────────────────┐
│ Session Manager │ ─── 保存响应到会话
└────────┬────────┘
         │
         ▼
┌─────────────────┐
│ Channel         │ ─── 发送响应给用户
└─────────────────┘
```

### 3.2 流式响应处理

```go
func (r *Router) processStream(ctx context.Context, msg *channel.Message) error {
    agent := r.Route(msg)
    sess := r.sessionMgr.GetOrCreate(msg.Sender.ID, msg.Channel)
    sess.AddMessage("user", msg.Content)
    
    stream, err := r.llmClient.ChatStream(ctx, &llm.Request{
        Model:    agent.Model,
        System:   agent.SystemPrompt,
        Messages: sess.GetMessages(),
        Tools:    r.getToolDefinitions(agent.Tools),
    })
    if err != nil {
        return err
    }
    
    var fullContent strings.Builder
    var toolCalls []llm.ToolCall
    
    for chunk := range stream {
        if chunk.Error != nil {
            return chunk.Error
        }
        
        if chunk.Content != "" {
            fullContent.WriteString(chunk.Content)
        }
        
        if len(chunk.ToolCalls) > 0 {
            toolCalls = append(toolCalls, chunk.ToolCalls...)
        }
    }
    
    if len(toolCalls) > 0 {
        return r.handleToolCalls(ctx, toolCalls, sess, agent)
    }
    
    sess.AddMessage("assistant", fullContent.String())
    return r.sendResponse(msg, fullContent.String())
}
```

## 4. 接口定义

### 4.1 HTTP API

```yaml
openapi: 3.0.0
info:
  title: Mujibot API
  version: 1.0.0

paths:
  /health:
    get:
      summary: 健康检查
      responses:
        200:
          description: 系统健康状态
          content:
            application/json:
              schema:
                type: object
                properties:
                  status:
                    type: string
                  uptime:
                    type: integer
                  memory:
                    type: object
                    properties:
                      alloc: integer
                      total: integer
                      sys: integer
                  channels:
                    type: object
                    additionalProperties:
                      type: boolean
                  messages:
                    type: object
                    properties:
                      total: integer
                      hourly: integer

  /metrics:
    get:
      summary: 性能指标
      responses:
        200:
          description: 系统性能指标
          content:
            application/json:
              schema:
                type: object
                properties:
                  llm:
                    type: object
                    properties:
                      totalRequests: integer
                      successRate: number
                      avgLatency: integer
                  sessions:
                    type: object
                    properties:
                      active: integer
                      total: integer
```

### 4.2 WebSocket API

```go
type WSMessage struct {
    Type    string          `json:"type"`
    Payload json.RawMessage `json:"payload"`
}

type WSEvent struct {
    Type    string `json:"type"`
    Payload any    `json:"payload"`
}

const (
    EventTypeMessage  = "message"
    EventTypeStatus   = "status"
    EventTypeError    = "error"
    EventTypeHealth   = "health"
)

func (s *WSServer) handleConn(conn *websocket.Conn) {
    defer conn.Close()
    
    for {
        var msg WSMessage
        if err := conn.ReadJSON(&msg); err != nil {
            return
        }
        
        switch msg.Type {
        case "send":
            var payload struct {
                Channel   string `json:"channel"`
                Recipient string `json:"recipient"`
                Message   string `json:"message"`
            }
            json.Unmarshal(msg.Payload, &payload)
            s.sendToChannel(payload.Channel, payload.Recipient, payload.Message)
            
        case "subscribe":
            var payload struct {
                Channel string `json:"channel"`
            }
            json.Unmarshal(msg.Payload, &payload)
            s.subscribe(conn, payload.Channel)
        }
    }
}
```

## 5. 目录结构

```
mujibot/
├── cmd/
│   └── mujibot/
│       └── main.go              # 入口点
├── internal/
│   ├── agent/
│   │   ├── router.go            # 智能体路由器
│   │   └── agent.go             # 智能体定义
│   ├── channel/
│   │   ├── manager.go           # 渠道管理器
│   │   ├── telegram.go          # Telegram 渠道
│   │   ├── discord.go           # Discord 渠道
│   │   └── websocket.go         # WebSocket 渠道
│   ├── config/
│   │   ├── manager.go           # 配置管理器
│   │   ├── config.go            # 配置结构
│   │   └── validator.go         # 配置验证
│   ├── llm/
│   │   ├── client.go            # LLM 客户端
│   │   ├── openai.go            # OpenAI 提供商
│   │   ├── anthropic.go         # Anthropic 提供商
│   │   └── ollama.go            # Ollama 提供商
│   ├── memory/
│   │   └── store.go             # 记忆存储
│   ├── server/
│   │   ├── http.go              # HTTP 服务器
│   │   └── health.go            # 健康检查
│   ├── session/
│   │   └── manager.go           # 会话管理器
│   └── tool/
│       ├── executor.go          # 工具执行器
│       ├── read.go              # 读取文件工具
│       ├── write.go             # 写入文件工具
│       ├── edit.go              # 编辑文件工具
│       ├── exec.go              # 执行命令工具
│       ├── grep.go              # 搜索工具
│       └── find.go              # 查找工具
├── pkg/
│   └── json5/
│       └── decode.go            # JSON5 解析器
├── configs/
│   └── config.example.json5     # 示例配置
├── scripts/
│   ├── build.sh                 # 构建脚本
│   └── install.sh               # 安装脚本
├── deployments/
│   └── systemd/
│       └── mujibot.service      # systemd 服务文件
├── go.mod
├── go.sum
├── Makefile
└── README.md
```

## 6. 错误处理

### 6.1 错误类型

```go
package errors

type Error struct {
    Code    string
    Message string
    Cause   error
}

const (
    ErrCodeConfig       = "CONFIG_ERROR"
    ErrCodeAuth         = "AUTH_ERROR"
    ErrCodeChannel      = "CHANNEL_ERROR"
    ErrCodeLLM          = "LLM_ERROR"
    ErrCodeTool         = "TOOL_ERROR"
    ErrCodeSession      = "SESSION_ERROR"
    ErrCodeTimeout      = "TIMEOUT_ERROR"
)

func New(code, message string, cause error) *Error {
    return &Error{
        Code:    code,
        Message: message,
        Cause:   cause,
    }
}

func (e *Error) Error() string {
    if e.Cause != nil {
        return fmt.Sprintf("[%s] %s: %v", e.Code, e.Message, e.Cause)
    }
    return fmt.Sprintf("[%s] %s", e.Code, e.Message)
}

func (e *Error) Unwrap() error {
    return e.Cause
}
```

### 6.2 Panic Recovery

```go
func RecoveryMiddleware(next Handler) Handler {
    return func(ctx context.Context, msg *channel.Message) (resp *Response, err error) {
        defer func() {
            if r := recover(); r != nil {
                stack := debug.Stack()
                log.Error().
                    Str("panic", fmt.Sprintf("%v", r)).
                    Str("stack", string(stack)).
                    Msg("Recovered from panic")
                err = errors.New(errors.ErrCodeInternal, "Internal error", nil)
            }
        }()
        return next(ctx, msg)
    }
}
```

## 7. 性能优化

### 7.1 对象池

```go
var messagePool = sync.Pool{
    New: func() any {
        return &channel.Message{
            Metadata: make(map[string]any, 4),
        }
    },
}

func acquireMessage() *channel.Message {
    return messagePool.Get().(*channel.Message)
}

func releaseMessage(m *channel.Message) {
    m.ID = ""
    m.Channel = ""
    m.Content = ""
    for k := range m.Metadata {
        delete(m.Metadata, k)
    }
    messagePool.Put(m)
}
```

### 7.2 内存监控

```go
func monitorMemory() {
    ticker := time.NewTicker(30 * time.Second)
    for range ticker.C {
        var m runtime.MemStats
        runtime.ReadMemStats(&m)
        
        if m.Alloc > 80*1024*1024 {
            log.Warn().
                Uint64("alloc_mb", m.Alloc/1024/1024).
                Msg("Memory usage high, triggering GC")
            runtime.GC()
        }
    }
}
```

### 7.3 并发控制

```go
type Semaphore struct {
    ch chan struct{}
}

func NewSemaphore(max int) *Semaphore {
    return &Semaphore{
        ch: make(chan struct{}, max),
    }
}

func (s *Semaphore) Acquire(ctx context.Context) error {
    select {
    case s.ch <- struct{}{}:
        return nil
    case <-ctx.Done():
        return ctx.Err()
    }
}

func (s *Semaphore) Release() {
    <-s.ch
}

var msgSemaphore = NewSemaphore(10)

func (r *Router) Process(ctx context.Context, msg *channel.Message) (*Response, error) {
    if err := msgSemaphore.Acquire(ctx); err != nil {
        return nil, err
    }
    defer msgSemaphore.Release()
    
    return r.process(ctx, msg)
}
```

## 8. 部署

### 8.1 systemd 服务

```ini
[Unit]
Description=Mujibot AI Assistant
After=network.target

[Service]
Type=simple
User=mujibot
Group=mujibot
WorkingDirectory=/home/mujibot
ExecStart=/usr/local/bin/mujibot --config /etc/mujibot/config.json5
Restart=on-failure
RestartSec=5
LimitNOFILE=65536
Environment=TELEGRAM_BOT_TOKEN=
Environment=OPENAI_API_KEY=

[Install]
WantedBy=multi-user.target
```

### 8.2 构建脚本

```bash
#!/bin/bash

VERSION=${1:-"dev"}
OUTPUT_DIR="./dist"

build() {
    GOOS=$1
    GOARCH=$2
    GOARM=$3
    NAME="mujibot-${GOOS}-${GOARCH}"
    
    if [ -n "$GOARM" ]; then
        NAME="mujibot-${GOOS}-${GOARCH}v${GOARM}"
        GOARM_FLAG="GOARM=${GOARM}"
    fi
    
    echo "Building ${NAME}..."
    
    CGO_ENABLED=0 GOOS=$GOOS GOARCH=$GOARCH ${GOARM_FLAG} go build \
        -ldflags="-s -w -X main.Version=${VERSION}" \
        -trimpath \
        -o "${OUTPUT_DIR}/${NAME}" \
        ./cmd/mujibot
    
    if [ "$GOOS" = "linux" ]; then
        upx --best "${OUTPUT_DIR}/${NAME}" 2>/dev/null || true
    fi
}

mkdir -p $OUTPUT_DIR

build linux arm 7
build linux arm64
build linux amd64
build darwin arm64
build darwin amd64
build windows amd64

echo "Build complete. Binaries in ${OUTPUT_DIR}/"
```

## 9. 测试策略

### 9.1 单元测试

```go
func TestSessionManager_GetOrCreate(t *testing.T) {
    mgr := session.NewManager(20, time.Hour)
    
    sess1 := mgr.GetOrCreate("user1", "telegram")
    assert.NotNil(t, sess1)
    assert.Equal(t, "telegram:user1", sess1.ID)
    
    sess2 := mgr.GetOrCreate("user1", "telegram")
    assert.Same(t, sess1, sess2)
}

func TestToolExecutor_DangerousCommand(t *testing.T) {
    exec := tool.NewExecTool("/tmp", 30*time.Second, func(cmd string) bool {
        return false
    })
    
    result, err := exec.Execute(context.Background(), map[string]any{
        "command": "rm -rf /",
    })
    
    assert.NoError(t, err)
    assert.Contains(t, result, "cancelled")
}
```

### 9.2 集成测试

```go
func TestMessageFlow(t *testing.T) {
    ctx := context.Background()
    
    cfg := config.LoadTestConfig()
    mgr := channel.NewManager()
    router := agent.NewRouter(cfg, ...)
    
    msg := &channel.Message{
        Channel: "telegram",
        Sender:  channel.Sender{ID: "test-user"},
        Content: "Hello",
    }
    
    resp, err := router.Process(ctx, msg)
    
    assert.NoError(t, err)
    assert.NotEmpty(t, resp.Content)
}
```

## 10. 监控和日志

### 10.1 结构化日志

```go
func setupLogger(cfg *config.LoggingConfig) *slog.Logger {
    var handler slog.Handler
    
    opts := &slog.HandlerOptions{
        Level: parseLevel(cfg.Level),
    }
    
    if cfg.Format == "json" {
        handler = slog.NewJSONHandler(os.Stdout, opts)
    } else {
        handler = slog.NewTextHandler(os.Stdout, opts)
    }
    
    return slog.New(handler)
}

func logMiddleware(logger *slog.Logger) func(Handler) Handler {
    return func(next Handler) Handler {
        return func(ctx context.Context, msg *channel.Message) (*Response, error) {
            start := time.Now()
            resp, err := next(ctx, msg)
            
            logger.Info("message processed",
                "channel", msg.Channel,
                "sender", msg.Sender.ID,
                "duration", time.Since(start),
                "error", err,
            )
            
            return resp, err
        }
    }
}
```

### 10.2 指标收集

```go
type Metrics struct {
    messagesTotal    prometheus.Counter
    messagesActive   prometheus.Gauge
    llmRequests      *prometheus.CounterVec
    llmLatency       prometheus.Histogram
    memoryUsage      prometheus.Gauge
}

func NewMetrics() *Metrics {
    return &Metrics{
        messagesTotal: prometheus.NewCounter(prometheus.CounterOpts{
            Name: "mujibot_messages_total",
            Help: "Total number of messages processed",
        }),
        messagesActive: prometheus.NewGauge(prometheus.GaugeOpts{
            Name: "mujibot_messages_active",
            Help: "Number of messages being processed",
        }),
        llmRequests: prometheus.NewCounterVec(prometheus.CounterOpts{
            Name: "mujibot_llm_requests_total",
            Help: "Total LLM API requests",
        }, []string{"provider", "status"}),
        llmLatency: prometheus.NewHistogram(prometheus.HistogramOpts{
            Name: "mujibot_llm_latency_seconds",
            Help: "LLM API request latency",
            Buckets: []float64{.1, .5, 1, 2, 5, 10, 30, 60},
        }),
        memoryUsage: prometheus.NewGauge(prometheus.GaugeOpts{
            Name: "mujibot_memory_bytes",
            Help: "Current memory usage",
        }),
    }
}
```
