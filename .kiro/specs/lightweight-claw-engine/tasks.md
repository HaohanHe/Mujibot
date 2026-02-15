# 任务列表

## 项目概览

Mujibot 轻量级 Claw 引擎 - 为低配置 ARM 设备设计的 AI 助手系统

**目标平台**: 玩客云 (1GB RAM, 8GB eMMC, armv7)  
**技术栈**: Go 1.21+, 静态链接, 零外部依赖

---

## Phase 1: 项目初始化

### 1.1 项目结构搭建
- [ ] 创建 Go 模块 `go mod init github.com/mujibot/mujibot`
- [ ] 创建标准目录结构
  - [ ] `cmd/mujibot/` - 入口点
  - [ ] `internal/` - 内部包
  - [ ] `pkg/` - 公共包
  - [ ] `configs/` - 配置文件
  - [ ] `scripts/` - 构建脚本
  - [ ] `deployments/` - 部署文件
- [ ] 创建 `.gitignore`
- [ ] 创建 `Makefile`

### 1.2 基础配置
- [ ] 配置 Go 版本约束 (go 1.21+)
- [ ] 添加必要依赖
  - [ ] `github.com/fsnotify/fsnotify` - 文件监控
  - [ ] `golang.org/x/net/websocket` - WebSocket
  - [ ] `gopkg.in/telebot.v3` - Telegram Bot
  - [ ] `github.com/bwmarrin/discordgo` - Discord Bot
- [ ] 配置静态编译选项 (`CGO_ENABLED=0`)

---

## Phase 2: 核心基础设施

### 2.1 配置管理器 (`internal/config`)
- [ ] 定义配置结构体
  ```go
  type Config struct {
      Server   ServerConfig
      Channels ChannelsConfig
      LLM      LLMConfig
      Agents   AgentsConfig
      Tools    ToolsConfig
      Session  SessionConfig
      Logging  LoggingConfig
  }
  ```
- [ ] 实现 JSON5 解析器 (`pkg/json5`)
  - [ ] 支持注释
  - [ ] 支持尾随逗号
  - [ ] 支持多行字符串
- [ ] 实现配置加载
  - [ ] 文件读取
  - [ ] 环境变量展开 `${VAR_NAME}`
  - [ ] 配置验证
- [ ] 实现热重载
  - [ ] 使用 fsnotify 监控文件变化
  - [ ] 配置变更回调机制
- [ ] 实现默认配置生成
- [ ] 编写单元测试

### 2.2 日志系统 (`internal/logger`)
- [ ] 封装 `log/slog` 标准库
- [ ] 支持多日志级别 (DEBUG, INFO, WARN, ERROR)
- [ ] 支持 JSON 格式输出
- [ ] 支持日志文件轮转 (5MB)
- [ ] 敏感信息过滤 (API Key, Token)
- [ ] 编写单元测试

### 2.3 错误处理 (`internal/errors`)
- [ ] 定义错误类型
  ```go
  type Error struct {
      Code    string
      Message string
      Cause   error
  }
  ```
- [ ] 定义错误码常量
- [ ] 实现 `Unwrap()` 支持错误链
- [ ] 编写单元测试

---

## Phase 3: 消息渠道层

### 3.1 渠道接口定义 (`internal/channel`)
- [ ] 定义 `Channel` 接口
  ```go
  type Channel interface {
      Name() string
      Connect(ctx context.Context) error
      Disconnect() error
      Send(ctx context.Context, recipient string, message string) error
      Messages() <-chan *Message
      IsConnected() bool
  }
  ```
- [ ] 定义 `Message` 结构体
- [ ] 定义 `Sender` 结构体

### 3.2 渠道管理器 (`internal/channel/manager.go`)
- [ ] 实现渠道注册和注销
- [ ] 实现消息聚合 channel
- [ ] 实现连接状态管理
- [ ] 实现重连机制 (指数退避)
- [ ] 编写单元测试

### 3.3 Telegram 渠道 (`internal/channel/telegram.go`)
- [ ] 使用 `gopkg.in/telebot.v3`
- [ ] 实现 Bot 连接
- [ ] 实现消息接收和转换
- [ ] 实现消息发送
- [ ] 实现用户白名单验证
- [ ] 实现群组消息处理
- [ ] 编写集成测试

### 3.4 Discord 渠道 (`internal/channel/discord.go`)
- [ ] 使用 `github.com/bwmarrin/discordgo`
- [ ] 实现 Bot 连接
- [ ] 实现消息接收和转换
- [ ] 实现消息发送
- [ ] 实现服务器白名单验证
- [ ] 编写集成测试

### 3.5 WebSocket 渠道 (`internal/channel/websocket.go`)
- [ ] 实现 WebSocket 服务器
- [ ] 定义消息协议
- [ ] 实现客户端订阅机制
- [ ] 编写单元测试

---

## Phase 4: 会话管理层

### 4.1 会话管理器 (`internal/session`)
- [ ] 定义 `Session` 结构体
- [ ] 实现 `Manager` 
  - [ ] 会话创建和获取
  - [ ] 消息历史管理 (最大20条)
  - [ ] LRU 淘汰策略
  - [ ] 空闲超时清理 (1小时)
- [ ] 实现会话持久化 (可选)
- [ ] 编写单元测试
- [ ] 性能测试 (内存占用)

---

## Phase 5: LLM 客户端层

### 5.1 LLM 客户端接口 (`internal/llm`)
- [ ] 定义 `Provider` 接口
  ```go
  type Provider interface {
      Name() string
      Chat(ctx context.Context, req *Request) (*Response, error)
      ChatStream(ctx context.Context, req *Request) (<-chan StreamChunk, error)
  }
  ```
- [ ] 定义请求/响应结构体
- [ ] 定义工具调用结构体

### 5.2 LLM 客户端 (`internal/llm/client.go`)
- [ ] 实现多提供商管理
- [ ] 实现重试机制 (指数退避, 最多3次)
- [ ] 实现超时控制 (60秒)
- [ ] 实现流式响应处理
- [ ] 编写单元测试

### 5.3 OpenAI 提供商 (`internal/llm/openai.go`)
- [ ] 实现 Chat Completions API 调用
- [ ] 实现流式响应解析
- [ ] 实现工具调用支持
- [ ] 支持自定义 BaseURL (兼容 API)
- [ ] 编写集成测试

### 5.4 Anthropic 提供商 (`internal/llm/anthropic.go`)
- [ ] 实现 Messages API 调用
- [ ] 实现流式响应解析
- [ ] 实现工具调用支持
- [ ] 编写集成测试

### 5.5 Ollama 提供商 (`internal/llm/ollama.go`)
- [ ] 实现 Generate API 调用
- [ ] 实现流式响应解析
- [ ] 编写集成测试

---

## Phase 6: 工具执行层

### 6.1 工具接口定义 (`internal/tool`)
- [ ] 定义 `Tool` 接口
  ```go
  type Tool interface {
      Name() string
      Description() string
      Parameters() map[string]Parameter
      Execute(ctx context.Context, args map[string]any) (string, error)
  }
  ```
- [ ] 定义参数结构体

### 6.2 工具执行器 (`internal/tool/executor.go`)
- [ ] 实现工具注册
- [ ] 实现工具调用
- [ ] 实现危险命令确认机制
- [ ] 实现沙箱路径限制
- [ ] 实现超时控制 (30秒)
- [ ] 生成 OpenAI 工具定义
- [ ] 编写单元测试

### 6.3 内置工具实现
- [ ] `read` - 读取文件
  - [ ] 路径安全检查
  - [ ] 文件大小限制 (1MB)
- [ ] `write` - 写入文件
  - [ ] 路径安全检查
  - [ ] 覆盖确认
- [ ] `edit` - 编辑文件
  - [ ] 精确搜索替换
- [ ] `ls` - 列出目录
- [ ] `grep` - 搜索内容
- [ ] `find` - 查找文件
- [ ] `exec` - 执行命令
  - [ ] 危险命令检测
  - [ ] 超时控制
  - [ ] 输出截断

---

## Phase 7: 智能体层

### 7.1 智能体定义 (`internal/agent`)
- [ ] 定义 `Agent` 结构体
- [ ] 定义系统提示词模板

### 7.2 智能体路由器 (`internal/agent/router.go`)
- [ ] 实现消息路由
  - [ ] 按发送者路由
  - [ ] 按群组路由
  - [ ] 按渠道路由
- [ ] 实现多智能体隔离
- [ ] 实现消息处理流程
- [ ] 实现工具调用循环
- [ ] 实现 Panic Recovery
- [ ] 编写单元测试

---

## Phase 8: 记忆系统

### 8.1 记忆存储 (`internal/memory`)
- [ ] 实现每日笔记存储
- [ ] 实现长期记忆存储
- [ ] 实现记忆加载
- [ ] 实现关键词搜索
- [ ] 文件大小限制 (100KB)
- [ ] 编写单元测试

---

## Phase 9: HTTP 服务层

### 9.1 HTTP 服务器 (`internal/server`)
- [ ] 实现健康检查端点 `/health`
- [ ] 实现指标端点 `/metrics`
- [ ] 实现优雅关闭
- [ ] 编写单元测试

### 9.2 健康检查 (`internal/server/health.go`)
- [ ] 返回内存使用
- [ ] 返回运行时间
- [ ] 返回渠道连接状态
- [ ] 返回消息统计

---

## Phase 10: 主程序入口

### 10.1 主程序 (`cmd/mujibot/main.go`)
- [ ] 解析命令行参数
  - [ ] `--config` 配置文件路径
  - [ ] `--version` 版本信息
  - [ ] `--help` 帮助信息
- [ ] 初始化日志
- [ ] 加载配置
- [ ] 初始化所有组件
- [ ] 启动服务
- [ ] 处理信号 (SIGINT, SIGTERM)
- [ ] 优雅关闭

---

## Phase 11: 性能优化

### 11.1 内存优化
- [ ] 实现 `sync.Pool` 对象复用
- [ ] 实现内存监控
- [ ] 实现自动 GC 触发 (>80MB)
- [ ] 内存泄漏检测

### 11.2 并发优化
- [ ] 实现并发控制信号量 (最大10)
- [ ] Goroutine 泄漏检测

### 11.3 二进制优化
- [ ] 编译优化 (`-ldflags="-s -w"`)
- [ ] UPX 压缩
- [ ] 二进制大小测试 (<15MB)

---

## Phase 12: 部署和发布

### 12.1 构建脚本 (`scripts/build.sh`)
- [ ] Linux armv7 构建
- [ ] Linux arm64 构建
- [ ] Linux amd64 构建
- [ ] Darwin arm64 构建
- [ ] Darwin amd64 构建
- [ ] Windows amd64 构建

### 12.2 systemd 服务 (`deployments/systemd/mujibot.service`)
- [ ] 服务文件编写
- [ ] 安装脚本

### 12.3 安装脚本 (`scripts/install.sh`)
- [ ] 下载二进制
- [ ] 创建用户
- [ ] 安装服务
- [ ] 生成默认配置

---

## Phase 13: 测试和文档

### 13.1 测试
- [ ] 单元测试覆盖率 >80%
- [ ] 集成测试
- [ ] 性能测试
  - [ ] 内存占用测试 (<10MB 空闲)
  - [ ] 启动时间测试 (<2秒)
  - [ ] 并发处理测试

### 13.2 文档
- [ ] README.md
- [ ] 配置说明文档
- [ ] API 文档
- [ ] 部署文档

---

## 里程碑

| 里程碑 | 阶段 | 预计完成 | 状态 |
|--------|------|----------|------|
| M1: 项目骨架 | Phase 1 | - | ⏳ 待开始 |
| M2: 核心基础 | Phase 2-3 | - | ⏳ 待开始 |
| M3: LLM 集成 | Phase 4-5 | - | ⏳ 待开始 |
| M4: 工具系统 | Phase 6 | - | ⏳ 待开始 |
| M5: 智能体完整 | Phase 7-8 | - | ⏳ 待开始 |
| M6: 服务就绪 | Phase 9-10 | - | ⏳ 待开始 |
| M7: 优化发布 | Phase 11-13 | - | ⏳ 待开始 |

---

## 依赖清单

| 包名 | 版本 | 用途 |
|------|------|------|
| `gopkg.in/telebot.v3` | latest | Telegram Bot |
| `github.com/bwmarrin/discordgo` | latest | Discord Bot |
| `github.com/fsnotify/fsnotify` | v1.7.0 | 文件监控 |
| `golang.org/x/net` | latest | WebSocket |

---

## 风险和缓解

| 风险 | 影响 | 缓解措施 |
|------|------|----------|
| Telegram/Discord API 变更 | 高 | 抽象接口层，便于替换实现 |
| 内存超限 | 高 | 严格内存监控，自动 GC |
| LLM API 不稳定 | 中 | 重试机制，多提供商支持 |
| 二进制过大 | 中 | 编译优化，UPX 压缩 |

---

## 开发规范

### 代码规范
- 遵循 Go 官方代码规范
- 使用 `gofmt` 格式化
- 使用 `golint` 检查
- 不使用 cgo
- 不使用反射

### 提交规范
- `feat:` 新功能
- `fix:` 修复
- `docs:` 文档
- `test:` 测试
- `refactor:` 重构
- `perf:` 性能优化

### 分支策略
- `main` - 稳定版本
- `develop` - 开发版本
- `feature/*` - 功能分支
- `hotfix/*` - 热修复分支
