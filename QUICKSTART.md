# Mujibot 快速入门

## 1. 安装

### 玩客云/ARM设备

```bash
# 一键安装
curl -fsSL https://raw.githubusercontent.com/mujibot/mujibot/main/scripts/install.sh | sudo bash

# 或手动下载
wget https://github.com/HaohanHe/mujibot/releases/download/v1.0.0/mujibot-1.0.0-linux-armv7.tar.gz
tar -xzf mujibot-1.0.0-linux-armv7.tar.gz
sudo cp mujibot-1.0.0-linux-armv7/mujibot /usr/local/bin/
```

### Docker

```bash
docker run -d \
  --name mujibot \
  -p 8080:8080 \
  -e OPENAI_API_KEY=sk-xxx \
  -e TELEGRAM_BOT_TOKEN=xxx \
  -v $(pwd)/config.json5:/config.json5 \
  mujibot:latest
```

## 2. 配置

### 最小配置

```bash
# 创建配置目录
sudo mkdir -p /opt/mujibot

# 设置环境变量
export OPENAI_API_KEY="sk-your-key"
export TELEGRAM_BOT_TOKEN="your-bot-token"

# 生成默认配置
sudo mujibot --config /opt/mujibot/config.json5
```

### 编辑配置

```bash
sudo nano /opt/mujibot/config.json5
```

关键配置项：

```json
{
  "llm": {
    "provider": "openai",
    "model": "gpt-4o-mini",
    "apiKey": "${OPENAI_API_KEY}"
  },
  "channels": {
    "telegram": {
      "enabled": true,
      "token": "${TELEGRAM_BOT_TOKEN}",
      "allowedUsers": [123456789]
    }
  }
}
```

## 3. 启动

### Systemd

```bash
# 安装服务
sudo cp scripts/mujibot.service /etc/systemd/system/
sudo systemctl daemon-reload

# 启动
sudo systemctl enable --now mujibot

# 查看状态
sudo systemctl status mujibot
```

### 手动启动

```bash
sudo mujibot --config /opt/mujibot/config.json5
```

## 4. 使用

### Telegram

1. 找到你的Bot（使用@BotFather创建）
2. 发送 `/start`
3. 开始对话！

### Web调试界面

访问 `http://<你的IP>:8080`

功能：
- 实时系统状态
- 消息日志
- 直接发送测试消息

### API

```bash
# 健康检查
curl http://localhost:8080/api/status

# 发送测试消息
curl -X POST http://localhost:8080/api/send \
  -H "Content-Type: application/json" \
  -d '{"message": "Hello"}'
```

## 5. 监控

### 查看日志

```bash
# 实时日志
sudo journalctl -u mujibot -f

# 最近100行
sudo journalctl -u mujibot -n 100
```

### 系统状态

```bash
curl http://localhost:8080/api/status | jq
```

## 6. 故障排除

### 无法启动

```bash
# 检查配置
sudo mujibot --config /opt/mujibot/config.json5 --help

# 查看日志
sudo journalctl -u mujibot -n 50 --no-pager
```

### 内存过高

```bash
# 查看内存使用
curl http://localhost:8080/api/status | jq .memory

# 重启服务
sudo systemctl restart mujibot
```

### Telegram无响应

1. 检查Bot Token是否正确
2. 确认已向Bot发送 `/start`
3. 检查用户ID是否在白名单中

## 7. 更新

```bash
# 停止服务
sudo systemctl stop mujibot

# 下载新版本
wget https://github.com/HaohanHe/mujibot/releases/download/v1.0.1/mujibot-1.0.1-linux-armv7.tar.gz
tar -xzf mujibot-1.0.1-linux-armv7.tar.gz
sudo cp mujibot-1.0.1-linux-armv7/mujibot /usr/local/bin/

# 启动服务
sudo systemctl start mujibot
```

## 8. 卸载

```bash
sudo systemctl stop mujibot
sudo systemctl disable mujibot
sudo rm -f /usr/local/bin/mujibot
sudo rm -f /etc/systemd/system/mujibot.service
sudo rm -rf /opt/mujibot
```

## 常用命令速查

| 命令 | 说明 |
|------|------|
| `sudo systemctl start mujibot` | 启动服务 |
| `sudo systemctl stop mujibot` | 停止服务 |
| `sudo systemctl restart mujibot` | 重启服务 |
| `sudo systemctl status mujibot` | 查看状态 |
| `sudo journalctl -u mujibot -f` | 查看日志 |
| `curl http://localhost:8080/api/status` | 健康检查 |
