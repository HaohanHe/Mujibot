# 需求文档

## 简介

Mujibot引擎是一个资源高效的个人AI助手系统，专为低配置ARM设备（如玩客云，仅1GB RAM和8GB eMMC存储）设计。该系统基于OpenClaw架构理念，使用Go语言重新实现，提供单二进制部署、极低资源占用（<10MB RAM空闲）和快速启动的特性。系统通过Gateway网关管理多渠道消息路由、LLM集成和工具执行，同时保持最小的资源占用。

**与OpenClaw的主要区别：**

| 特性 | OpenClaw | Mujibot轻量引擎 |
|------|----------|-----------------|
| 运行时 | Node.js >= 22 | 无依赖（静态链接） |
| 内存占用 | 较高（Node.js运行时） | <10MB空闲，<50MB峰值 |
| 二进制大小 | 需Node环境 | <15MB单文件 |
| 浏览器控制 | 支持（Playwright） | 不支持（资源限制） |
| Docker沙箱 | 支持 | 不支持（资源限制） |
| Web UI | 支持 | 不支持（仅消息渠道） |
| 图像处理 | 支持 | 不支持（资源限制） |

## 术语表

- **Gateway网关**: 核心进程，管理所有消息渠道连接、路由和生命周期
- **轻量级Claw引擎**: 基于OpenClaw架构理念的Go语言实现，针对低配置设备优化
- **消息渠道(Channel)**: 用于用户交互的外部消息服务（Telegram、Discord、WhatsApp）
- **智能体(Agent)**: 处理用户请求并与LLM交互的AI代理实例
- **智能体路由器**: 管理多个隔离AI智能体实例的路由组件
- **LLM提供商**: 外部大语言模型API服务（OpenAI、Anthropic、Ollama）
- **工具(Tool)**: 执行文件操作、系统命令等任务的可扩展工具集
- **技能(Skill)**: 可扩展功能模块，提供特定领域能力
- **会话管理器**: 按发送者、群组维护对话上下文的组件
- **配置管理器**: 管理JSON5配置文件和热重载的组件
- **ARM设备**: 低配置ARM架构设备（例如玩客云：晶晨S805四核1.5GHz，1GB DDR3，8GB eMMC）
- **单二进制文件**: 包含整个应用程序的静态链接Go可执行文件

## 目标硬件规格

### 玩客云（主要目标平台）

| 组件 | 规格 |
|------|------|
| CPU | 晶晨S805，四核ARM Cortex-A5 @ 1.5GHz（armv7架构） |
| 内存 | 1GB DDR3 |
| 存储 | 8GB eMMC |
| 网络 | 千兆以太网 |
| 接口 | HDMI 1.4, USB 2.0 x2, microSD卡槽 |
| 功耗 | 约5W |
| 操作系统 | Armbian Linux |

### 其他支持平台

- Raspberry Pi Zero 2W / 3B / 4B（armv7/arm64）
- x86_64 Linux服务器（Ubuntu、Debian、Alpine）
- 其他Armbian支持的ARM设备

## 需求

### 需求1：极致资源效率

**用户故事：** 作为低配置ARM设备的用户，我希望系统消耗极少的资源，以便它能在仅有1GB RAM和8GB存储的玩客云设备上流畅运行。

#### 验收标准

1. WHEN 轻量级Claw引擎启动时，THE 系统 SHALL 在空闲状态下消耗少于10MB的RAM
2. WHEN 轻量级Claw引擎处理单个消息时，THE 系统 SHALL 在峰值使用时消耗少于50MB的RAM
3. THE 单二进制文件 SHALL 占用少于15MB的磁盘存储空间（使用UPX压缩后<8MB）
4. WHEN 轻量级Claw引擎启动时，THE 系统 SHALL 在1GB RAM的ARM设备上2秒内完成初始化
5. WHEN 轻量级Claw引擎连续运行7天时，THE 系统 SHALL 保持内存使用低于50MB且无内存泄漏
6. THE 系统 SHALL 使用Go语言实现以确保高效的内存管理和垃圾回收
7. THE 系统 SHALL 使用Go的`sync.Pool`复用对象以减少GC压力
8. THE 系统 SHALL 避免使用cgo以保持静态链接和最小二进制大小

### 需求2：跨平台兼容性

**用户故事：** 作为运行不同Linux发行版和架构的用户，我希望系统能跨平台工作，以便我可以在各种设备上部署它而无需修改。

#### 验收标准

1. THE 轻量级Claw引擎 SHALL 支持ARM架构（armv7、arm64）
2. THE 轻量级Claw引擎 SHALL 支持x86_64架构
3. THE 轻量级Claw引擎 SHALL 在Armbian Linux发行版上运行
4. THE 轻量级Claw引擎 SHALL 在主流Linux发行版（Ubuntu、Debian、Alpine）上运行
5. THE 单二进制文件 SHALL 静态链接，无需外部运行时依赖（无需Node.js、Python、glibc）
6. THE 系统 SHALL 使用Go语言的交叉编译功能支持多架构构建
7. THE 系统 SHALL 使用`CGO_ENABLED=0`进行纯静态编译
8. THE 系统 SHALL 在Alpine Linux（musl libc）上无需额外依赖运行

### 需求3：Gateway网关和多渠道支持

**用户故事：** 作为用户，我希望通过多个消息渠道控制AI助手，以便我可以使用熟悉的界面从任何地方与它交互。

#### 验收标准

1. THE Gateway网关 SHALL 作为核心进程管理所有消息渠道连接
2. THE Gateway网关 SHALL 支持Telegram Bot API作为主要消息渠道
3. THE Gateway网关 SHALL 支持Discord Bot API作为次要消息渠道
4. THE Gateway网关 SHALL 支持WebSocket接口用于Web客户端扩展
5. WHEN 用户通过已配置渠道发送消息时，THE Gateway网关 SHALL 在2秒内接收并路由消息
6. WHERE 配置了多个消息渠道时，THE Gateway网关 SHALL 支持同时连接到所有已配置的渠道
7. WHEN 消息渠道连接失败时，THE Gateway网关 SHALL 使用指数退避重试并继续运行其他渠道
8. THE Gateway网关 SHALL 在处理消息前验证用户身份
9. WHEN Gateway网关完成任务时，THE 系统 SHALL 将响应发送回原始消息渠道
10. THE 系统 SHALL 使用Go原生的`net/http`和`golang.org/x/net/websocket`实现网络通信

### 需求4：多智能体路由

**用户故事：** 作为高级用户，我希望运行多个隔离的AI智能体实例，以便我可以为不同的任务或用户群组配置不同的行为。

#### 验收标准

1. THE 智能体路由器 SHALL 支持配置多个独立的AI智能体实例
2. WHEN 消息到达时，THE 智能体路由器 SHALL 根据发送者、群组或渠道将消息路由到正确的智能体
3. THE 智能体路由器 SHALL 隔离不同智能体的会话上下文
4. WHEN 智能体实例崩溃时，THE 智能体路由器 SHALL 隔离故障并继续运行其他智能体
5. THE 智能体路由器 SHALL 在1GB RAM设备上支持至少3个并发智能体实例
6. THE 系统 SHALL 使用Go的goroutine实现轻量级并发智能体处理
7. THE 系统 SHALL 使用`recover()`防止单个goroutine panic导致进程崩溃

### 需求5：LLM提供商集成

**用户故事：** 作为用户，我希望连接到不同的LLM提供商，以便我可以选择最适合我需求和预算的AI模型。

#### 验收标准

1. THE 轻量级Claw引擎 SHALL 支持OpenAI API集成（GPT-4o、GPT-4o-mini）
2. THE 轻量级Claw引擎 SHALL 支持Anthropic Claude API集成
3. THE 轻量级Claw引擎 SHALL 支持本地Ollama API集成
4. THE 轻量级Claw引擎 SHALL 支持OpenAI兼容API（DeepSeek、Moonshot等国内模型）
5. WHEN LLM API调用失败时，THE 轻量级Claw引擎 SHALL 使用指数退避重试最多3次
6. WHEN LLM API调用超过60秒时，THE 轻量级Claw引擎 SHALL 超时并返回错误消息
7. THE 配置管理器 SHALL 使用环境变量存储API凭证（避免明文存储）
8. THE 系统 SHALL 支持流式响应以减少内存占用和首字延迟
9. THE 系统 SHALL 使用`streaming`模式处理LLM响应以避免缓冲完整响应

### 需求6：工具系统和任务执行

**用户故事：** 作为用户，我希望助手通过可扩展的工具系统执行任务，以便我可以远程自动化文件操作和命令执行。

#### 验收标准

1. THE 工具系统 SHALL 提供标准接口供LLM调用工具
2. WHEN LLM请求读取文件时，THE 工具系统 SHALL 读取文件并返回其内容（限制1MB以内）
3. WHEN LLM请求写入文件时，THE 工具系统 SHALL 将内容写入指定的文件路径
4. WHEN LLM请求执行命令时，THE 工具系统 SHALL 执行shell命令并返回输出
5. WHEN 请求危险命令（rm -rf、dd、mkfs、chmod 777）时，THE 工具系统 SHALL 在执行前要求明确确认
6. WHEN 任务执行失败时，THE 工具系统 SHALL 返回包含失败原因的描述性错误消息
7. THE 工具系统 SHALL 在执行特权操作前强制执行权限检查
8. THE 工具系统 SHALL 支持注册自定义工具而无需修改核心代码
9. THE 工具系统 SHALL 限制文件访问到配置的工作目录（沙箱隔离）
10. THE 工具系统 SHALL 限制命令执行超时时间为30秒

### 需求7：会话管理

**用户故事：** 作为用户，我希望系统维护对话上下文，以便我可以进行多轮自然对话而无需重复背景信息。

#### 验收标准

1. THE 会话管理器 SHALL 按发送者ID维护独立的对话上下文
2. THE 会话管理器 SHALL 按群组ID维护独立的对话上下文
3. WHEN 用户发送消息时，THE 会话管理器 SHALL 将历史消息包含在LLM请求中
4. THE 会话管理器 SHALL 在1小时无活动后自动清理会话以释放内存
5. THE 会话管理器 SHALL 限制每个会话最多保留最近20条消息以控制内存使用
6. WHEN 会话超过内存限制时，THE 会话管理器 SHALL 删除最旧的消息
7. THE 会话管理器 SHALL 使用LRU（最近最少使用）策略管理会话缓存
8. THE 会话管理器 SHALL 支持持久化会话到文件系统（可选）

### 需求8：配置管理和热重载

**用户故事：** 作为系统管理员，我希望通过JSON5配置文件管理系统，以便我可以在不重新编译应用程序的情况下管理设置。

#### 验收标准

1. THE 配置管理器 SHALL 在启动时从JSON5文件加载配置
2. WHEN 配置文件缺失时，THE 配置管理器 SHALL 创建包含示例值的默认配置文件
3. WHEN 配置文件包含无效语法时，THE 配置管理器 SHALL 报告具体的验证错误并拒绝启动
4. THE 配置管理器 SHALL 支持环境变量覆盖敏感凭证
5. WHEN 检测到配置文件更改时，THE 配置管理器 SHALL 在不需要重启的情况下热重载设置
6. THE 配置管理器 SHALL 支持JSON5格式的注释和尾随逗号以提高可读性
7. THE 配置管理器 SHALL 使用`fsnotify`监控配置文件变化

### 需求9：日志和监控

**用户故事：** 作为系统管理员，我希望监控系统活动并排查问题，以便我可以维护可靠的运行。

#### 验收标准

1. THE 轻量级Claw引擎 SHALL 记录所有传入消息及其时间戳和用户标识符
2. THE 轻量级Claw引擎 SHALL 记录所有LLM API调用及其请求/响应元数据
3. THE 轻量级Claw引擎 SHALL 记录所有工具执行及其命令详情和结果
4. WHEN 发生错误时，THE 轻量级Claw引擎 SHALL 记录完整的错误堆栈跟踪
5. THE 轻量级Claw引擎 SHALL 支持可配置的日志级别（DEBUG、INFO、WARN、ERROR）
6. THE 轻量级Claw引擎 SHALL 在日志文件超过5MB时轮转以防止磁盘空间耗尽
7. THE 轻量级Claw引擎 SHALL 使用结构化日志格式（JSON）以便于解析
8. THE 系统 SHALL 使用轻量级日志库（如`zerolog`或`slog`）以减少内存开销

### 需求10：安全和访问控制

**用户故事：** 作为注重安全的用户，我希望系统验证用户身份并控制访问，以便未经授权的用户无法在我的设备上执行命令。

#### 验收标准

1. THE Gateway网关 SHALL 基于消息渠道用户ID验证用户身份
2. THE 配置管理器 SHALL 维护授权用户ID的白名单
3. WHEN 未授权用户发送命令时，THE Gateway网关 SHALL 拒绝请求并记录尝试
4. THE 配置管理器 SHALL 从环境变量读取API凭证（不以明文存储在配置文件中）
5. THE 轻量级Claw引擎 SHALL 不以明文形式记录敏感信息（密码、API密钥）
6. THE 工具系统 SHALL 实施最小权限原则，限制文件访问到配置的工作目录
7. THE 工具系统 SHALL 禁止执行系统关键命令（reboot、shutdown、init）

### 需求11：优雅降级和错误处理

**用户故事：** 作为用户，我希望系统优雅地处理错误，以便临时故障不会导致整个应用程序崩溃。

#### 验收标准

1. WHEN 消息渠道连接失败时，THE Gateway网关 SHALL 使用指数退避尝试重新连接（最大间隔5分钟）
2. WHEN LLM提供商不可用时，THE 轻量级Claw引擎 SHALL 返回用户友好的错误消息
3. WHEN 智能体实例崩溃时，THE 智能体路由器 SHALL 隔离故障并继续运行其他智能体
4. WHEN 磁盘空间严重不足（少于50MB）时，THE 轻量级Claw引擎 SHALL 禁用日志记录并警告用户
5. WHEN 内存使用超过80MB时，THE 轻量级Claw引擎 SHALL 触发垃圾回收并清理旧会话
6. THE 系统 SHALL 使用Go语言的`recover()`机制防止单个goroutine崩溃导致整个进程退出
7. THE 系统 SHALL 实现健康检查端点用于监控进程存活状态

### 需求12：部署和分发

**用户故事：** 作为用户，我希望部署过程简单，以便我可以在没有复杂设置程序的情况下安装和运行系统。

#### 验收标准

1. THE 轻量级Claw引擎 SHALL 作为单个静态链接的Go二进制可执行文件分发
2. THE 单二进制文件 SHALL 包括ARM（armv7、arm64）和x86_64构建版本
3. WHEN 首次执行二进制文件时，THE 系统 SHALL 生成默认JSON5配置文件
4. THE 轻量级Claw引擎 SHALL 提供显示版本信息的`--version`标志
5. THE 轻量级Claw引擎 SHALL 提供显示使用说明的`--help`标志
6. THE 系统 SHALL 支持通过systemd服务文件作为后台守护进程运行
7. THE 系统 SHALL 支持通过`--config`标志指定配置文件路径

### 需求13：消息处理和路由

**用户故事：** 作为用户，我希望系统智能地将我的消息路由到LLM并执行结果操作，以便我可以进行触发自动化的自然对话。

#### 验收标准

1. WHEN 用户发送消息时，THE Gateway网关 SHALL 将消息转发到智能体路由器
2. WHEN 智能体路由器接收消息时，THE 智能体路由器 SHALL 将消息路由到正确的智能体实例
3. WHEN 智能体将消息发送到LLM时，THE 系统 SHALL 包含会话上下文
4. WHEN LLM提供商返回带有工具调用的响应时，THE 工具系统 SHALL 执行请求的工具
5. WHEN 工具执行完成时，THE 系统 SHALL 将结果发送回LLM提供商以生成最终响应
6. WHEN LLM提供商返回最终响应时，THE Gateway网关 SHALL 通过原始消息渠道将其发送给用户
7. THE 系统 SHALL 使用Go语言的goroutine实现并发消息处理
8. THE 系统 SHALL 限制并发消息处理数量以防止内存溢出（最大10个并发）

### 需求14：系统健康监控

**用户故事：** 作为系统管理员，我希望监控系统健康指标，以便我可以在问题导致故障之前检测到它们。

#### 验收标准

1. THE 轻量级Claw引擎 SHALL 在localhost上暴露健康检查HTTP端点（默认端口8080）
2. WHEN 查询健康检查端点`/health`时，THE 系统 SHALL 返回当前内存使用、运行时间和连接状态
3. THE 轻量级Claw引擎 SHALL 跟踪并报告每小时处理的消息数量
4. THE 轻量级Claw引擎 SHALL 跟踪并报告LLM API成功/失败率
5. WHEN 系统健康下降（内存>70MB、连接失败）时，THE 轻量级Claw引擎 SHALL 记录警告
6. THE 健康检查端点 SHALL 返回JSON格式的指标数据
7. THE 系统 SHALL 使用Go的`runtime`包获取内存统计信息

### 需求15：轻量化架构约束

**用户故事：** 作为低配置设备用户，我希望系统排除重量级特性，以便它能在1GB RAM设备上可靠运行。

#### 验收标准

1. THE 轻量级Claw引擎 SHALL NOT 包含Docker沙箱隔离功能
2. THE 轻量级Claw引擎 SHALL NOT 包含浏览器控制功能（Playwright/Puppeteer）
3. THE 轻量级Claw引擎 SHALL NOT 包含图像处理或计算机视觉功能
4. THE 轻量级Claw引擎 SHALL NOT 包含嵌入式数据库（使用文件系统存储）
5. THE 轻量级Claw引擎 SHALL NOT 包含Web UI界面（仅通过消息渠道交互）
6. THE 轻量级Claw引擎 SHALL NOT 包含语音合成/识别功能
7. THE 轻量级Claw引擎 SHALL NOT 包含视频处理功能
8. THE 系统 SHALL 优先使用Go标准库而非第三方依赖以减小二进制大小
9. THE 系统 SHALL 避免使用反射（reflection）以提高性能

### 需求16：记忆系统（可选功能）

**用户故事：** 作为用户，我希望系统能够记住之前的对话和重要信息，以便我可以进行更自然的长期交互。

#### 验收标准

1. THE 系统 SHALL 支持可选的长期记忆存储（基于文件系统）
2. THE 记忆系统 SHALL 将每日笔记存储在`memory/YYYY-MM-DD.md`文件中
3. THE 记忆系统 SHALL 将长期记忆存储在`MEMORY.md`文件中
4. THE 记忆系统 SHALL 限制单个记忆文件大小不超过100KB
5. THE 系统 SHALL 在会话开始时加载最近2天的每日笔记
6. THE 记忆系统 SHALL 支持通过关键词搜索历史记忆

## 技术架构约束

### 必须使用

| 组件 | 选择 | 原因 |
|------|------|------|
| 编程语言 | Go 1.21+ | 静态链接、高效GC、跨平台 |
| 配置格式 | JSON5 | 兼容OpenClaw、支持注释 |
| 日志库 | slog（标准库）或zerolog | 零分配、结构化 |
| HTTP客户端 | net/http（标准库） | 无依赖、稳定 |
| WebSocket | golang.org/x/net/websocket | 轻量级 |

### 禁止使用

| 组件 | 原因 |
|------|------|
| cgo | 增加二进制大小、破坏静态链接 |
| Docker SDK | 资源消耗过大 |
| 浏览器自动化 | 资源消耗过大 |
| 图像处理库 | 资源消耗过大 |
| 重型ORM | 资源消耗过大 |

### 编译优化

```bash
# 生产构建命令
CGO_ENABLED=0 GOOS=linux GOARCH=arm GOARM=7 go build \
  -ldflags="-s -w" \
  -trimpath \
  -o mujibot-armv7

# 使用UPX进一步压缩（可选）
upx --best mujibot-armv7
```

## 配置文件示例

```json5
{
  // 服务器配置
  server: {
    port: 8080,
    healthCheck: true,
  },
  
  // 消息渠道配置
  channels: {
    telegram: {
      enabled: true,
      token: "${TELEGRAM_BOT_TOKEN}", // 从环境变量读取
      allowedUsers: [123456789], // 授权用户ID白名单
    },
    discord: {
      enabled: false,
      token: "${DISCORD_BOT_TOKEN}",
      allowedGuilds: [],
    },
  },
  
  // LLM配置
  llm: {
    provider: "openai", // openai | anthropic | ollama
    model: "gpt-4o-mini",
    apiKey: "${OPENAI_API_KEY}",
    baseURL: "", // 可选，用于兼容API
    timeout: 60,
    maxRetries: 3,
  },
  
  // 智能体配置
  agents: {
    default: {
      name: "Mujibot",
      systemPrompt: "你是一个运行在低功耗设备上的AI助手。",
    },
  },
  
  // 工具配置
  tools: {
    workDir: "/home/user/workspace",
    timeout: 30,
    confirmDangerous: true,
  },
  
  // 会话配置
  session: {
    maxMessages: 20,
    idleTimeout: 3600, // 1小时
  },
  
  // 日志配置
  logging: {
    level: "info", // debug | info | warn | error
    file: "/var/log/mujibot/app.log",
    maxSize: 5, // MB
    format: "json",
  },
}
```

## 非功能性需求

### 性能指标

| 指标 | 目标值 |
|------|--------|
| 空闲内存占用 | <10MB |
| 峰值内存占用 | <50MB |
| 启动时间 | <2秒 |
| 消息响应延迟 | <2秒（不含LLM） |
| 二进制大小 | <15MB（压缩后<8MB） |
| 7天内存泄漏 | 0 |

### 可靠性指标

| 指标 | 目标值 |
|------|--------|
| 可用性 | 99.5% |
| 故障恢复时间 | <30秒 |
| 数据丢失率 | 0（会话可选持久化） |

## 附录

### A. 玩客云刷机参考

玩客云可刷入Armbian系统以获得完整的Linux环境：
- 支持的Armbian版本：Armbian 23.08+ (Bookworm/Bullseye)
- 推荐使用SD卡启动以保护eMMC
- 刷机教程参考：https://blog.zeruns.com/archives/671.html

### B. 与OpenClaw功能对比

| 功能 | OpenClaw | Mujibot | 说明 |
|------|----------|---------|------|
| Telegram | ✅ | ✅ | 完全支持 |
| Discord | ✅ | ✅ | 完全支持 |
| WhatsApp | ✅ | ❌ | 需要额外库，资源消耗大 |
| iMessage | ✅ | ❌ | 仅macOS |
| Slack | ✅ | ❌ | 企业场景，资源消耗大 |
| 飞书/钉钉 | ✅ | ❌ | 国内平台，可后续扩展 |
| 浏览器控制 | ✅ | ❌ | 资源限制 |
| Docker沙箱 | ✅ | ❌ | 资源限制 |
| Web UI | ✅ | ❌ | 资源限制 |
| 图像分析 | ✅ | ❌ | 资源限制 |
| 语音合成 | ✅ | ❌ | 资源限制 |
| 多智能体 | ✅ | ✅ | 完全支持 |
| 工具调用 | ✅ | ✅ | 完全支持 |
| 会话管理 | ✅ | ✅ | 完全支持 |
| 记忆系统 | ✅ | ✅ | 完全支持 |
| 热重载 | ✅ | ✅ | 完全支持 |
| 健康检查 | ✅ | ✅ | 完全支持 |
