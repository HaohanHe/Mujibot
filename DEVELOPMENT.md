# Mujibot 开发指南

## 开发环境

### 要求

- Go 1.21+
- Make
- Git

### 设置

```bash
# 克隆仓库
git clone https://github.com/HaohanHe/mujibot.git
cd mujibot

# 安装依赖
make deps

# 验证安装
go version
make build
```

## 项目结构

```
internal/
├── agent/          # 智能体路由
├── channel/        # 消息渠道
│   ├── telegram/
│   ├── discord/
│   └── feishu/
├── config/         # 配置管理
├── gateway/        # 核心网关
├── health/         # 健康检查
├── llm/            # LLM集成
├── logger/         # 日志系统
├── session/        # 会话管理
├── tools/          # 工具系统
└── web/            # Web界面
```

## 开发流程

### 1. 创建分支

```bash
git checkout -b feature/your-feature
```

### 2. 开发

```bash
# 开发模式（带调试信息）
make dev

# 运行测试
make test

# 格式化代码
make fmt
```

### 3. 测试

```bash
# 运行所有测试
go test ./...

# 运行特定包测试
go test ./internal/config/...

# 带覆盖率
go test -cover ./...
```

### 4. 提交

```bash
git add .
git commit -m "feat: add your feature"
git push origin feature/your-feature
```

## 添加新功能

### 添加新的消息渠道

1. 在 `internal/channel/` 创建新目录
2. 实现渠道接口：

```go
package mychannel

type Bot struct {
    // 配置
}

func NewBot(cfg config.MyChannelConfig, log *logger.Logger) *Bot {
    return &Bot{}
}

func (b *Bot) Start() error {
    // 启动连接
    return nil
}

func (b *Bot) Stop() {
    // 停止连接
}

func (b *Bot) OnMessage(handler MessageHandler) {
    // 注册处理器
}

func (b *Bot) SendMessage(chatID, content string) error {
    // 发送消息
    return nil
}
```

3. 在 `gateway.go` 中集成

### 添加新的LLM提供商

1. 在 `internal/llm/provider.go` 添加新提供商：

```go
type MyProvider struct {
    apiKey string
    // ...
}

func NewMyProvider(apiKey string, ...) *MyProvider {
    return &MyProvider{apiKey: apiKey}
}

func (p *MyProvider) Chat(messages []session.Message, tools []Tool) (*Response, error) {
    // 实现聊天逻辑
}

func (p *MyProvider) ChatStream(messages []session.Message, tools []Tool, callback func(chunk string)) (*Response, error) {
    // 实现流式聊天
}

func (p *MyProvider) GetModel() string {
    return p.model
}
```

2. 在 `NewProvider` 函数中添加创建逻辑

### 添加新的工具

1. 在 `internal/tools/manager.go` 添加新工具：

```go
type MyTool struct {
    manager *Manager
}

func (t *MyTool) Name() string {
    return "my_tool"
}

func (t *MyTool) Description() string {
    return "工具描述"
}

func (t *MyTool) Parameters() map[string]interface{} {
    return map[string]interface{}{
        "type": "object",
        "properties": map[string]interface{}{
            "param": map[string]interface{}{
                "type":        "string",
                "description": "参数描述",
            },
        },
        "required": []string{"param"},
    }
}

func (t *MyTool) Execute(args map[string]interface{}) (string, error) {
    // 实现工具逻辑
    return "result", nil
}
```

2. 在 `registerBuiltinTools` 中注册

## 调试

### 日志级别

```go
// 开发时使用debug级别
log, _ := logger.New(logger.Config{Level: "debug"})

log.Debug("debug message", "key", "value")
log.Info("info message")
log.Warn("warning")
log.Error("error", "err", err)
```

### 性能分析

```bash
# CPU分析
go tool pprof http://localhost:6060/debug/pprof/profile

# 内存分析
go tool pprof http://localhost:6060/debug/pprof/heap
```

### 内存优化

```go
// 使用sync.Pool复用对象
var bufferPool = sync.Pool{
    New: func() interface{} {
        return make([]byte, 1024)
    },
}

buf := bufferPool.Get().([]byte)
defer bufferPool.Put(buf)
```

## 测试

### 单元测试

```go
func TestNewManager(t *testing.T) {
    log, _ := logger.New(logger.Config{Level: "error"})
    mgr, err := NewManager("./test_config.json5", log)
    if err != nil {
        t.Fatalf("failed to create manager: %v", err)
    }
    defer mgr.Close()
    
    cfg := mgr.Get()
    if cfg.Server.Port == 0 {
        t.Error("port should not be zero")
    }
}
```

### 基准测试

```go
func BenchmarkProcessMessage(b *testing.B) {
    agent := createTestAgent()
    
    b.ResetTimer()
    for i := 0; i < b.N; i++ {
        agent.ProcessMessage("user1", "test", "Hello")
    }
}
```

## 构建

### 本地构建

```bash
# 开发模式
make dev

# 生产模式
make build

# 运行
make run
```

### 交叉编译

```bash
# ARMv7
make build-armv7

# ARM64
make build-arm64

# x86_64
make build-amd64

# 全部
make build-all
```

### UPX压缩

```bash
make compress
```

## 发布

### 创建Release

```bash
# 1. 更新版本号
# 2. 创建tag
git tag -a v1.0.0 -m "Release v1.0.0"
git push origin v1.0.0

# 3. 构建发布包
make release
```

### Docker

```bash
# 构建镜像
make docker

# 运行
docker-compose up -d
```

## 代码规范

### 命名规范

- 包名：小写，简短
- 结构体：PascalCase
- 接口：PascalCase，以`-er`结尾
- 函数：PascalCase（导出）/ camelCase（内部）
- 常量：PascalCase

### 注释规范

```go
// Manager 配置管理器
type Manager struct {
    // config 当前配置
    config *Config
}

// Load 从文件加载配置
// 返回错误如果文件不存在或格式无效
func (m *Manager) Load() error {
    // ...
}
```

### 错误处理

```go
// 包装错误
if err != nil {
    return fmt.Errorf("failed to load config: %w", err)
}

// 自定义错误
var ErrNotFound = errors.New("not found")

// 检查错误
if errors.Is(err, ErrNotFound) {
    // 处理未找到
}
```

## 性能目标

| 指标 | 目标 |
|------|------|
| 空闲内存 | <10MB |
| 峰值内存 | <50MB |
| 启动时间 | <2秒 |
| 消息延迟 | <2秒（不含LLM） |
| 二进制大小 | <15MB |

## 常见问题

### 编译错误

```bash
# 清理缓存
make clean
go clean -cache

# 重新下载依赖
make deps
```

### 运行时panic

```go
// 使用recover
defer func() {
    if r := recover(); r != nil {
        log.Error("panic recovered", "error", r, "stack", string(debug.Stack()))
    }
}()
```

## 贡献指南

1. Fork 仓库
2. 创建功能分支
3. 提交更改
4. 创建 Pull Request

### PR 要求

- 代码通过所有测试
- 添加必要的测试
- 更新文档
- 遵循代码规范

## 参考

- [Go Code Review Comments](https://github.com/golang/go/wiki/CodeReviewComments)
- [Effective Go](https://golang.org/doc/effective_go.html)
- [Go Modules](https://golang.org/ref/mod)
