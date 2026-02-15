package telegram

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/HaohanHe/mujibot/internal/config"
	"github.com/HaohanHe/mujibot/internal/logger"
)

// Bot Telegram Bot
type Bot struct {
	token        string
	allowedUsers map[int64]bool
	apiURL       string
	client       *http.Client
	updateOffset int64
	handlers     []MessageHandler
	mu           sync.RWMutex
	running      bool
	stopCh       chan struct{}
	log          *logger.Logger
}

// MessageHandler 消息处理函数
type MessageHandler func(userID int64, username, text string, chatID int64) (string, error)

// Update Telegram更新
type Update struct {
	UpdateID int64   `json:"update_id"`
	Message  *Message `json:"message"`
}

// Message Telegram消息
type Message struct {
	MessageID int64    `json:"message_id"`
	From      *User    `json:"from"`
	Chat      *Chat    `json:"chat"`
	Date      int64    `json:"date"`
	Text      string   `json:"text"`
}

// User Telegram用户
type User struct {
	ID        int64  `json:"id"`
	IsBot     bool   `json:"is_bot"`
	FirstName string `json:"first_name"`
	Username  string `json:"username"`
}

// Chat Telegram聊天
type Chat struct {
	ID   int64  `json:"id"`
	Type string `json:"type"`
}

// NewBot 创建Telegram Bot
func NewBot(cfg config.TelegramConfig, log *logger.Logger) *Bot {
	allowedUsers := make(map[int64]bool)
	for _, uid := range cfg.AllowedUsers {
		allowedUsers[uid] = true
	}

	return &Bot{
		token:        cfg.Token,
		allowedUsers: allowedUsers,
		apiURL:       "https://api.telegram.org/bot" + cfg.Token,
		client:       &http.Client{Timeout: 30 * time.Second},
		handlers:     make([]MessageHandler, 0),
		stopCh:       make(chan struct{}),
		log:          log,
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

	b.log.Info("telegram bot starting")

	// 获取bot信息
	if err := b.getMe(); err != nil {
		return fmt.Errorf("failed to get bot info: %w", err)
	}

	// 启动轮询
	go b.pollLoop()

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

	close(b.stopCh)
	b.log.Info("telegram bot stopped")
}

// IsRunning 检查是否运行中
func (b *Bot) IsRunning() bool {
	b.mu.RLock()
	defer b.mu.RUnlock()
	return b.running
}

// SendMessage 发送消息
func (b *Bot) SendMessage(chatID int64, text string) error {
	// 限制消息长度
	if len(text) > 4096 {
		text = text[:4093] + "..."
	}

	reqBody := map[string]interface{}{
		"chat_id": chatID,
		"text":    text,
		"parse_mode": "Markdown",
	}

	return b.apiRequest("sendMessage", reqBody)
}

// SendHTMLMessage 发送HTML格式消息
func (b *Bot) SendHTMLMessage(chatID int64, text string) error {
	// 限制消息长度
	if len(text) > 4096 {
		text = text[:4093] + "..."
	}

	reqBody := map[string]interface{}{
		"chat_id":    chatID,
		"text":       text,
		"parse_mode": "HTML",
	}

	return b.apiRequest("sendMessage", reqBody)
}

// getMe 获取Bot信息
func (b *Bot) getMe() error {
	resp, err := b.client.Get(b.apiURL + "/getMe")
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}

	var result struct {
		OK     bool `json:"ok"`
		Result struct {
			Username string `json:"username"`
		} `json:"result"`
	}

	if err := json.Unmarshal(body, &result); err != nil {
		return err
	}

	if !result.OK {
		return fmt.Errorf("telegram api error: %s", string(body))
	}

	b.log.Info("telegram bot connected", "username", result.Result.Username)
	return nil
}

// pollLoop 轮询循环
func (b *Bot) pollLoop() {
	ticker := time.NewTicker(time.Second)
	defer ticker.Stop()

	backoff := time.Second

	for {
		select {
		case <-b.stopCh:
			return
		case <-ticker.C:
			updates, err := b.getUpdates()
			if err != nil {
				b.log.Error("failed to get updates", "error", err)
				// 指数退避
				time.Sleep(backoff)
				if backoff < 5*time.Minute {
					backoff *= 2
				}
				continue
			}

			// 重置退避
			backoff = time.Second

			// 处理更新
			for _, update := range updates {
				b.handleUpdate(update)
				if update.UpdateID >= b.updateOffset {
					b.updateOffset = update.UpdateID + 1
				}
			}
		}
	}
}

// getUpdates 获取更新
func (b *Bot) getUpdates() ([]Update, error) {
	url := fmt.Sprintf("%s/getUpdates?offset=%d&limit=100", b.apiURL, b.updateOffset)

	resp, err := b.client.Get(url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var result struct {
		OK     bool     `json:"ok"`
		Result []Update `json:"result"`
	}

	if err := json.Unmarshal(body, &result); err != nil {
		return nil, err
	}

	if !result.OK {
		return nil, fmt.Errorf("telegram api error: %s", string(body))
	}

	return result.Result, nil
}

// handleUpdate 处理更新
func (b *Bot) handleUpdate(update Update) {
	if update.Message == nil || update.Message.Text == "" {
		return
	}

	msg := update.Message
	userID := msg.From.ID
	username := msg.From.Username
	if username == "" {
		username = msg.From.FirstName
	}

	// 检查用户权限
	if len(b.allowedUsers) > 0 && !b.allowedUsers[userID] {
		b.log.Warn("unauthorized user", "user_id", userID, "username", username)
		b.SendMessage(msg.Chat.ID, "⛔ 未授权的用户")
		return
	}

	b.log.Info("telegram message received", "user_id", userID, "username", username, "text", truncate(msg.Text, 50))

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

			response, err := h(userID, username, msg.Text, msg.Chat.ID)
			if err != nil {
				b.log.Error("handler error", "error", err)
				b.SendMessage(msg.Chat.ID, "❌ 处理消息时出错: "+err.Error())
				return
			}

			if response != "" {
				if err := b.SendMessage(msg.Chat.ID, response); err != nil {
					b.log.Error("failed to send message", "error", err)
				}
			}
		}(handler)
	}
}

// apiRequest 发送API请求
func (b *Bot) apiRequest(method string, reqBody map[string]interface{}) error {
	data, err := json.Marshal(reqBody)
	if err != nil {
		return err
	}

	resp, err := b.client.Post(
		b.apiURL+"/"+method,
		"application/json",
		strings.NewReader(string(data)),
	)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}

	var result struct {
		OK          bool   `json:"ok"`
		Description string `json:"description"`
	}

	if err := json.Unmarshal(body, &result); err != nil {
		return err
	}

	if !result.OK {
		return fmt.Errorf("telegram api error: %s", result.Description)
	}

	return nil
}

// truncate 截断字符串
func truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "..."
}

// ParseUserID 解析用户ID字符串
func ParseUserID(s string) (int64, error) {
	return strconv.ParseInt(s, 10, 64)
}
