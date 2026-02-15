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
	Server   ServerConfig            `json:"server"`
	Channels ChannelsConfig          `json:"channels"`
	LLM      LLMConfig               `json:"llm"`
	Agents   map[string]AgentConfig  `json:"agents"`
	Tools    ToolsConfig             `json:"tools"`
	Session  SessionConfig           `json:"session"`
	Logging  LoggingConfig           `json:"logging"`
	Memory   MemoryConfig            `json:"memory"`
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
	Enabled      bool    `json:"enabled"`
	Token        string  `json:"token"`
	AllowedUsers []int64 `json:"allowedUsers"`
}

// DiscordConfig Discord配置
type DiscordConfig struct {
	Enabled       bool     `json:"enabled"`
	Token         string   `json:"token"`
	AllowedGuilds []string `json:"allowedGuilds"`
}

// FeishuConfig 飞书配置
type FeishuConfig struct {
	Enabled      bool     `json:"enabled"`
	AppID        string   `json:"appId"`
	AppSecret    string   `json:"appSecret"`
	EncryptKey   string   `json:"encryptKey"`
	AllowedUsers []string `json:"allowedUsers"`
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

// AgentConfig 智能体配置
type AgentConfig struct {
	Name         string   `json:"name"`
	SystemPrompt string   `json:"systemPrompt"`
	Tools        []string `json:"tools"`
}

// ToolsConfig 工具配置
type ToolsConfig struct {
	WorkDir          string              `json:"workDir"`
	Timeout          int                 `json:"timeout"`
	ConfirmDangerous bool                `json:"confirmDangerous"`
	AllowedCommands  []string            `json:"allowedCommands"`
	BlockedCommands  []string            `json:"blockedCommands"`
	EnabledTools     map[string]bool     `json:"enabledTools"`     // 工具开关
	CustomAPIs       []CustomAPIConfig   `json:"customAPIs"`       // 用户自定义API
}

// CustomAPIConfig 自定义API配置
type CustomAPIConfig struct {
	Name        string            `json:"name"`        // API名称
	Description string            `json:"description"` // 描述
	URL         string            `json:"url"`         // API URL模板
	Method      string            `json:"method"`      // HTTP方法
	Headers     map[string]string `json:"headers"`     // 请求头
	APIKey      string            `json:"apiKey"`      // API密钥
	Timeout     int               `json:"timeout"`     // 超时时间
	Enabled     bool              `json:"enabled"`     // 是否启用
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
  "agents": {
    "default": {
      "name": "Mujibot",
      "systemPrompt": "你是一个运行在低功耗ARM设备上的AI助手。你高效、简洁、 helpful。",
      "tools": ["read_file", "write_file", "execute_command", "list_directory"]
    }
  },
  "tools": {
    "workDir": "/tmp/mujibot",
    "timeout": 30,
    "confirmDangerous": true,
    "allowedCommands": [],
    "blockedCommands": ["reboot", "shutdown", "init", "poweroff", "halt"],
    "enabledTools": {
      "read_file": true,
      "write_file": true,
      "list_directory": true,
      "execute_command": true,
      "web_search": true,
      "http_request": true,
      "weather": true,
      "ip_info": true,
      "exchange_rate": true,
      "memory_read": true,
      "memory_write": true
    },
    "customAPIs": [
      {
        "name": "example_api",
        "description": "示例自定义API",
        "url": "https://api.example.com/data?q={query}",
        "method": "GET",
        "headers": {
          "Authorization": "Bearer {apiKey}"
        },
        "apiKey": "${EXAMPLE_API_KEY}",
        "timeout": 10,
        "enabled": false
      }
    ]
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
