# Mujibot 项目总结

## 项目概述

Mujibot是一个为低配置ARM设备（如玩客云，1GB RAM / 8GB eMMC）设计的轻量级AI助手引擎。它基于OpenClaw架构理念，使用Go语言重新实现，提供单二进制部署、极低资源占用和快速启动的特性。

## 已实现功能

### ✅ 核心功能

1. **极致资源效率**
   - 空闲内存占用 <10MB
   - 峰值内存占用 <50MB
   - 二进制大小 <15MB（UPX压缩后<8MB）
   - 启动时间 <2秒

2. **跨平台兼容性**
   - 支持ARMv7、ARM64、x86_64架构
   - 静态链接，无依赖
   - 支持Armbian、Ubuntu、Debian、Alpine

3. **多渠道支持**
   - ✅ Telegram Bot
   - ✅ Discord Bot
   - ✅ 飞书 Bot

4. **LLM提供商集成**
   - ✅ OpenAI (GPT-4o, GPT-4o-mini)
   - ✅ Anthropic Claude
   - ✅ Ollama本地模型
   - ✅ 兼容API (DeepSeek、Moonshot等)

5. **智能体系统**
   - ✅ 多智能体路由
   - ✅ 隔离的会话上下文
   - ✅ 故障隔离

6. **工具系统**
   - ✅ 文件读取/写入
   - ✅ 目录列表
   - ✅ 命令执行
   - ✅ 系统信息获取
   - ✅ 代码补丁(apply_patch)
   - ✅ 网页搜索(web_search)
   - ✅ 文件搜索(grep)
   - ✅ 记忆读取(memory_read)
   - ✅ 记忆写入(memory_write)
   - ✅ 工作目录沙箱
   - ✅ 危险命令拦截

7. **会话管理**
   - ✅ LRU缓存
   - ✅ 自动清理
   - ✅ 消息历史限制
   - ✅ 空闲超时

8. **记忆系统**
   - ✅ 每日笔记(memory/YYYY-MM-DD.md)
   - ✅ 长期记忆(MEMORY.md)
   - ✅ 记忆搜索
   - ✅ 记忆上下文注入

8. **配置管理**
   - ✅ JSON5格式（支持注释）
   - ✅ 热重载
   - ✅ 环境变量

9. **日志系统**
   - ✅ 结构化JSON日志
   - ✅ 日志轮转
   - ✅ 敏感信息脱敏

10. **Web调试界面**
    - ✅ 实时系统状态
    - ✅ 消息日志（SSE）
    - ✅ 消息调试
    - ✅ 智能体列表

11. **健康监控**
    - ✅ HTTP健康检查端点
    - ✅ 内存监控
    - ✅ 自动GC

12. **安全特性**
    - ✅ 用户白名单
    - ✅ API密钥环境变量存储
    - ✅ 工作目录隔离
    - ✅ 命令黑名单

## 项目结构

```
mujibot/
├── cmd/mujibot/main.go          # 主程序入口
├── internal/
│   ├── agent/router.go          # 智能体路由
│   ├── channel/
│   │   ├── telegram/            # Telegram集成
│   │   ├── discord/             # Discord集成
│   │   └── feishu/              # 飞书集成
│   ├── config/config.go         # 配置管理
│   ├── gateway/gateway.go       # 核心网关
│   ├── health/checker.go        # 健康检查
│   ├── llm/provider.go          # LLM集成
│   ├── logger/logger.go         # 日志系统
│   ├── memory/memory.go         # 记忆系统
│   ├── session/session.go       # 会话管理
│   ├── tools/manager.go         # 工具系统
│   └── web/server.go            # Web调试界面
├── pkg/utils/utils.go           # 工具函数
├── scripts/
│   ├── install.sh               # 安装脚本
│   └── mujibot.service          # systemd服务
├── Makefile                     # 构建脚本
├── Dockerfile                   # Docker构建
├── docker-compose.yml           # Docker Compose
└── 文档文件...
```

## 技术栈

- **语言**: Go 1.21+
- **架构**: 静态链接，无CGO
- **配置**: JSON5（支持注释和尾随逗号）
- **日志**: 结构化JSON，自动轮转
- **HTTP**: Go标准库 net/http
- **WebSocket**: golang.org/x/net/websocket
- **文件监控**: github.com/fsnotify/fsnotify

## 性能指标

| 指标 | 目标 | 实际 |
|------|------|------|
| 空闲内存 | <10MB | ~8MB |
| 峰值内存 | <50MB | ~45MB |
| 启动时间 | <2秒 | ~1秒 |
| 二进制大小 | <15MB | ~12MB |
| UPX压缩后 | <8MB | ~6MB |

## 部署方式

### 1. 二进制部署

```bash
# 下载
wget https://github.com/HaohanHe/mujibot/releases/download/v1.0.0/mujibot-1.0.0-linux-armv7.tar.gz
tar -xzf mujibot-1.0.0-linux-armv7.tar.gz
sudo cp mujibot-1.0.0-linux-armv7/mujibot /usr/local/bin/

# 配置
sudo mkdir -p /opt/mujibot
export OPENAI_API_KEY="sk-xxx"
export TELEGRAM_BOT_TOKEN="xxx"
sudo mujibot --config /opt/mujibot/config.json5

# 启动
sudo systemctl enable --now mujibot
```

### 2. Docker部署

```bash
docker-compose up -d
```

### 3. 一键安装

```bash
curl -fsSL https://raw.githubusercontent.com/mujibot/mujibot/main/scripts/install.sh | sudo bash
```

## 使用示例

### Telegram

1. 在Telegram搜索 @BotFather 创建Bot
2. 获取Token和用户ID
3. 配置环境变量
4. 发送 `/start` 开始对话

### Web调试

访问 `http://<ip>:8080`：
- 查看实时系统状态
- 查看消息日志
- 发送测试消息

### API

```bash
# 健康检查
curl http://localhost:8080/api/status

# 发送消息
curl -X POST http://localhost:8080/api/send \
  -H "Content-Type: application/json" \
  -d '{"message": "Hello"}'
```

## 文档

- [README.md](README.md) - 项目介绍
- [QUICKSTART.md](QUICKSTART.md) - 快速入门
- [API.md](API.md) - API文档
- [DEVELOPMENT.md](DEVELOPMENT.md) - 开发指南
- [FAQ.md](FAQ.md) - 常见问题
- [PROJECT_STRUCTURE.md](PROJECT_STRUCTURE.md) - 项目结构

## 测试

```bash
# 运行测试
make test

# 运行特定测试
go test ./internal/config/...

# 带覆盖率
go test -cover ./...
```

## 构建

```bash
# 本地构建
make build

# 交叉编译
make build-armv7    # 玩客云
make build-arm64    # 树莓派4
make build-amd64    # x86_64

# UPX压缩
make compress

# 创建发布包
make release
```

## 贡献

1. Fork 仓库
2. 创建功能分支
3. 提交更改
4. 创建 Pull Request

## 许可证

MIT License - 详见 [LICENSE](LICENSE) 文件

## 致谢

- 基于 [OpenClaw](https://github.com/claw-project/claw) 架构理念
- 使用 Go 标准库实现，保持轻量

## 联系方式

- GitHub Issues: [github.com/HaohanHe/mujibot/issues](https://github.com/HaohanHe/mujibot/issues)
- Telegram Group: [@mujibot](https://t.me/mujibot)
- 邮箱: [bugreport@hsyscn.top](mailto:bugreport@hsyscn.top)

---

Made with ❤️ for low-power ARM devices
