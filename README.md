# Mujibot 轻量级AI助手引擎

[![Version](https://img.shields.io/badge/version-1.0.0-blue.svg)](https://github.com/HaohanHe/mujibot)
[![Go Version](https://img.shields.io/badge/go-1.21+-00ADD8.svg)](https://golang.org)
[![License](https://img.shields.io/badge/license-MIT-green.svg)](LICENSE)

Mujibot是一个资源高效的个人AI助手系统，专为低配置ARM设备（如玩客云，仅1GB RAM和8GB eMMC存储）设计。该系统基于OpenClaw架构理念，使用Go语言重新实现，提供单二进制部署、极低资源占用（<10MB RAM空闲）和快速启动的特性。

## 特性

- **极致轻量**: 空闲内存占用 <10MB，峰值 <50MB
- **单二进制文件**: 静态链接，无需依赖，<15MB（UPX压缩后<8MB）
- **多渠道支持**: Telegram、Discord、飞书
- **多智能体**: 支持多个隔离的AI智能体实例
- **LLM集成**: 支持OpenAI、Anthropic Claude、Ollama本地模型
- **工具系统**: 文件操作、命令执行、安全沙箱
- **会话管理**: LRU缓存、自动清理、上下文保持
- **热重载**: 配置文件变更无需重启
- **Web调试界面**: 实时日志、系统状态、消息调试
- **健康监控**: HTTP端点、内存监控、自动GC

## 系统要求

### 目标硬件

| 组件 | 最低配置 | 推荐配置 |
|------|---------|---------|
| CPU | ARM Cortex-A5 1.5GHz (4核) | ARM Cortex-A53 1.8GHz |
| 内存 | 512MB | 1GB+ |
| 存储 | 50MB | 100MB+ |
| 网络 | 有线/无线 | 有线千兆 |

### 支持平台

- 玩客云 (晶晨S805, ARMv7)
- 树莓派 Zero 2W / 3B / 4B
- 其他Armbian支持的ARM设备
- x86_64 Linux服务器

## 快速开始

### 一键安装

```bash
# 使用安装脚本（需要root权限）
curl -fsSL https://raw.githubusercontent.com/HaohanHe/mujibot/main/scripts/install.sh | sudo bash
```

### 手动安装

1. **下载二进制文件**

```bash
# 检测架构
ARCH=$(uname -m)
case $ARCH in
    x86_64) TARGET="amd64" ;;
    aarch64|arm64) TARGET="arm64" ;;
    armv7l|armv7) TARGET="armv7" ;;
esac

# 下载
wget https://github.com/HaohanHe/mujibot/releases/download/v1.0.0/mujibot-1.0.0-linux-${TARGET}.tar.gz
tar -xzf mujibot-1.0.0-linux-${TARGET}.tar.gz
sudo cp mujibot-1.0.0-linux-${TARGET}/mujibot /usr/local/bin/
```

2. **创建配置目录**

```bash
sudo mkdir -p /opt/mujibot
sudo mkdir -p /var/log/mujibot
```

3. **创建配置文件**

```bash
sudo mujibot --config /opt/mujibot/config.json5
# 首次运行会自动创建默认配置文件
```

4. **编辑配置**

```bash
sudo nano /opt/mujibot/config.json5
```

5. **设置环境变量**

```bash
export TELEGRAM_BOT_TOKEN="your_telegram_bot_token"
export OPENAI_API_KEY="your_openai_api_key"
```

6. **启动服务**

```bash
sudo systemctl enable --now mujibot
```

## 配置

### 配置文件示例

```json
{
  "server": {
    "port": 8080,
    "healthCheck": true
  },
  "channels": {
    "telegram": {
      "enabled": true,
      "token": "${TELEGRAM_BOT_TOKEN}",
      "allowedUsers": [123456789]
    },
    "discord": {
      "enabled": false,
      "token": "${DISCORD_BOT_TOKEN}",
      "allowedGuilds": []
    },
    "feishu": {
      "enabled": false,
      "appId": "${FEISHU_APP_ID}",
      "appSecret": "${FEISHU_APP_SECRET}",
      "encryptKey": "${FEISHU_ENCRYPT_KEY}",
      "allowedUsers": []
    }
  },
  "llm": {
    "provider": "openai",
    "model": "gpt-4o-mini",
    "apiKey": "${OPENAI_API_KEY}",
    "baseURL": "",
    "timeout": 60,
    "maxRetries": 3
  },
  "agents": {
    "default": {
      "name": "Mujibot",
      "systemPrompt": "你是一个运行在低功耗设备上的AI助手。",
      "tools": ["read_file", "write_file", "execute_command", "list_directory"]
    }
  },
  "tools": {
    "workDir": "/opt/mujibot/workspace",
    "timeout": 30,
    "confirmDangerous": true,
    "blockedCommands": ["reboot", "shutdown", "init", "poweroff"]
  },
  "session": {
    "maxMessages": 20,
    "idleTimeout": 3600,
    "maxSessions": 100
  },
  "logging": {
    "level": "info",
    "file": "/var/log/mujibot/app.log",
    "maxSize": 5,
    "format": "json"
  }
}
```

### 环境变量

| 变量 | 说明 | 必需 |
|------|------|------|
| `TELEGRAM_BOT_TOKEN` | Telegram Bot API Token | 可选 |
| `DISCORD_BOT_TOKEN` | Discord Bot API Token | 可选 |
| `FEISHU_APP_ID` | 飞书应用ID | 可选 |
| `FEISHU_APP_SECRET` | 飞书应用密钥 | 可选 |
| `FEISHU_ENCRYPT_KEY` | 飞书加密密钥 | 可选 |
| `OPENAI_API_KEY` | OpenAI API密钥 | 条件 |
| `ANTHROPIC_API_KEY` | Anthropic API密钥 | 条件 |

## 构建

### 从源码构建

```bash
# 克隆仓库
git clone https://github.com/HaohanHe/mujibot.git
cd mujibot

# 安装依赖
make deps

# 构建当前架构
make build

# 构建所有架构
make build-all

# UPX压缩
make compress
```

### 交叉编译

```bash
# ARMv7 (玩客云)
make build-armv7

# ARM64 (树莓派4)
make build-arm64

# x86_64
make build-amd64
```

## 监控

### Web调试界面

启动后访问 `http://<ip>:8080` 查看：

- 实时系统状态（内存、Goroutines、会话数）
- 实时消息日志
- 智能体列表
- 消息调试（直接发送测试消息）

### API端点

| 端点 | 说明 |
|------|------|
| `GET /api/status` | 系统状态 |
| `GET /api/logs` | 最近日志 |
| `GET /api/sessions` | 会话统计 |
| `GET /api/agents` | 智能体列表 |
| `GET /api/config` | 配置信息 |
| `POST /api/send` | 发送测试消息 |

### 健康检查

```bash
curl http://localhost:8080/api/status
```

## 日志

```bash
# 查看实时日志
sudo journalctl -u mujibot -f

# 查看最近100行
sudo journalctl -u mujibot -n 100

# 查看日志文件
sudo tail -f /var/log/mujibot/app.log
```

## 故障排除

### 内存使用过高

```bash
# 查看内存使用
curl http://localhost:8080/api/status | jq .memory

# 手动触发GC
sudo systemctl kill -s USR1 mujibot
```

### 服务无法启动

```bash
# 检查配置
sudo mujibot --config /opt/mujibot/config.json5 --help

# 查看详细日志
sudo journalctl -u mujibot -n 200 --no-pager
```

### Telegram Bot不响应

1. 检查Token是否正确
2. 确认已向Bot发送 `/start`
3. 检查用户ID是否在白名单中

### 飞书Webhook配置

1. 在飞书开放平台创建应用
2. 启用机器人功能
3. 配置事件订阅URL: `http://<your-server>:8080/webhook/feishu`
4. 订阅 `im.message.receive_v1` 事件

## 文档

- [快速入门](QUICKSTART.md)
- [常见问题](FAQ.md)
- [开发指南](DEVELOPMENT.md)
- [API文档](API.md)

## 贡献

欢迎提交Issue和PR！

```bash
# 开发模式运行
make dev
make run

# 运行测试
make test

# 代码格式化
make fmt
```

## 许可证

MIT License - 详见 [LICENSE](LICENSE) 文件

## 致谢

- 基于 [OpenClaw](https://github.com/claw-project/claw) 架构理念
- 使用 Go 标准库实现，保持轻量

## 联系方式

- GitHub Issues: [github.com/HaohanHe/mujibot/issues](https://github.com/HaohanHe/mujibot/issues)
- 邮箱: [bugreport@hsyscn.top](mailto:bugreport@hsyscn.top)
