# Mujibot 功能增强计划

## 一、动态运行环境

### 问题
当前硬编码 "低功耗ARM设备"，不准确。

### 解决方案
动态获取系统信息：

```go
// 获取运行环境信息
func getSystemInfo() string {
    // 系统类型: Linux/Windows/Darwin
    // 架构: amd64/arm64/arm
    // 内核版本
    // 主机名
}
```

### 系统提示词示例
```
- 当前时间: 2026-02-15 15:30:00
- 系统类型: Linux (Armbian)
- 系统架构: armv7l (ARM Cortex-A5)
- 内核版本: 6.1.63-current-sunxi
- 主机名: mujibot-server
```

---

## 二、LLM 提供商扩展

### 当前支持
- OpenAI
- Anthropic
- Ollama

### 需要新增（OpenAI 兼容 API）

| 提供商 | baseURL | 模型 | 特点 |
|--------|---------|------|------|
| DeepSeek | api.deepseek.com | deepseek-chat, deepseek-reasoner | 国产，性价比高 |
| MiniMax | api.minimax.chat | abab6.5s-chat | 国产，多模态 |
| 小米 MiMo | api.mimo.ai | MiMo-V2-Flash | 国产，Agent友好 |
| Moonshot/Kimi | api.moonshot.cn | moonshot-v1-8k, kimi-k2 | 长文本 |
| 智谱 GLM | open.bigmodel.cn | glm-4, glm-4-flash | 国产 |
| 通义千问 | dashscope.aliyuncs.com | qwen-turbo, qwen-plus | 阿里云 |
| 豆包 | ark.cn-beijing.volces.com | doubao-pro | 字节跳动 |
| Groq | api.groq.com | llama-3.1-70b, mixtral | 极速推理 |
| SiliconFlow | api.siliconflow.cn | 多模型代理 | 国内代理 |

### 实现方案
配置文件预设：

```json
{
  "llm": {
    "provider": "deepseek",
    "model": "deepseek-chat",
    "apiKey": "${DEEPSEEK_API_KEY}",
    "baseURL": "https://api.deepseek.com"
  },
  "llmPresets": {
    "deepseek": {
      "baseURL": "https://api.deepseek.com",
      "models": ["deepseek-chat", "deepseek-reasoner"]
    },
    "minimax": {
      "baseURL": "https://api.minimax.chat/v1",
      "models": ["abab6.5s-chat", "abab6.5g-chat"]
    },
    "mimo": {
      "baseURL": "https://api.mimo.ai/v1",
      "models": ["MiMo-V2-Flash"]
    },
    "moonshot": {
      "baseURL": "https://api.moonshot.cn/v1",
      "models": ["moonshot-v1-8k", "moonshot-v1-32k", "kimi-k2-0711-preview"]
    },
    "zhipu": {
      "baseURL": "https://open.bigmodel.cn/api/paas/v4",
      "models": ["glm-4", "glm-4-flash", "glm-4-plus"]
    },
    "qwen": {
      "baseURL": "https://dashscope.aliyuncs.com/compatible-mode/v1",
      "models": ["qwen-turbo", "qwen-plus", "qwen-max"]
    },
    "groq": {
      "baseURL": "https://api.groq.com/openai/v1",
      "models": ["llama-3.1-70b-versatile", "mixtral-8x7b-32768"]
    }
  }
}
```

---

## 三、记忆系统增强（海马体模式）

### 当前问题
1. 无自动上下文压缩
2. 用户说"记住这个"无法智能识别
3. 关键词检测不可靠

### 解决方案

#### 3.1 自动上下文压缩
当会话消息超过阈值时，自动压缩旧消息：

```go
func (s *Session) compressHistory() {
    if len(s.Messages) > MaxMessages {
        // 提取关键信息
        summary := llm.Summarize(s.Messages[:len(s.Messages)-KeepRecent])
        // 替换为摘要
        s.Messages = append([]Message{{Role: "system", Content: summary}}, 
                             s.Messages[len(s.Messages)-KeepRecent:]...)
    }
}
```

#### 3.2 海马体记忆系统
用户重要信息自动存储：

```go
type Hippocampus struct {
    LongTermMemory map[string]MemoryItem  // 按主题索引
    RecentFacts    []MemoryItem           // 最近事实
    UserPreferences map[string]string     // 用户偏好
}

type MemoryItem struct {
    ID          string    `json:"id"`
    Category    string    `json:"category"`     // preference, fact, event, contact
    Content     string    `json:"content"`      // 原始内容
    Keywords    []string  `json:"keywords"`     // 关键词
    Importance  int       `json:"importance"`   // 重要性 1-10
    CreatedAt   time.Time `json:"createdAt"`
    LastAccessed time.Time `json:"lastAccessed"`
    AccessCount int       `json:"accessCount"`  // 访问次数
}
```

#### 3.3 智能记忆识别
LLM 自动判断是否需要记忆：

系统提示词添加：
```
## 记忆规则

当用户表达以下意图时，自动调用 memory_write 工具：
1. "记住..." / "别忘了..." / "记下来..."
2. "我喜欢..." / "我讨厌..." / "我的..."
3. 重要日期、联系方式、地址等
4. 用户反复提及的信息

记忆分类：
- preference: 用户偏好
- fact: 事实信息
- event: 事件/日期
- contact: 联系人信息
```

---

## 四、多语言支持

### 语言配置

```json
{
  "language": {
    "default": "en",
    "supported": ["zh-CN", "en-US", "ja-JP"],
    "current": "zh-CN"
  }
}
```

### 系统提示词多语言

```go
var SystemPrompts = map[string]string{
    "zh-CN": "你是一个运行在 %s 上的AI助手...",
    "en-US": "You are an AI assistant running on %s...",
    "ja-JP": "あなたは %s で動作するAIアシスタントです...",
}
```

### 部署时语言选择

首次启动时检测：
1. 检查配置文件中的语言设置
2. 如果没有，进入交互式选择
3. 类似手机激活流程

```
┌─────────────────────────────────────┐
│                                     │
│           Hello / 你好 / こんにちは   │
│                                     │
│     Please select your language:    │
│                                     │
│     1. English (US)                 │
│     2. 简体中文                      │
│     3. 日本語                        │
│                                     │
│     Enter [1-3]: _                  │
│                                     │
└─────────────────────────────────────┘
```

### 用户语言检测

系统提示词添加：
```
## 用户语言

当前用户使用语言: {detected_language}

请使用与用户相同的语言回复。
```

---

## 五、Web 界面工具管理

### 工具管理页面

```
┌─────────────────────────────────────────────────────┐
│  工具管理 / Tool Management                          │
├─────────────────────────────────────────────────────┤
│                                                     │
│  内置工具 / Built-in Tools                          │
│  ─────────────────────────────────────────────────  │
│  [✓] read_file        读取文件                      │
│  [✓] write_file       写入文件                      │
│  [ ] execute_command  执行命令 (已禁用)              │
│  [✓] web_search       网页搜索                      │
│  [✓] weather          天气查询                      │
│  [✓] ip_info          IP信息                        │
│  [✓] exchange_rate    汇率查询                      │
│                                                     │
│  自定义 API / Custom APIs                           │
│  ─────────────────────────────────────────────────  │
│  [+ 添加新API]                                      │
│                                                     │
│  ┌─────────────────────────────────────────────┐   │
│  │ 名称: my_weather_api                         │   │
│  │ URL: https://api.weather.com/v1?city={city} │   │
│  │ 方法: [GET ▼]  超时: [10] 秒                  │   │
│  │ API Key: [••••••••••••] [测试]               │   │
│  │ [启用] [删除] [保存]                          │   │
│  └─────────────────────────────────────────────┘   │
│                                                     │
└─────────────────────────────────────────────────────┘
```

### API 端点

```
GET  /api/tools              # 获取所有工具列表
POST /api/tools/toggle       # 切换工具开关
POST /api/tools/custom       # 添加自定义API
PUT  /api/tools/custom/:id   # 更新自定义API
DELETE /api/tools/custom/:id # 删除自定义API
POST /api/tools/custom/:id/test # 测试自定义API
```

---

## 六、实现优先级

| 优先级 | 任务 | 预计工作量 |
|--------|------|------------|
| P0 | 动态运行环境 | 小 |
| P0 | LLM 提供商扩展 | 中 |
| P1 | 海马体记忆系统 | 大 |
| P1 | 多语言支持 | 中 |
| P2 | Web 工具管理界面 | 中 |
| P2 | 自动上下文压缩 | 中 |

---

## 七、文件修改清单

### 需要修改的文件
1. `internal/config/config.go` - 添加语言配置、LLM预设
2. `internal/llm/provider.go` - 支持更多提供商
3. `internal/agent/router.go` - 动态系统提示词
4. `internal/memory/manager.go` - 海马体记忆
5. `internal/web/server.go` - 工具管理API
6. `cmd/mujibot/main.go` - 首次启动语言选择

### 需要新增的文件
1. `internal/system/info.go` - 系统信息获取
2. `internal/memory/hippocampus.go` - 海马体记忆
3. `internal/i18n/i18n.go` - 国际化支持
4. `internal/web/tools.go` - 工具管理处理器
