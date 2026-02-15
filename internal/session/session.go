package session

import (
	"container/list"
	"sync"
	"time"

	"github.com/HaohanHe/mujibot/internal/logger"
)

// Message 消息结构
type Message struct {
	Role      string    `json:"role"`
	Content   string    `json:"content"`
	Timestamp time.Time `json:"timestamp"`
	ToolCalls []ToolCall `json:"tool_calls,omitempty"`
}

// ToolCall 工具调用
type ToolCall struct {
	ID       string `json:"id"`
	Type     string `json:"type"`
	Function struct {
		Name      string `json:"name"`
		Arguments string `json:"arguments"`
	} `json:"function"`
}

// Session 会话结构
type Session struct {
	ID           string
	UserID       string
	Channel      string
	AgentID      string
	Messages     []Message
	LastActivity time.Time
	mu           sync.RWMutex
}

// Manager 会话管理器
type Manager struct {
	sessions     map[string]*list.Element
	lruList      *list.List
	maxMessages  int
	idleTimeout  time.Duration
	maxSessions  int
	mu           sync.RWMutex
	log          *logger.Logger
	cleanupTimer *time.Timer
	stopCh       chan struct{}
}

// sessionEntry LRU列表中的条目
type sessionEntry struct {
	key     string
	session *Session
}

// NewManager 创建会话管理器
func NewManager(maxMessages, idleTimeoutSec, maxSessions int, log *logger.Logger) *Manager {
	m := &Manager{
		sessions:    make(map[string]*list.Element),
		lruList:     list.New(),
		maxMessages: maxMessages,
		idleTimeout: time.Duration(idleTimeoutSec) * time.Second,
		maxSessions: maxSessions,
		log:         log,
		stopCh:      make(chan struct{}),
	}

	go m.cleanupLoop()

	return m
}

// GetOrCreate 获取或创建会话
func (m *Manager) GetOrCreate(userID, channel, agentID string) *Session {
	key := m.makeKey(userID, channel, agentID)

	m.mu.Lock()
	defer m.mu.Unlock()

	if elem, ok := m.sessions[key]; ok {
		// 移动到队首（最近使用）
		m.lruList.MoveToFront(elem)
		session := elem.Value.(*sessionEntry).session
		session.LastActivity = time.Now()
		return session
	}

	// 检查是否超过最大会话数
	if len(m.sessions) >= m.maxSessions {
		m.evictLRU()
	}

	// 创建新会话
	session := &Session{
		ID:           key,
		UserID:       userID,
		Channel:      channel,
		AgentID:      agentID,
		Messages:     make([]Message, 0, m.maxMessages),
		LastActivity: time.Now(),
	}

	entry := &sessionEntry{key: key, session: session}
	elem := m.lruList.PushFront(entry)
	m.sessions[key] = elem

	m.log.Debug("session created", "key", key, "total", len(m.sessions))
	return session
}

// Get 获取会话（不更新LRU）
func (m *Manager) Get(userID, channel, agentID string) *Session {
	key := m.makeKey(userID, channel, agentID)

	m.mu.Lock()
	defer m.mu.Unlock()

	if elem, ok := m.sessions[key]; ok {
		m.lruList.MoveToFront(elem)
		return elem.Value.(*sessionEntry).session
	}
	return nil
}

// AddMessage 添加消息到会话
func (m *Manager) AddMessage(session *Session, role, content string) {
	session.mu.Lock()
	defer session.mu.Unlock()

	msg := Message{
		Role:      role,
		Content:   content,
		Timestamp: time.Now(),
	}

	session.Messages = append(session.Messages, msg)
	session.LastActivity = time.Now()

	// 限制消息数量
	if len(session.Messages) > m.maxMessages {
		session.Messages = session.Messages[len(session.Messages)-m.maxMessages:]
	}
}

// AddToolCallMessage 添加带工具调用的消息
func (m *Manager) AddToolCallMessage(session *Session, role, content string, toolCalls []ToolCall) {
	session.mu.Lock()
	defer session.mu.Unlock()

	msg := Message{
		Role:      role,
		Content:   content,
		Timestamp: time.Now(),
		ToolCalls: toolCalls,
	}

	session.Messages = append(session.Messages, msg)
	session.LastActivity = time.Now()

	// 限制消息数量
	if len(session.Messages) > m.maxMessages {
		session.Messages = session.Messages[len(session.Messages)-m.maxMessages:]
	}
}

// GetMessages 获取会话消息历史
func (m *Manager) GetMessages(session *Session) []Message {
	session.mu.RLock()
	defer session.mu.RUnlock()

	// 返回副本
	result := make([]Message, len(session.Messages))
	copy(result, session.Messages)
	return result
}

// Clear 清空会话消息
func (m *Manager) Clear(session *Session) {
	session.mu.Lock()
	defer session.mu.Unlock()

	session.Messages = session.Messages[:0]
	session.LastActivity = time.Now()
}

// Delete 删除会话
func (m *Manager) Delete(userID, channel, agentID string) {
	key := m.makeKey(userID, channel, agentID)

	m.mu.Lock()
	defer m.mu.Unlock()

	if elem, ok := m.sessions[key]; ok {
		m.lruList.Remove(elem)
		delete(m.sessions, key)
		m.log.Debug("session deleted", "key", key)
	}
}

// GetStats 获取会话统计
func (m *Manager) GetStats() map[string]interface{} {
	m.mu.RLock()
	defer m.mu.RUnlock()

	return map[string]interface{}{
		"total_sessions": len(m.sessions),
		"max_sessions":   m.maxSessions,
		"max_messages":   m.maxMessages,
		"idle_timeout":   m.idleTimeout.Seconds(),
	}
}

// makeKey 生成会话键
func (m *Manager) makeKey(userID, channel, agentID string) string {
	return channel + ":" + userID + ":" + agentID
}

// evictLRU 淘汰最久未使用的会话
func (m *Manager) evictLRU() {
	elem := m.lruList.Back()
	if elem == nil {
		return
	}

	entry := elem.Value.(*sessionEntry)
	m.lruList.Remove(elem)
	delete(m.sessions, entry.key)

	m.log.Debug("session evicted", "key", entry.key, "reason", "lru")
}

// cleanupLoop 定期清理空闲会话
func (m *Manager) cleanupLoop() {
	ticker := time.NewTicker(time.Minute)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			m.cleanup()
		case <-m.stopCh:
			return
		}
	}
}

// cleanup 清理空闲会话
func (m *Manager) cleanup() {
	m.mu.Lock()
	defer m.mu.Unlock()

	now := time.Now()
	toDelete := make([]string, 0)

	for elem := m.lruList.Back(); elem != nil; {
		entry := elem.Value.(*sessionEntry)
		next := elem.Prev()

		if now.Sub(entry.session.LastActivity) > m.idleTimeout {
			toDelete = append(toDelete, entry.key)
			m.lruList.Remove(elem)
			delete(m.sessions, entry.key)
		}

		elem = next
	}

	if len(toDelete) > 0 {
		m.log.Info("sessions cleaned up", "count", len(toDelete))
	}
}

// Close 关闭会话管理器
func (m *Manager) Close() {
	close(m.stopCh)

	m.mu.Lock()
	defer m.mu.Unlock()

	m.sessions = make(map[string]*list.Element)
	m.lruList.Init()
}
