# Mujibot API 文档

## 基础信息

- **Base URL**: `http://localhost:8080`
- **Content-Type**: `application/json`

## 状态端点

### GET /api/status

获取系统状态信息。

**响应示例**:

```json
{
  "status": "ok",
  "timestamp": 1704067200,
  "memory": {
    "alloc": 8388608,
    "heap_alloc": 6291456,
    "heap_sys": 16777216,
    "sys": 25165824
  },
  "goroutines": 15,
  "sessions": {
    "total_sessions": 5,
    "max_sessions": 100,
    "max_messages": 20
  }
}
```

### GET /api/logs

获取最近的调试日志。

**响应示例**:

```json
[
  {
    "time": "14:30:25",
    "type": "user",
    "source": "telegram",
    "content": "Hello",
    "user_id": "123456789",
    "channel": "telegram"
  },
  {
    "time": "14:30:26",
    "type": "assistant",
    "source": "telegram",
    "content": "Hello! How can I help you?",
    "user_id": "123456789",
    "channel": "telegram"
  }
]
```

### GET /api/sessions

获取会话统计信息。

**响应示例**:

```json
{
  "total_sessions": 5,
  "max_sessions": 100,
  "max_messages": 20,
  "idle_timeout": 3600
}
```

### GET /api/agents

获取智能体列表。

**响应示例**:

```json
[
  {
    "id": "default",
    "name": "Mujibot",
    "model": "gpt-4o-mini"
  }
]
```

### GET /api/config

获取配置信息（隐藏敏感信息）。

**响应示例**:

```json
{
  "server": {
    "port": 8080,
    "healthCheck": true
  },
  "channels": {
    "telegram": {
      "enabled": true
    },
    "discord": {
      "enabled": false
    },
    "feishu": {
      "enabled": false
    }
  },
  "llm": {
    "provider": "openai",
    "model": "gpt-4o-mini",
    "baseURL": ""
  },
  "agents": 1
}
```

## 消息端点

### POST /api/send

发送测试消息。

**请求体**:

```json
{
  "message": "Hello, Mujibot!",
  "agent_id": "default"
}
```

**响应示例**:

```json
{
  "response": "Hello! How can I help you today?"
}
```

**错误响应**:

```json
{
  "error": "agent not found: invalid_id"
}
```

### GET /api/messages/stream

消息流（Server-Sent Events）。

**示例**:

```javascript
const eventSource = new EventSource('/api/messages/stream');

eventSource.onmessage = (event) => {
  const msg = JSON.parse(event.data);
  console.log(msg);
};
```

**事件数据**:

```json
{
  "time": "14:30:25",
  "type": "user",
  "source": "web",
  "content": "Hello",
  "user_id": "web_user",
  "channel": "web"
}
```

## Webhook 端点

### POST /webhook/feishu

飞书事件Webhook。

**说明**: 用于接收飞书开放平台的事件推送。

**配置步骤**:

1. 在飞书开放平台创建应用
2. 启用机器人功能
3. 配置事件订阅URL: `http://<your-server>:8080/webhook/feishu`
4. 订阅 `im.message.receive_v1` 事件

## 健康检查

### GET /health

简单的健康检查。

**响应**:

```json
{
  "status": "healthy"
}
```

## 错误处理

所有API错误都会返回适当的HTTP状态码和错误信息：

| 状态码 | 说明 |
|--------|------|
| 200 | 成功 |
| 400 | 请求参数错误 |
| 404 | 资源不存在 |
| 405 | 方法不允许 |
| 500 | 服务器内部错误 |

**错误响应格式**:

```json
{
  "error": "error message"
}
```

## 限制

- 消息长度限制：4096字符
- 请求频率限制：无（建议自行控制）
- 并发连接限制：无

## 示例代码

### cURL

```bash
# 获取状态
curl http://localhost:8080/api/status

# 发送消息
curl -X POST http://localhost:8080/api/send \
  -H "Content-Type: application/json" \
  -d '{"message": "Hello"}'

# 获取日志
curl http://localhost:8080/api/logs
```

### Python

```python
import requests

# 获取状态
response = requests.get('http://localhost:8080/api/status')
print(response.json())

# 发送消息
response = requests.post(
    'http://localhost:8080/api/send',
    json={'message': 'Hello'}
)
print(response.json()['response'])
```

### JavaScript

```javascript
// 获取状态
fetch('http://localhost:8080/api/status')
  .then(res => res.json())
  .then(data => console.log(data));

// 发送消息
fetch('http://localhost:8080/api/send', {
  method: 'POST',
  headers: {'Content-Type': 'application/json'},
  body: JSON.stringify({message: 'Hello'})
})
  .then(res => res.json())
  .then(data => console.log(data.response));
```
