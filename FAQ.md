# Mujibot 常见问题

## 安装问题

### Q: 玩客云如何安装？

A: 玩客云使用ARMv7架构，下载对应版本：

```bash
wget https://github.com/HaohanHe/mujibot/releases/download/v1.0.0/mujibot-1.0.0-linux-armv7.tar.gz
tar -xzf mujibot-1.0.0-linux-armv7.tar.gz
sudo cp mujibot-1.0.0-linux-armv7/mujibot /usr/local/bin/
sudo chmod +x /usr/local/bin/mujibot
```

### Q: 需要安装Go吗？

A: 不需要。我们提供预编译的二进制文件，无需任何运行时依赖。

### Q: 支持哪些操作系统？

A: 支持Linux（ARMv7、ARM64、x86_64），包括：
- Armbian
- Ubuntu
- Debian
- Alpine

## 配置问题

### Q: 如何获取Telegram Bot Token？

A:
1. 在Telegram搜索 @BotFather
2. 发送 `/newbot`
3. 按提示设置名称和用户名
4. 保存获得的Token

### Q: 如何获取Telegram用户ID？

A:
1. 在Telegram搜索 @userinfobot
2. 发送任意消息
3. 机器人会回复你的用户ID

### Q: 如何配置飞书？

A:
1. 访问 [飞书开放平台](https://open.feishu.cn/)
2. 创建企业自建应用
3. 启用机器人功能
4. 获取 App ID 和 App Secret
5. 配置事件订阅URL: `http://<your-server>:8080/webhook/feishu`
6. 订阅 `im.message.receive_v1` 事件

### Q: 可以使用国内LLM吗？

A: 可以，配置 `baseURL` 即可：

```json5
llm: {
  provider: "openai",
  model: "deepseek-chat",
  apiKey: "${DEEPSEEK_API_KEY}",
  baseURL: "https://api.deepseek.com/v1",
}
```

支持的兼容API：
- DeepSeek
- Moonshot (Kimi)
- 其他OpenAI兼容API

### Q: 如何配置Ollama本地模型？

A:

```json5
llm: {
  provider: "ollama",
  model: "llama2",
  apiKey: "",  // 不需要
  baseURL: "http://localhost:11434",
}
```

确保Ollama已安装并运行：
```bash
ollama run llama2
```

## 运行问题

### Q: 服务无法启动？

A: 检查以下步骤：

1. 检查配置文件语法
```bash
sudo mujibot --config /opt/mujibot/config.json5
# 查看错误输出
```

2. 检查日志
```bash
sudo journalctl -u mujibot -n 50
```

3. 检查端口占用
```bash
sudo lsof -i :8080
```

4. 检查权限
```bash
ls -la /opt/mujibot/
ls -la /var/log/mujibot/
```

### Q: 内存使用过高？

A:

1. 检查当前内存使用
```bash
curl http://localhost:8080/api/status | jq .memory
```

2. 调整会话配置
```json5
session: {
  maxMessages: 10,      // 减少保留消息数
  idleTimeout: 1800,    // 减少空闲超时
  maxSessions: 50,      // 减少最大会话数
}
```

3. 手动触发GC
```bash
sudo systemctl kill -s USR1 mujibot
```

### Q: Telegram Bot不响应？

A:

1. 检查Token是否正确
2. 确认已向Bot发送 `/start`
3. 检查用户ID是否在白名单中
4. 查看日志
```bash
sudo journalctl -u mujibot -f
```

### Q: 飞书Webhook配置失败？

A:

1. 确保服务器可被公网访问
2. 检查端口8080是否开放
3. 验证加密密钥配置（如果使用加密）
4. 查看日志中的错误信息

## 功能问题

### Q: 如何添加自定义工具？

A: 编辑 `internal/tools/manager.go`，实现Tool接口：

```go
type MyTool struct{}

func (t *MyTool) Name() string { return "my_tool" }
func (t *MyTool) Description() string { return "描述" }
func (t *MyTool) Parameters() map[string]interface{} { return params }
func (t *MyTool) Execute(args map[string]interface{}) (string, error) {
    // 实现逻辑
    return "result", nil
}
```

然后在 `registerBuiltinTools` 中注册。

### Q: 如何创建多个智能体？

A: 在配置中添加：

```json5
agents: {
  default: {
    name: "Mujibot",
    systemPrompt: "通用助手",
  },
  coder: {
    name: "Code Assistant",
    systemPrompt: "编程专家",
    tools: ["read_file", "write_file"],
  },
}
```

### Q: 如何限制用户访问？

A: 在渠道配置中设置白名单：

```json5
channels: {
  telegram: {
    enabled: true,
    token: "${TELEGRAM_BOT_TOKEN}",
    allowedUsers: [123456789, 987654321],
  },
}
```

### Q: 支持语音消息吗？

A: 当前版本不支持。这是为了控制资源占用，后续版本可能添加。

### Q: 支持图片处理吗？

A: 当前版本不支持。这是为了控制资源占用，建议使用其他方案。

## 性能问题

### Q: 如何优化内存使用？

A:

1. 使用更小的LLM模型（如gpt-4o-mini）
2. 减少会话保留消息数
3. 缩短会话超时时间
4. 使用本地Ollama模型

### Q: 如何查看性能指标？

A:

```bash
# 系统状态
curl http://localhost:8080/api/status

# 会话统计
curl http://localhost:8080/api/sessions

# 实时日志
curl http://localhost:8080/api/messages/stream
```

### Q: 二进制文件太大？

A: 使用UPX压缩：

```bash
# 安装UPX
sudo apt-get install upx

# 压缩
upx --best /usr/local/bin/mujibot
```

压缩后通常可以从15MB减少到8MB以下。

## 安全问题

### Q: 如何保护API密钥？

A:

1. 使用环境变量（推荐）
```bash
export OPENAI_API_KEY="sk-xxx"
```

2. 配置文件权限
```bash
chmod 600 /opt/mujibot/config.json5
```

3. 使用专用用户运行
```bash
useradd --system mujibot
```

### Q: 工具系统安全吗？

A: 工具系统有多层安全保护：

1. **工作目录限制**: 只能访问配置的工作目录
2. **命令黑名单**: 禁止危险命令（reboot、shutdown等）
3. **危险命令确认**: rm -rf等命令需要确认
4. **超时限制**: 命令执行超时30秒

### Q: 如何禁用危险工具？

A: 在智能体配置中指定允许的工具：

```json5
agents: {
  default: {
    tools: ["read_file", "list_directory"],  // 只允许安全工具
  },
}
```

## 调试问题

### Q: 如何启用调试日志？

A: 修改配置：

```json5
logging: {
  level: "debug",
  file: "/var/log/mujibot/debug.log",
}
```

然后重启服务。

### Q: Web调试界面无法访问？

A:

1. 检查服务是否运行
```bash
sudo systemctl status mujibot
```

2. 检查端口是否监听
```bash
sudo netstat -tlnp | grep 8080
```

3. 检查防火墙
```bash
sudo ufw status
sudo ufw allow 8080/tcp
```

### Q: 如何查看实时消息流？

A: 使用浏览器访问 `http://<ip>:8080` 或命令行：

```bash
curl http://localhost:8080/api/messages/stream
```

## 更新问题

### Q: 如何更新到新版本？

A:

```bash
# 停止服务
sudo systemctl stop mujibot

# 备份配置
sudo cp /opt/mujibot/config.json5 /opt/mujibot/config.json5.bak

# 下载新版本
wget https://github.com/HaohanHe/mujibot/releases/download/v1.0.1/mujibot-1.0.1-linux-armv7.tar.gz
tar -xzf mujibot-1.0.1-linux-armv7.tar.gz
sudo cp mujibot-1.0.1-linux-armv7/mujibot /usr/local/bin/

# 启动服务
sudo systemctl start mujibot
```

### Q: 如何回滚？

A:

```bash
# 停止服务
sudo systemctl stop mujibot

# 恢复旧版本
sudo cp /path/to/backup/mujibot /usr/local/bin/
sudo cp /opt/mujibot/config.json5.bak /opt/mujibot/config.json5

# 启动服务
sudo systemctl start mujibot
```

## 其他问题

### Q: 如何贡献代码？

A: 参见 [DEVELOPMENT.md](DEVELOPMENT.md)

### Q: 如何报告Bug？

A: 在GitHub Issues中提交，包含：
- 问题描述
- 复现步骤
- 环境信息（OS、架构、版本）
- 相关日志

### Q: 有商业支持吗？

A: 目前仅社区支持。如有商业需求请联系维护者。

### Q: 未来路线图？

A: 计划中的功能：
- WhatsApp支持
- 更多LLM提供商
- 插件系统
- 集群支持

---

还有其他问题？请在GitHub Issues中提问！
