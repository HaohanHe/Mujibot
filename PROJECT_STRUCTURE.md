# Mujibot 项目结构

```
mujibot/
├── cmd/
│   └── mujibot/
│       └── main.go              # 应用程序入口
├── internal/
│   ├── agent/
│   │   └── router.go            # 智能体路由器和Agent管理
│   ├── channel/
│   │   ├── telegram/
│   │   │   └── telegram.go      # Telegram Bot集成
│   │   ├── discord/
│   │   │   └── discord.go       # Discord Bot集成
│   │   └── feishu/
│   │       └── feishu.go        # 飞书Bot集成
│   ├── config/
│   │   └── config.go            # JSON5配置管理、热重载
│   ├── gateway/
│   │   └── gateway.go           # 核心网关、消息路由
│   ├── health/
│   │   └── checker.go           # 健康检查、系统监控
│   ├── llm/
│   │   └── provider.go          # LLM提供商集成
│   ├── logger/
│   │   └── logger.go            # 结构化日志、轮转
│   ├── session/
│   │   └── session.go           # 会话管理、LRU缓存
│   ├── tools/
│   │   └── manager.go           # 工具系统、安全沙箱
│   └── web/
│       └── server.go            # Web调试界面、SSE
├── pkg/
│   └── utils/
│       └── utils.go             # 通用工具函数
├── scripts/
│   ├── install.sh               # 一键安装脚本
│   └── mujibot.service          # systemd服务文件
├── build/
│   └── (编译输出目录)
├── config.json5.example         # 配置文件示例
├── docker-compose.yml           # Docker Compose配置
├── Dockerfile                   # Docker镜像构建
├── go.mod                       # Go模块定义
├── go.sum                       # Go依赖校验
├── Makefile                     # 构建脚本
├── README.md                    # 项目文档
└── LICENSE                      # MIT许可证
```

## 模块说明

### 核心模块

| 模块 | 文件 | 说明 |
|------|------|------|
| Gateway | `internal/gateway/gateway.go` | 核心网关，管理所有组件生命周期 |
| Config | `internal/config/config.go` | JSON5配置解析、热重载、环境变量 |
| Logger | `internal/logger/logger.go` | 结构化JSON日志、文件轮转 |

### 消息渠道

| 渠道 | 文件 | 说明 |
|------|------|------|
| Telegram | `internal/channel/telegram/telegram.go` | Telegram Bot API集成 |
| Discord | `internal/channel/discord/discord.go` | Discord Bot API集成 |
| Feishu | `internal/channel/feishu/feishu.go` | 飞书开放平台集成 |

### AI功能

| 模块 | 文件 | 说明 |
|------|------|------|
| LLM | `internal/llm/provider.go` | OpenAI、Claude、Ollama集成 |
| Agent | `internal/agent/router.go` | 智能体路由、多Agent管理 |
| Session | `internal/session/session.go` | 会话上下文、LRU缓存 |
| Tools | `internal/tools/manager.go` | 文件操作、命令执行、安全沙箱 |

### 监控和调试

| 模块 | 文件 | 说明 |
|------|------|------|
| Health | `internal/health/checker.go` | 健康检查、系统指标 |
| Web | `internal/web/server.go` | Web调试界面、实时日志SSE |

## 依赖关系

```
main.go
  └── gateway
      ├── config
      ├── logger
      ├── session
      ├── tools
      ├── llm
      ├── agent (依赖 llm, tools, session)
      ├── health
      ├── web (依赖 config, session, agent, health)
      ├── telegram
      ├── discord
      └── feishu
```

## 编译输出

```
build/
├── mujibot              # 当前架构
├── mujibot-armv7        # ARMv7 (玩客云)
├── mujibot-arm64        # ARM64 (树莓派4)
└── mujibot-amd64        # x86_64
```

## 运行时文件

```
/opt/mujibot/
├── config.json5         # 配置文件
├── workspace/           # 工具工作目录
└── memory/              # 记忆存储（可选）

/var/log/mujibot/
└── app.log              # 应用日志
```
