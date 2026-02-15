package feishu

import (
	"bytes"
	"crypto/aes"
	"crypto/cipher"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"sync"
	"time"

	"github.com/HaohanHe/mujibot/internal/config"
	"github.com/HaohanHe/mujibot/internal/logger"
)

// Bot 飞书Bot
type Bot struct {
	appID          string
	appSecret      string
	encryptKey     string
	allowedUsers   map[string]bool
	apiURL         string
	client         *http.Client
	accessToken    string
	tokenExpireAt  time.Time
	handlers       []MessageHandler
	mu             sync.RWMutex
	log            *logger.Logger
}

// MessageHandler 消息处理函数
type MessageHandler func(userID, username, content string) (string, error)

// Event 飞书事件
type Event struct {
	UUID      string          `json:"uuid"`
	Token     string          `json:"token"`
	TS        string          `json:"ts"`
	Type      string          `json:"type"`
	Event     json.RawMessage `json:"event"`
	Challenge string          `json:"challenge"`
	Encrypt   string          `json:"encrypt"`
}

// MessageEvent 消息事件
type MessageEvent struct {
	Sender struct {
		SenderID struct {
			UnionID string `json:"union_id"`
			UserID  string `json:"user_id"`
			OpenID  string `json:"open_id"`
		} `json:"sender_id"`
		SenderType string `json:"sender_type"`
	} `json:"sender"`
	Message struct {
		MessageID   string `json:"message_id"`
		MessageType string `json:"message_type"`
		Content     string `json:"content"`
		ChatID      string `json:"chat_id"`
		ChatType    string `json:"chat_type"`
	} `json:"message"`
}

// NewBot 创建飞书Bot
func NewBot(cfg config.FeishuConfig, log *logger.Logger) *Bot {
	allowedUsers := make(map[string]bool)
	for _, uid := range cfg.AllowedUsers {
		allowedUsers[uid] = true
	}

	return &Bot{
		appID:        cfg.AppID,
		appSecret:    cfg.AppSecret,
		encryptKey:   cfg.EncryptKey,
		allowedUsers: allowedUsers,
		apiURL:       "https://open.feishu.cn/open-apis",
		client:       &http.Client{Timeout: 30 * time.Second},
		handlers:     make([]MessageHandler, 0),
		log:          log,
	}
}

// OnMessage 注册消息处理器
func (b *Bot) OnMessage(handler MessageHandler) {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.handlers = append(b.handlers, handler)
}

// Start 启动Bot（飞书通过Webhook接收事件，不需要主动启动）
func (b *Bot) Start() error {
	b.log.Info("feishu bot initialized", "app_id", b.appID)
	return nil
}

// Stop 停止Bot
func (b *Bot) Stop() {
	b.log.Info("feishu bot stopped")
}

// HandleEvent 处理飞书事件（由HTTP服务器调用）
func (b *Bot) HandleEvent(body []byte) ([]byte, error) {
	var event Event
	if err := json.Unmarshal(body, &event); err != nil {
		return nil, fmt.Errorf("failed to parse event: %w", err)
	}

	// 处理URL验证（首次配置Webhook时）
	if event.Challenge != "" {
		return json.Marshal(map[string]string{"challenge": event.Challenge})
	}

	// 解密事件（如果配置了加密）
	if event.Encrypt != "" && b.encryptKey != "" {
		decrypted, err := b.decrypt(event.Encrypt)
		if err != nil {
			return nil, fmt.Errorf("failed to decrypt event: %w", err)
		}
		if err := json.Unmarshal(decrypted, &event); err != nil {
			return nil, fmt.Errorf("failed to parse decrypted event: %w", err)
		}
	}

	// 处理不同类型的事件
	switch event.Type {
	case "url_verification":
		return json.Marshal(map[string]string{"challenge": event.Challenge})

	case "event_callback":
		if err := b.handleEventCallback(event.Event); err != nil {
			b.log.Error("failed to handle event callback", "error", err)
		}
	}

	return json.Marshal(map[string]string{"status": "ok"})
}

// handleEventCallback 处理事件回调
func (b *Bot) handleEventCallback(eventData json.RawMessage) error {
	// 解析事件体
	var eventBody struct {
		Type    string          `json:"type"`
		Message json.RawMessage `json:"message"`
	}
	if err := json.Unmarshal(eventData, &eventBody); err != nil {
		return err
	}

	// 只处理消息事件
	if eventBody.Type != "im.message.receive_v1" {
		return nil
	}

	// 解析消息事件
	var msgEvent MessageEvent
	if err := json.Unmarshal(eventData, &msgEvent); err != nil {
		return err
	}

	userID := msgEvent.Sender.SenderID.OpenID
	username := msgEvent.Sender.SenderID.UserID
	content := b.parseMessageContent(msgEvent.Message.Content, msgEvent.Message.MessageType)

	// 检查用户权限
	if len(b.allowedUsers) > 0 && !b.allowedUsers[userID] {
		b.log.Warn("unauthorized user", "user_id", userID)
		b.SendMessage(userID, "⛔ 未授权的用户")
		return nil
	}

	b.log.Info("feishu message received", "user_id", userID, "username", username, "content", truncate(content, 50))

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

			response, err := h(userID, username, content)
			if err != nil {
				b.log.Error("handler error", "error", err)
				b.SendMessage(userID, "❌ 处理消息时出错: "+err.Error())
				return
			}

			if response != "" {
				if err := b.SendMessage(userID, response); err != nil {
					b.log.Error("failed to send message", "error", err)
				}
			}
		}(handler)
	}

	return nil
}

// SendMessage 发送消息
func (b *Bot) SendMessage(userID, content string) error {
	// 确保有访问令牌
	if err := b.ensureAccessToken(); err != nil {
		return fmt.Errorf("failed to get access token: %w", err)
	}

	// 构建消息内容
	msgContent := map[string]interface{}{
		"text": content,
	}
	contentData, _ := json.Marshal(msgContent)

	reqBody := map[string]interface{}{
		"receive_id": userID,
		"content":    string(contentData),
		"msg_type":   "text",
	}

	return b.apiRequest("POST", "/im/v1/messages?receive_id_type=open_id", reqBody)
}

// SendRichMessage 发送富文本消息
func (b *Bot) SendRichMessage(userID string, content map[string]interface{}) error {
	// 确保有访问令牌
	if err := b.ensureAccessToken(); err != nil {
		return fmt.Errorf("failed to get access token: %w", err)
	}

	contentData, _ := json.Marshal(content)

	reqBody := map[string]interface{}{
		"receive_id": userID,
		"content":    string(contentData),
		"msg_type":   "post",
	}

	return b.apiRequest("POST", "/im/v1/messages?receive_id_type=open_id", reqBody)
}

// ensureAccessToken 确保有有效的访问令牌
func (b *Bot) ensureAccessToken() error {
	b.mu.Lock()
	defer b.mu.Unlock()

	// 检查现有令牌是否有效
	if b.accessToken != "" && time.Now().Before(b.tokenExpireAt.Add(-60*time.Second)) {
		return nil
	}

	// 获取新令牌
	reqBody := map[string]interface{}{
		"app_id":     b.appID,
		"app_secret": b.appSecret,
	}

	data, err := json.Marshal(reqBody)
	if err != nil {
		return err
	}

	resp, err := b.client.Post(
		b.apiURL+"/auth/v3/tenant_access_token/internal",
		"application/json",
		bytes.NewReader(data),
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
		Code              int    `json:"code"`
		Msg               string `json:"msg"`
		TenantAccessToken string `json:"tenant_access_token"`
		Expire            int    `json:"expire"`
	}

	if err := json.Unmarshal(body, &result); err != nil {
		return err
	}

	if result.Code != 0 {
		return fmt.Errorf("feishu auth error: %s", result.Msg)
	}

	b.accessToken = result.TenantAccessToken
	b.tokenExpireAt = time.Now().Add(time.Duration(result.Expire) * time.Second)

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

	req.Header.Set("Authorization", "Bearer "+b.accessToken)
	req.Header.Set("Content-Type", "application/json")

	resp, err := b.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		respBody, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("feishu api error: %s - %s", resp.Status, string(respBody))
	}

	return nil
}

// parseMessageContent 解析消息内容
func (b *Bot) parseMessageContent(content, msgType string) string {
	switch msgType {
	case "text":
		var textContent struct {
			Text string `json:"text"`
		}
		if err := json.Unmarshal([]byte(content), &textContent); err == nil {
			return textContent.Text
		}
		return content
	default:
		return content
	}
}

// decrypt 解密事件数据
func (b *Bot) decrypt(encrypt string) ([]byte, error) {
	if b.encryptKey == "" {
		return nil, fmt.Errorf("encrypt key not configured")
	}

	// Base64解码
	ciphertext, err := base64.StdEncoding.DecodeString(encrypt)
	if err != nil {
		return nil, err
	}

	// 计算AES密钥
	hash := sha256.Sum256([]byte(b.encryptKey))
	key := hash[:16]

	// AES-128-CBC解密
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}

	if len(ciphertext) < aes.BlockSize {
		return nil, fmt.Errorf("ciphertext too short")
	}

	iv := ciphertext[:aes.BlockSize]
	ciphertext = ciphertext[aes.BlockSize:]

	mode := cipher.NewCBCDecrypter(block, iv)
	mode.CryptBlocks(ciphertext, ciphertext)

	// 去除PKCS7填充
	padding := int(ciphertext[len(ciphertext)-1])
	if padding > aes.BlockSize || padding == 0 {
		return nil, fmt.Errorf("invalid padding")
	}

	return ciphertext[:len(ciphertext)-padding], nil
}

// truncate 截断字符串
func truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "..."
}

// GetWebhookHandler 获取Webhook处理函数（用于HTTP服务器）
func (b *Bot) GetWebhookHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		body, err := io.ReadAll(r.Body)
		if err != nil {
			http.Error(w, "Failed to read body", http.StatusBadRequest)
			return
		}
		defer r.Body.Close()

		response, err := b.HandleEvent(body)
		if err != nil {
			b.log.Error("failed to handle event", "error", err)
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.Write(response)
	}
}
