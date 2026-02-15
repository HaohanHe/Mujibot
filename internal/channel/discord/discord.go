package discord

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"sync"
	"time"

	"github.com/HaohanHe/mujibot/internal/config"
	"github.com/HaohanHe/mujibot/internal/logger"
)

// Bot Discord Bot
type Bot struct {
	token         string
	allowedGuilds map[string]bool
	apiURL        string
	gatewayURL    string
	client        *http.Client
	wsConn        *WebSocketConn
	handlers      []MessageHandler
	mu            sync.RWMutex
	running       bool
	stopCh        chan struct{}
	sequence      int64
	sessionID     string
	log           *logger.Logger
}

// MessageHandler 消息处理函数
type MessageHandler func(userID, username, content, channelID string) (string, error)

// GatewayPayload Discord网关消息
type GatewayPayload struct {
	Op int             `json:"op"`
	D  json.RawMessage `json:"d"`
	S  int64           `json:"s"`
	T  string          `json:"t"`
}

// GatewayHello 网关Hello消息
type GatewayHello struct {
	HeartbeatInterval int `json:"heartbeat_interval"`
}

// GatewayIdentify 网关身份验证
type GatewayIdentify struct {
	Token      string                 `json:"token"`
	Properties map[string]interface{} `json:"properties"`
	Intents    int                    `json:"intents"`
}

// Message Discord消息
type Message struct {
	ID        string `json:"id"`
	ChannelID string `json:"channel_id"`
	GuildID   string `json:"guild_id"`
	Author    struct {
		ID       string `json:"id"`
		Username string `json:"username"`
		Bot      bool   `json:"bot"`
	} `json:"author"`
	Content string `json:"content"`
}

// NewBot 创建Discord Bot
func NewBot(cfg config.DiscordConfig, log *logger.Logger) *Bot {
	allowedGuilds := make(map[string]bool)
	for _, gid := range cfg.AllowedGuilds {
		allowedGuilds[gid] = true
	}

	return &Bot{
		token:         cfg.Token,
		allowedGuilds: allowedGuilds,
		apiURL:        "https://discord.com/api/v10",
		gatewayURL:    "wss://gateway.discord.gg/?v=10&encoding=json",
		client:        &http.Client{Timeout: 30 * time.Second},
		handlers:      make([]MessageHandler, 0),
		stopCh:        make(chan struct{}),
		log:           log,
	}
}

// OnMessage 注册消息处理器
func (b *Bot) OnMessage(handler MessageHandler) {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.handlers = append(b.handlers, handler)
}

// Start 启动Bot
func (b *Bot) Start() error {
	b.mu.Lock()
	if b.running {
		b.mu.Unlock()
		return fmt.Errorf("bot already running")
	}
	b.running = true
	b.mu.Unlock()

	b.log.Info("discord bot starting")

	// 获取网关URL
	if err := b.getGatewayURL(); err != nil {
		return fmt.Errorf("failed to get gateway url: %w", err)
	}

	// 连接WebSocket
	if err := b.connectWebSocket(); err != nil {
		return fmt.Errorf("failed to connect websocket: %w", err)
	}

	return nil
}

// Stop 停止Bot
func (b *Bot) Stop() {
	b.mu.Lock()
	if !b.running {
		b.mu.Unlock()
		return
	}
	b.running = false
	b.mu.Unlock()

	if b.wsConn != nil {
		b.wsConn.Close()
	}

	close(b.stopCh)
	b.log.Info("discord bot stopped")
}

// IsRunning 检查是否运行中
func (b *Bot) IsRunning() bool {
	b.mu.RLock()
	defer b.mu.RUnlock()
	return b.running
}

// SendMessage 发送消息
func (b *Bot) SendMessage(channelID, content string) error {
	// 限制消息长度
	if len(content) > 2000 {
		content = content[:1997] + "..."
	}

	reqBody := map[string]interface{}{
		"content": content,
	}

	return b.apiRequest("POST", "/channels/"+channelID+"/messages", reqBody)
}

// getGatewayURL 获取网关URL
func (b *Bot) getGatewayURL() error {
	resp, err := b.client.Get(b.apiURL + "/gateway")
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}

	var result struct {
		URL string `json:"url"`
	}

	if err := json.Unmarshal(body, &result); err != nil {
		return err
	}

	if result.URL != "" {
		b.gatewayURL = result.URL + "/?v=10&encoding=json"
	}

	return nil
}

// connectWebSocket 连接WebSocket
func (b *Bot) connectWebSocket() error {
	// 使用HTTP轮询作为简化实现
	go b.pollLoop()
	return nil
}

// pollLoop 轮询循环（简化实现）
func (b *Bot) pollLoop() {
	b.log.Info("discord bot using http polling mode")

	for {
		select {
		case <-b.stopCh:
			return
		default:
			// Discord Bot主要通过Webhook接收消息
			// 这里简化处理，实际使用时需要设置HTTP服务器接收Webhook
			time.Sleep(5 * time.Second)
		}
	}
}

// HandleWebhook 处理Webhook（需要外部HTTP服务器调用）
func (b *Bot) HandleWebhook(body []byte) error {
	var interaction struct {
		Type int `json:"type"`
		Data struct {
			Name string `json:"name"`
		} `json:"data"`
		Member struct {
			User struct {
				ID       string `json:"id"`
				Username string `json:"username"`
			} `json:"user"`
		} `json:"member"`
		ChannelID string `json:"channel_id"`
		GuildID   string `json:"guild_id"`
		Token     string `json:"token"`
		ID        string `json:"id"`
	}

	if err := json.Unmarshal(body, &interaction); err != nil {
		return err
	}

	// 处理斜杠命令
	if interaction.Type == 2 { // ApplicationCommand
		userID := interaction.Member.User.ID
		username := interaction.Member.User.Username
		channelID := interaction.ChannelID

		// 检查Guild权限
		if len(b.allowedGuilds) > 0 && !b.allowedGuilds[interaction.GuildID] {
			b.log.Warn("unauthorized guild", "guild_id", interaction.GuildID)
			return nil
		}

		content := "/" + interaction.Data.Name

		b.log.Info("discord command received", "user_id", userID, "username", username, "command", content)

		// 调用处理器
		b.mu.RLock()
		handlers := make([]MessageHandler, len(b.handlers))
		copy(handlers, b.handlers)
		b.mu.RUnlock()

		for _, handler := range handlers {
			go func(h MessageHandler) {
				defer func() {
					if r := recover(); r != nil {
						b.log.Error("handler panic", "error", r)
					}
				}()

				response, err := h(userID, username, content, channelID)
				if err != nil {
					b.log.Error("handler error", "error", err)
					return
				}

				if response != "" {
					if err := b.SendMessage(channelID, response); err != nil {
						b.log.Error("failed to send message", "error", err)
					}
				}
			}(handler)
		}
	}

	return nil
}

// apiRequest 发送API请求
func (b *Bot) apiRequest(method, endpoint string, reqBody map[string]interface{}) error {
	var body io.Reader
	if reqBody != nil {
		data, err := json.Marshal(reqBody)
		if err != nil {
			return err
		}
		body = bytes.NewReader(data)
	}

	req, err := http.NewRequest(method, b.apiURL+endpoint, body)
	if err != nil {
		return err
	}

	req.Header.Set("Authorization", "Bot "+b.token)
	req.Header.Set("Content-Type", "application/json")

	resp, err := b.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		respBody, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("discord api error: %s - %s", resp.Status, string(respBody))
	}

	return nil
}

// WebSocketConn WebSocket连接（简化）
type WebSocketConn struct {
	conn interface{}
}

// Close 关闭连接
func (w *WebSocketConn) Close() error {
	return nil
}

// truncate 截断字符串
func truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "..."
}
