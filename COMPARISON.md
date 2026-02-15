# Mujibot vs OpenClaw 功能对比

## 核心功能对比

| 功能 | OpenClaw | Mujibot | 说明 |
|------|----------|---------|------|
| **运行时** | Node.js >= 22 | Go 1.21+ | Mujibot 单二进制，无依赖 |
| **内存占用** | 较高 (Node.js) | <10MB 空闲 | Mujibot 针对低资源设备优化 |
| **二进制大小** | 需 Node 运行时 | <15MB | Mujibot 单文件部署 |

## 消息渠道

| 渠道 | OpenClaw | Mujibot | 备注 |
|------|----------|---------|------|
| Telegram | 支持 | 支持 | 完整实现 |
| Discord | 支持 | 支持 | 完整实现 |
| 飞书 | 不支持 | 支持 | Mujibot 特色 |
| WhatsApp | 支持 | 不支持 | OpenClaw 特色 |
| Slack | 支持 | 不支持 | - |
| Signal | 支持 | 不支持 | - |
| iMessage | 支持 | 不支持 | - |
| Microsoft Teams | 支持 | 不支持 | - |
| Google Chat | 支持 | 不支持 | - |
| Matrix | 支持 | 不支持 | - |
| WebChat | 支持 | Web调试界面 | 功能类似 |

## LLM 支持

| 提供商 | OpenClaw | Mujibot |
|--------|----------|---------|
| OpenAI | 支持 (OAuth + API Key) | 支持 (API Key) |
| Anthropic | 支持 (OAuth + API Key) | 支持 (API Key) |
| Ollama | 支持 | 支持 |
| 其他 OpenAI 兼容 | 支持 | 支持 |

## 工具系统

| 工具 | OpenClaw | Mujibot |
|------|----------|---------|
| 文件读取 | 支持 | read_file |
| 文件写入 | 支持 | write_file |
| 目录列表 | 支持 | list_directory |
| 命令执行 | 支持 | execute_command |
| 文本搜索 | 支持 | grep |
| 网页搜索 | 支持 | web_search |
| HTTP请求 | 支持 | http_request |
| 系统信息 | 支持 | get_system_info |
| 进程列表 | 支持 | list_processes |
| 记忆系统 | 支持 | memory_read/write |
| 浏览器控制 | 支持 | 不支持 |
| Canvas | 支持 | 不支持 |
| Cron 定时 | 支持 | 不支持 |

## 会话管理

| 功能 | OpenClaw | Mujibot |
|------|----------|---------|
| 会话历史 | 支持 | 支持 |
| LRU 淘汰 | 支持 | 支持 |
| 多会话隔离 | 支持 | 支持 |
| 上下文保持 | 支持 | 支持 |

## 安全特性

| 功能 | OpenClaw | Mujibot |
|------|----------|---------|
| 用户白名单 | 支持 | 支持 |
| DM 配对码 | 支持 | 不支持 |
| 命令黑名单 | 不适用 | 支持 |
| 危险操作确认 | 不适用 | 支持 |
| 工作目录限制 | 不适用 | 支持 |

## 部署方式

| 方式 | OpenClaw | Mujibot |
|------|----------|---------|
| 二进制部署 | 支持 (打包后) | 支持 (原生) |
| Docker | 支持 | 支持 |
| Systemd 服务 | 支持 | 支持 |
| 配置热重载 | 支持 | 支持 |

## 特色功能

### OpenClaw 特色
- 语音唤醒 + 通话模式 (macOS/iOS/Android)
- Live Canvas 可视化工作区
- macOS/iOS/Android 配套应用
- OAuth 订阅登录 (Claude Pro/Max, ChatGPT)
- 更多消息渠道

### Mujibot 特色
- 飞书支持
- 极低资源占用 (<10MB 空闲)
- 单二进制部署 (无运行时依赖)
- 专为 ARM 低功耗设备优化
- 内置记忆系统 (每日笔记)
- 命令执行安全沙箱

## 适用场景

| 场景 | 推荐 |
|------|------|
| 玩客云/树莓派等低功耗设备 | Mujibot |
| 需要飞书集成 | Mujibot |
| 需要 WhatsApp/Slack 等渠道 | OpenClaw |
| 需要语音交互 | OpenClaw |
| 需要可视化 Canvas | OpenClaw |
| 追求极简部署 | Mujibot |
| 已有 Node.js 环境 | OpenClaw |
| 资源受限环境 | Mujibot |

## 总结

Mujibot 是 OpenClaw 的轻量化 Go 实现，专注于：
1. **低资源设备** - 玩客云、树莓派等 ARM 设备
2. **单二进制部署** - 无需 Node.js 运行时
3. **国内用户友好** - 支持飞书
4. **安全沙箱** - 命令执行安全控制

OpenClaw 功能更全面，适合：
1. 需要更多消息渠道
2. 需要语音交互
3. 需要可视化 Canvas
4. 已有 Node.js 环境
