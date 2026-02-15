package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"sync"

	"github.com/fsnotify/fsnotify"
	"github.com/HaohanHe/mujibot/internal/logger"
)

// Config 主配置结构
type Config struct {
	Server     ServerConfig            `json:"server"`
	Channels   ChannelsConfig          `json:"channels"`
	LLM        LLMConfig               `json:"llm"`
	LLMPresets map[string]LLMPreset    `json:"llmPresets"`
	Language   LanguageConfig          `json:"language"`
	Agents     map[string]AgentConfig  `json:"agents"`
	Tools      ToolsConfig             `json:"tools"`
	Session    SessionConfig           `json:"session"`
	Logging    LoggingConfig           `json:"logging"`
	Memory     MemoryConfig            `json:"memory"`
}

// ServerConfig 服务器配置
type ServerConfig struct {
	Port        int  `json:"port"`
	HealthCheck bool `json:"healthCheck"`
}

// ChannelsConfig 消息渠道配置
type ChannelsConfig struct {
	Telegram TelegramConfig `json:"telegram"`
	Discord  DiscordConfig  `json:"discord"`
	Feishu   FeishuConfig   `json:"feishu"`
}

// TelegramConfig Telegram配置
type TelegramConfig struct {
	Enabled       bool    `json:"enabled"`
	Token         string  `json:"token"`
	AllowedUsers  []int64 `json:"allowedUsers"`
	NotifyEnabled bool    `json:"notifyEnabled"` // 启用通知
}

// DiscordConfig Discord配置
type DiscordConfig struct {
	Enabled       bool     `json:"enabled"`
	Token         string   `json:"token"`
	AllowedGuilds []string `json:"allowedGuilds"`
	NotifyEnabled bool     `json:"notifyEnabled"` // 启用通知
}

// FeishuConfig 飞书配置
type FeishuConfig struct {
	Enabled       bool     `json:"enabled"`
	AppID         string   `json:"appId"`
	AppSecret     string   `json:"appSecret"`
	EncryptKey    string   `json:"encryptKey"`
	AllowedUsers  []string `json:"allowedUsers"`
	NotifyEnabled bool     `json:"notifyEnabled"` // 启用通知
}

// LLMConfig LLM提供商配置
type LLMConfig struct {
	Provider   string `json:"provider"`
	Model      string `json:"model"`
	APIKey     string `json:"apiKey"`
	BaseURL    string `json:"baseURL"`
	Timeout    int    `json:"timeout"`
	MaxRetries int    `json:"maxRetries"`
}

// LLMPreset LLM预设配置
type LLMPreset struct {
	Name        string   `json:"name"`
	BaseURL     string   `json:"baseURL"`
	Models      []string `json:"models"`
	Description string   `json:"description"`
}

// LanguageConfig 语言配置
type LanguageConfig struct {
	Default  string   `json:"default"`
	Current  string   `json:"current"`
	Supported []string `json:"supported"`
}

// AgentConfig 智能体配置
type AgentConfig struct {
	Name         string   `json:"name"`
	SystemPrompt string   `json:"systemPrompt"`
	Tools        []string `json:"tools"`
}

// ToolsConfig 工具配置
type ToolsConfig struct {
	WorkDir              string            `json:"workDir"`
	Timeout              int               `json:"timeout"`
	ConfirmDangerous     bool              `json:"confirmDangerous"`     // 高危操作需确认
	UnattendedMode       bool              `json:"unattendedMode"`       // 无人值守模式
	AlwaysAllowDangerous []string          `json:"alwaysAllowDangerous"` // 始终允许的危险操作
	AllowedCommands      []string          `json:"allowedCommands"`
	BlockedCommands      []string          `json:"blockedCommands"`
	EnabledTools         map[string]bool   `json:"enabledTools"`     // 工具开关
	WebSearchEnabled     bool              `json:"webSearchEnabled"` // 联网搜索开关
	TerminalEnabled      bool              `json:"terminalEnabled"`  // 终端接管开关
	CustomAPIs           []CustomAPIConfig `json:"customAPIs"`       // 用户自定义API
}

// CustomAPIConfig 自定义API配置
type CustomAPIConfig struct {
	Name        string            `json:"name"`
	Description string            `json:"description"`
	URL         string            `json:"url"`
	Method      string            `json:"method"`
	Headers     map[string]string `json:"headers"`
	APIKey      string            `json:"apiKey"`
	Timeout     int               `json:"timeout"`
	Enabled     bool              `json:"enabled"`
}

// SessionConfig 会话配置
type SessionConfig struct {
	MaxMessages int `json:"maxMessages"`
	IdleTimeout int `json:"idleTimeout"`
	MaxSessions int `json:"maxSessions"`
}

// LoggingConfig 日志配置
type LoggingConfig struct {
	Level   string `json:"level"`
	File    string `json:"file"`
	MaxSize int    `json:"maxSize"`
	Format  string `json:"format"`
}

// MemoryConfig 记忆系统配置
type MemoryConfig struct {
	Enabled    bool   `json:"enabled"`
	MemoryDir  string `json:"memoryDir"`
	MaxFileSize int   `json:"maxFileSize"`
}

// Manager 配置管理器
type Manager struct {
	config     *Config
	configPath string
	watcher    *fsnotify.Watcher
	mu         sync.RWMutex
	onChange   []func(*Config)
	log        *logger.Logger
}

// NewManager 创建配置管理器
func NewManager(configPath string, log *logger.Logger) (*Manager, error) {
	m := &Manager{
		configPath: configPath,
		onChange:   make([]func(*Config), 0),
		log:        log,
	}

	// 检查配置文件是否存在
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		// 创建默认配置文件
		if err := m.createDefaultConfig(); err != nil {
			return nil, fmt.Errorf("failed to create default config: %w", err)
		}
	}

	// 加载配置
	if err := m.Load(); err != nil {
		return nil, err
	}

	// 启动文件监控
	if err := m.watch(); err != nil {
		log.Warn("failed to watch config file", "error", err)
	}

	return m, nil
}

// Load 加载配置文件
func (m *Manager) Load() error {
	data, err := os.ReadFile(m.configPath)
	if err != nil {
		return fmt.Errorf("failed to read config file: %w", err)
	}

	// 解析JSON5（支持注释和尾随逗号）
	jsonData := stripJSON5Comments(string(data))

	var config Config
	if err := json.Unmarshal([]byte(jsonData), &config); err != nil {
		return fmt.Errorf("failed to parse config file: %w", err)
	}

	// 替换环境变量
	m.replaceEnvVars(&config)

	// 验证配置
	if err := m.validate(&config); err != nil {
		return fmt.Errorf("config validation failed: %w", err)
	}

	m.mu.Lock()
	m.config = &config
	m.mu.Unlock()

	m.log.Info("config loaded successfully", "path", m.configPath)
	return nil
}

// Get 获取当前配置
func (m *Manager) Get() *Config {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.config
}

// Update 更新配置
func (m *Manager) Update(cfg *Config) {
	m.mu.Lock()
	m.config = cfg
	m.mu.Unlock()

	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		m.log.Error("failed to marshal config", "error", err)
		return
	}

	if err := os.WriteFile(m.configPath, data, 0644); err != nil {
		m.log.Error("failed to write config", "error", err)
	}
}

// OnChange 注册配置变更回调
func (m *Manager) OnChange(fn func(*Config)) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.onChange = append(m.onChange, fn)
}

// Close 关闭配置管理器
func (m *Manager) Close() error {
	if m.watcher != nil {
		return m.watcher.Close()
	}
	return nil
}

// createDefaultConfig 创建默认配置文件
func (m *Manager) createDefaultConfig() error {
	defaultConfig := `{
  "server": {
    "port": 8080,
    "healthCheck": true
  },
  "channels": {
    "telegram": {
      "enabled": false,
      "token": "${TELEGRAM_BOT_TOKEN}",
      "allowedUsers": []
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
  "llmPresets": {
    "openai": {
      "name": "OpenAI",
      "baseURL": "https://api.openai.com/v1",
      "models": ["gpt-4o", "gpt-4o-mini", "gpt-4-turbo", "gpt-3.5-turbo", "o1-preview", "o1-mini"],
      "description": "OpenAI GPT models"
    },
    "anthropic": {
      "name": "Anthropic Claude",
      "baseURL": "https://api.anthropic.com/v1",
      "models": ["claude-sonnet-4-20250514", "claude-3-5-sonnet-20241022", "claude-3-haiku-20240307", "claude-3-opus-20240229"],
      "description": "Anthropic Claude models"
    },
    "openai-native": {
      "name": "OpenAI Native",
      "baseURL": "https://api.openai.com/v1",
      "models": ["gpt-4o", "gpt-4o-mini", "o1", "o1-mini", "o3-mini"],
      "description": "OpenAI native API"
    },
    "gemini": {
      "name": "Google Gemini",
      "baseURL": "https://generativelanguage.googleapis.com/v1beta",
      "models": ["gemini-2.0-flash", "gemini-1.5-pro", "gemini-1.5-flash"],
      "description": "Google Gemini models"
    },
    "deepseek": {
      "name": "DeepSeek",
      "baseURL": "https://api.deepseek.com",
      "models": ["deepseek-chat", "deepseek-reasoner", "deepseek-coder"],
      "description": "DeepSeek - high quality, low cost"
    },
    "minimax": {
      "name": "MiniMax",
      "baseURL": "https://api.minimax.chat/v1",
      "models": ["MiniMax-Text-01", "abab6.5s-chat", "abab6.5g-chat"],
      "description": "MiniMax AI models"
    },
    "moonshot": {
      "name": "Moonshot Kimi",
      "baseURL": "https://api.moonshot.cn/v1",
      "models": ["moonshot-v1-8k", "moonshot-v1-32k", "moonshot-v1-128k", "kimi-k2-0711-preview"],
      "description": "Moonshot Kimi - long context"
    },
    "zhipu": {
      "name": "Zhipu GLM",
      "baseURL": "https://open.bigmodel.cn/api/paas/v4",
      "models": ["glm-4", "glm-4-flash", "glm-4-plus", "glm-4-long", "glm-z1-air", "glm-z1-airx"],
      "description": "Zhipu AI GLM models"
    },
    "qwen": {
      "name": "Alibaba Qwen",
      "baseURL": "https://dashscope.aliyuncs.com/compatible-mode/v1",
      "models": ["qwen-turbo", "qwen-plus", "qwen-max", "qwen-long", "qwen-coder-plus", "qwen-2.5-72b-instruct"],
      "description": "Alibaba Tongyi Qwen models"
    },
    "qwen-code": {
      "name": "Qwen Code",
      "baseURL": "https://dashscope.aliyuncs.com/compatible-mode/v1",
      "models": ["qwen-coder-plus", "qwen-coder-turbo"],
      "description": "Qwen Code - coding specialist"
    },
    "doubao": {
      "name": "ByteDance Doubao",
      "baseURL": "https://ark.cn-beijing.volces.com/api/v3",
      "models": ["doubao-pro-32k", "doubao-pro-128k", "doubao-lite-4k"],
      "description": "ByteDance Doubao models"
    },
    "groq": {
      "name": "Groq",
      "baseURL": "https://api.groq.com/openai/v1",
      "models": ["llama-3.3-70b-versatile", "llama-3.1-8b-instant", "mixtral-8x7b-32768", "gemma2-9b-it"],
      "description": "Groq - ultra fast inference"
    },
    "xai": {
      "name": "xAI Grok",
      "baseURL": "https://api.x.ai/v1",
      "models": ["grok-beta", "grok-2-1212", "grok-2-vision-1212"],
      "description": "xAI Grok models"
    },
    "mistral": {
      "name": "Mistral AI",
      "baseURL": "https://api.mistral.ai/v1",
      "models": ["mistral-large-latest", "mistral-medium", "codestral-latest", "ministral-8b-latest"],
      "description": "Mistral AI models"
    },
    "cerebras": {
      "name": "Cerebras",
      "baseURL": "https://api.cerebras.ai/v1",
      "models": ["llama-3.3-70b", "llama-3.1-8b"],
      "description": "Cerebras - fast inference"
    },
    "fireworks": {
      "name": "Fireworks AI",
      "baseURL": "https://api.fireworks.ai/inference/v1",
      "models": ["accounts/fireworks/models/llama-v3-70b-instruct", "accounts/fireworks/models/qwen2p5-72b-instruct"],
      "description": "Fireworks AI models"
    },
    "sambanova": {
      "name": "SambaNova",
      "baseURL": "https://api.sambanova.ai/v1",
      "models": ["Meta-Llama-3.1-8B-Instruct", "Meta-Llama-3.1-70B-Instruct"],
      "description": "SambaNova models"
    },
    "siliconflow": {
      "name": "SiliconFlow",
      "baseURL": "https://api.siliconflow.cn/v1",
      "models": ["Qwen/Qwen2.5-7B-Instruct", "deepseek-ai/DeepSeek-V2.5", "meta-llama/Llama-3.3-70B-Instruct"],
      "description": "SiliconFlow - multi-model proxy"
    },
    "openrouter": {
      "name": "OpenRouter",
      "baseURL": "https://openrouter.ai/api/v1",
      "models": ["anthropic/claude-sonnet-4", "openai/gpt-4o", "deepseek/deepseek-chat"],
      "description": "OpenRouter - 500+ models"
    },
    "deepinfra": {
      "name": "DeepInfra",
      "baseURL": "https://api.deepinfra.com/v1/openai",
      "models": ["meta-llama/Llama-3.3-70B-Instruct", "deepseek-ai/DeepSeek-R1"],
      "description": "DeepInfra models"
    },
    "zai": {
      "name": "Z.ai",
      "baseURL": "https://api.z.ai/v1",
      "models": ["zai-1", "zai-2"],
      "description": "Z.ai models"
    },
    "chutes": {
      "name": "Chutes",
      "baseURL": "https://llm.chutes.ai/v1",
      "models": ["chutes-ai/Llama-3.3-70B-Instruct"],
      "description": "Chutes AI"
    },
    "featherless": {
      "name": "Featherless",
      "baseURL": "https://api.featherless.ai/v1",
      "models": ["featherless-ai/Qwen2.5-72B-Instruct"],
      "description": "Featherless AI"
    },
    "ollama": {
      "name": "Ollama Local",
      "baseURL": "http://localhost:11434/v1",
      "models": ["llama3.2", "llama3.1", "qwen2.5", "deepseek-r1", "codellama", "mistral"],
      "description": "Ollama local models"
    },
    "lmstudio": {
      "name": "LM Studio Local",
      "baseURL": "http://localhost:1234/v1",
      "models": ["local-model"],
      "description": "LM Studio local models"
    },
    "litellm": {
      "name": "LiteLLM Proxy",
      "baseURL": "http://localhost:4000/v1",
      "models": ["gpt-4", "claude-3-sonnet"],
      "description": "LiteLLM proxy server"
    }
  },
  "language": {
    "default": "en-US",
    "current": "en-US",
    "supported": ["en-US", "zh-CN", "ja-JP"]
  },
  "agents": {
    "default": {
      "name": "Mujibot",
      "systemPrompt": "You are an AI assistant running on a low-power device. You are efficient, concise, and helpful.",
      "tools": ["read_file", "write_file", "execute_command", "list_directory"]
    }
  },
  "tools": {
    "workDir": "/tmp/mujibot",
    "timeout": 30,
    "confirmDangerous": true,
    "unattendedMode": false,
    "alwaysAllowDangerous": [],
    "allowedCommands": [],
    "blockedCommands": ["reboot", "shutdown", "init", "poweroff", "halt"],
    "enabledTools": {
      "read_file": true,
      "write_file": true,
      "list_directory": true,
      "execute_command": true,
      "weather": true,
      "ip_info": true,
      "exchange_rate": true,
      "memory_read": true,
      "memory_write": true
    },
    "webSearchEnabled": false,
    "terminalEnabled": false,
    "customAPIs": []
  },
  "session": {
    "maxMessages": 20,
    "idleTimeout": 3600,
    "maxSessions": 100
  },
  "logging": {
    "level": "info",
    "file": "",
    "maxSize": 5,
    "format": "json"
  },
  "memory": {
    "enabled": true,
    "memoryDir": "./memory",
    "maxFileSize": 102400
  }
}`

	dir := filepath.Dir(m.configPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}

	return os.WriteFile(m.configPath, []byte(defaultConfig), 0644)
}

// replaceEnvVars 替换配置中的环境变量
func (m *Manager) replaceEnvVars(config *Config) {
	config.Channels.Telegram.Token = m.getEnvOrDefault(config.Channels.Telegram.Token, "")
	config.Channels.Discord.Token = m.getEnvOrDefault(config.Channels.Discord.Token, "")
	config.Channels.Feishu.AppID = m.getEnvOrDefault(config.Channels.Feishu.AppID, "")
	config.Channels.Feishu.AppSecret = m.getEnvOrDefault(config.Channels.Feishu.AppSecret, "")
	config.Channels.Feishu.EncryptKey = m.getEnvOrDefault(config.Channels.Feishu.EncryptKey, "")
	config.LLM.APIKey = m.getEnvOrDefault(config.LLM.APIKey, "")
}

// getEnvOrDefault 获取环境变量值
func (m *Manager) getEnvOrDefault(value, defaultValue string) string {
	if !strings.HasPrefix(value, "${") || !strings.HasSuffix(value, "}") {
		return value
	}

	envVar := value[2 : len(value)-1]
	envValue := os.Getenv(envVar)
	if envValue == "" {
		return defaultValue
	}
	return envValue
}

// validate 验证配置
func (m *Manager) validate(config *Config) error {
	// 验证LLM配置
	if config.LLM.Provider == "" {
		return fmt.Errorf("llm.provider is required")
	}
	if config.LLM.APIKey == "" && config.LLM.Provider != "ollama" {
		return fmt.Errorf("llm.apiKey is required for provider %s", config.LLM.Provider)
	}

	// 验证至少启用一个渠道
	if !config.Channels.Telegram.Enabled && !config.Channels.Discord.Enabled && !config.Channels.Feishu.Enabled {
		m.log.Warn("no channel enabled, gateway will not receive messages")
	}

	// 验证工具工作目录
	if config.Tools.WorkDir == "" {
		config.Tools.WorkDir = "/tmp/mujibot"
	}

	return nil
}

// watch 监控配置文件变化
func (m *Manager) watch() error {
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		return err
	}

	m.watcher = watcher

	go func() {
		for {
			select {
			case event, ok := <-watcher.Events:
				if !ok {
					return
				}
				if event.Op&fsnotify.Write == fsnotify.Write {
					m.log.Info("config file changed, reloading")
					if err := m.Load(); err != nil {
						m.log.Error("failed to reload config", "error", err)
					} else {
						m.notifyChange()
					}
				}
			case err, ok := <-watcher.Errors:
				if !ok {
					return
				}
				m.log.Error("config watcher error", "error", err)
			}
		}
	}()

	return watcher.Add(m.configPath)
}

// notifyChange 通知配置变更
func (m *Manager) notifyChange() {
	m.mu.RLock()
	callbacks := make([]func(*Config), len(m.onChange))
	copy(callbacks, m.onChange)
	config := m.config
	m.mu.RUnlock()

	for _, fn := range callbacks {
		go fn(config)
	}
}

// stripJSON5Comments 去除JSON5注释
func stripJSON5Comments(input string) string {
	// 去除单行注释
	singleLineComment := regexp.MustCompile(`//.*$`)
	input = singleLineComment.ReplaceAllString(input, "")

	// 去除多行注释
	multiLineComment := regexp.MustCompile(`/[\*][\s\S]*?\*/`)
	input = multiLineComment.ReplaceAllString(input, "")

	// 去除尾随逗号
	trailingComma := regexp.MustCompile(`,(\s*[}\]])`)
	input = trailingComma.ReplaceAllString(input, "$1")

	return input
}
