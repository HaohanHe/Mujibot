package web

import (
	"encoding/json"
	"fmt"
	"html/template"
	"net/http"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/HaohanHe/mujibot/internal/agent"
	"github.com/HaohanHe/mujibot/internal/config"
	"github.com/HaohanHe/mujibot/internal/health"
	"github.com/HaohanHe/mujibot/internal/logger"
	"github.com/HaohanHe/mujibot/internal/session"
)

// Server Web服务器
type Server struct {
	port       int
	config     *config.Manager
	sessionMgr *session.Manager
	agentRouter *agent.Router
	healthCheck *health.Checker
	log        *logger.Logger
	mu         sync.RWMutex
	clients    map[chan string]bool
	messages   []DebugMessage
	maxMsgs    int
	feishuHandler http.HandlerFunc
}

// DebugMessage 调试消息
type DebugMessage struct {
	Time      string `json:"time"`
	Type      string `json:"type"`
	Source    string `json:"source"`
	Content   string `json:"content"`
	UserID    string `json:"user_id,omitempty"`
	Channel   string `json:"channel,omitempty"`
}

// NewServer 创建Web服务器
func NewServer(port int, cfg *config.Manager, sessionMgr *session.Manager, agentRouter *agent.Router, healthCheck *health.Checker, log *logger.Logger) *Server {
	return &Server{
		port:        port,
		config:      cfg,
		sessionMgr:  sessionMgr,
		agentRouter: agentRouter,
		healthCheck: healthCheck,
		log:         log,
		clients:     make(map[chan string]bool),
		messages:    make([]DebugMessage, 0, 100),
		maxMsgs:     100,
	}
}

// SetFeishuHandler 设置飞书Webhook处理器
func (s *Server) SetFeishuHandler(handler http.HandlerFunc) {
	s.feishuHandler = handler
}

// Start 启动Web服务器
func (s *Server) Start() error {
	mux := http.NewServeMux()

	// 静态文件
	mux.HandleFunc("/", s.handleIndex)
	mux.HandleFunc("/static/", s.handleStatic)

	// API端点
	mux.HandleFunc("/api/status", s.handleStatus)
	mux.HandleFunc("/api/logs", s.handleLogs)
	mux.HandleFunc("/api/sessions", s.handleSessions)
	mux.HandleFunc("/api/agents", s.handleAgents)
	mux.HandleFunc("/api/config", s.handleConfig)
	mux.HandleFunc("/api/send", s.handleSendMessage)
	mux.HandleFunc("/api/messages/stream", s.handleMessageStream)

	// 飞书Webhook
	mux.HandleFunc("/webhook/feishu", s.handleFeishuWebhook)

	s.log.Info("web server starting", "port", s.port)

	go func() {
		if err := http.ListenAndServe(fmt.Sprintf(":%d", s.port), mux); err != nil {
			s.log.Error("web server error", "error", err)
		}
	}()

	return nil
}

// LogMessage 记录调试消息
func (s *Server) LogMessage(msgType, source, content, userID, channel string) {
	msg := DebugMessage{
		Time:    time.Now().Format("15:04:05"),
		Type:    msgType,
		Source:  source,
		Content: content,
		UserID:  userID,
		Channel: channel,
	}

	s.mu.Lock()
	s.messages = append(s.messages, msg)
	if len(s.messages) > s.maxMsgs {
		s.messages = s.messages[len(s.messages)-s.maxMsgs:]
	}

	// 广播到所有连接的客户端
	data, _ := json.Marshal(msg)
	for client := range s.clients {
		select {
		case client <- string(data):
		default:
			// 客户端缓冲区满，跳过
		}
	}
	s.mu.Unlock()
}

// handleIndex 处理首页
func (s *Server) handleIndex(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" {
		http.NotFound(w, r)
		return
	}

	tmpl := template.Must(template.New("index").Parse(indexHTML))
	tmpl.Execute(w, nil)
}

// handleStatic 处理静态文件
func (s *Server) handleStatic(w http.ResponseWriter, r *http.Request) {
	path := strings.TrimPrefix(r.URL.Path, "/static/")

	switch path {
	case "style.css":
		w.Header().Set("Content-Type", "text/css")
		w.Write([]byte(styleCSS))
	case "app.js":
		w.Header().Set("Content-Type", "application/javascript")
		w.Write([]byte(appJS))
	default:
		http.NotFound(w, r)
	}
}

// handleStatus 处理状态API
func (s *Server) handleStatus(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var m runtime.MemStats
	runtime.ReadMemStats(&m)

	status := map[string]interface{}{
		"status":    "ok",
		"timestamp": time.Now().Unix(),
		"memory": map[string]interface{}{
			"alloc":       m.Alloc,
			"total_alloc": m.TotalAlloc,
			"sys":         m.Sys,
			"heap_alloc":  m.HeapAlloc,
			"heap_sys":    m.HeapSys,
		},
		"goroutines": runtime.NumGoroutine(),
		"sessions":   s.sessionMgr.GetStats(),
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(status)
}

// handleLogs 处理日志API
func (s *Server) handleLogs(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	s.mu.RLock()
	logs := make([]DebugMessage, len(s.messages))
	copy(logs, s.messages)
	s.mu.RUnlock()

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(logs)
}

// handleSessions 处理会话API
func (s *Server) handleSessions(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	stats := s.sessionMgr.GetStats()

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(stats)
}

// handleAgents 处理智能体API
func (s *Server) handleAgents(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	agents := s.agentRouter.GetAllAgents()
	agentList := make([]map[string]interface{}, 0, len(agents))
	for id, a := range agents {
		agentList = append(agentList, map[string]interface{}{
			"id":       id,
			"name":     a.Name,
			"model":    a.Provider.GetModel(),
		})
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(agentList)
}

// handleConfig 处理配置API
func (s *Server) handleConfig(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	cfg := s.config.Get()

	// 隐藏敏感信息
	safeConfig := map[string]interface{}{
		"server": map[string]interface{}{
			"port":        cfg.Server.Port,
			"healthCheck": cfg.Server.HealthCheck,
		},
		"channels": map[string]interface{}{
			"telegram": map[string]interface{}{
				"enabled": len(cfg.Channels.Telegram.AllowedUsers) > 0,
			},
			"discord": map[string]interface{}{
				"enabled": len(cfg.Channels.Discord.AllowedGuilds) > 0,
			},
			"feishu": map[string]interface{}{
				"enabled": cfg.Channels.Feishu.AppID != "",
			},
		},
		"llm": map[string]interface{}{
			"provider": cfg.LLM.Provider,
			"model":    cfg.LLM.Model,
			"baseURL":  cfg.LLM.BaseURL,
		},
		"agents": len(cfg.Agents),
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(safeConfig)
}

// handleSendMessage 处理发送消息API
func (s *Server) handleSendMessage(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		Message string `json:"message"`
		AgentID string `json:"agent_id"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// 获取智能体
	agent, err := s.agentRouter.Route("web_user", "web", req.AgentID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// 处理消息
	response, err := s.agentRouter.ProcessMessage(agent, "web_user", "web", req.Message)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// 记录调试消息
	s.LogMessage("user", "web", req.Message, "web_user", "web")
	s.LogMessage("assistant", "web", response, "web_user", "web")

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"response": response})
}

// handleMessageStream 处理消息流（SSE）
func (s *Server) handleMessageStream(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")

	// 创建客户端通道
	client := make(chan string, 10)
	s.mu.Lock()
	s.clients[client] = true
	s.mu.Unlock()

	defer func() {
		s.mu.Lock()
		delete(s.clients, client)
		s.mu.Unlock()
		close(client)
	}()

	// 发送现有消息
	s.mu.RLock()
	for _, msg := range s.messages {
		data, _ := json.Marshal(msg)
		fmt.Fprintf(w, "data: %s\n\n", data)
	}
	s.mu.RUnlock()
	w.(http.Flusher).Flush()

	// 等待新消息
	for {
		select {
		case msg, ok := <-client:
			if !ok {
				return
			}
			fmt.Fprintf(w, "data: %s\n\n", msg)
			w.(http.Flusher).Flush()
		case <-r.Context().Done():
			return
		}
	}
}

// handleFeishuWebhook 处理飞书Webhook
func (s *Server) handleFeishuWebhook(w http.ResponseWriter, r *http.Request) {
	if s.feishuHandler == nil {
		http.Error(w, "Feishu not enabled", http.StatusServiceUnavailable)
		return
	}
	s.feishuHandler(w, r)
}

// indexHTML 首页HTML
const indexHTML = `<!DOCTYPE html>
<html lang="zh-CN">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>Mujibot 调试控制台</title>
    <link rel="stylesheet" href="/static/style.css">
</head>
<body>
    <div class="container">
        <header>
            <h1>🤖 Mujibot 调试控制台</h1>
            <div class="status-indicator" id="status">● 连接中</div>
        </header>

        <div class="main-content">
            <div class="left-panel">
                <div class="panel">
                    <h2>系统状态</h2>
                    <div id="system-status" class="status-grid">
                        <div class="status-item">
                            <span class="label">内存使用:</span>
                            <span class="value" id="memory">-</span>
                        </div>
                        <div class="status-item">
                            <span class="label">Goroutines:</span>
                            <span class="value" id="goroutines">-</span>
                        </div>
                        <div class="status-item">
                            <span class="label">会话数:</span>
                            <span class="value" id="sessions">-</span>
                        </div>
                    </div>
                </div>

                <div class="panel">
                    <h2>配置信息</h2>
                    <div id="config-info" class="config-info">加载中...</div>
                </div>

                <div class="panel">
                    <h2>智能体列表</h2>
                    <div id="agent-list" class="agent-list">加载中...</div>
                </div>
            </div>

            <div class="right-panel">
                <div class="panel chat-panel">
                    <h2>消息调试</h2>
                    <div id="message-log" class="message-log"></div>
                    <div class="input-area">
                        <select id="agent-select">
                            <option value="">默认智能体</option>
                        </select>
                        <input type="text" id="message-input" placeholder="输入消息测试..." maxlength="500">
                        <button id="send-btn">发送</button>
                    </div>
                </div>
            </div>
        </div>

        <footer>
            <p>Mujibot Lightweight AI Assistant | <a href="/api/status" target="_blank">API状态</a></p>
        </footer>
    </div>

    <script src="/static/app.js"></script>
</body>
</html>`

// styleCSS 样式CSS
const styleCSS = `
* {
    margin: 0;
    padding: 0;
    box-sizing: border-box;
}

body {
    font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, sans-serif;
    background: #1a1a2e;
    color: #eee;
    min-height: 100vh;
}

.container {
    max-width: 1400px;
    margin: 0 auto;
    padding: 20px;
}

header {
    display: flex;
    justify-content: space-between;
    align-items: center;
    padding: 20px 0;
    border-bottom: 2px solid #16213e;
    margin-bottom: 20px;
}

header h1 {
    font-size: 24px;
    color: #00d9ff;
}

.status-indicator {
    display: flex;
    align-items: center;
    gap: 8px;
    font-size: 14px;
}

.status-indicator.connected {
    color: #00ff88;
}

.status-indicator.disconnected {
    color: #ff4757;
}

.main-content {
    display: grid;
    grid-template-columns: 350px 1fr;
    gap: 20px;
}

.panel {
    background: #16213e;
    border-radius: 12px;
    padding: 20px;
    margin-bottom: 20px;
}

.panel h2 {
    font-size: 16px;
    color: #00d9ff;
    margin-bottom: 15px;
    padding-bottom: 10px;
    border-bottom: 1px solid #0f3460;
}

.status-grid {
    display: grid;
    gap: 10px;
}

.status-item {
    display: flex;
    justify-content: space-between;
    padding: 8px 0;
    border-bottom: 1px solid #0f3460;
}

.status-item:last-child {
    border-bottom: none;
}

.status-item .label {
    color: #888;
}

.status-item .value {
    color: #00ff88;
    font-family: monospace;
}

.config-info, .agent-list {
    font-size: 13px;
    line-height: 1.6;
}

.config-item {
    padding: 5px 0;
    border-bottom: 1px solid #0f3460;
}

.config-item:last-child {
    border-bottom: none;
}

.config-key {
    color: #888;
}

.config-value {
    color: #00d9ff;
}

.agent-item {
    padding: 10px;
    background: #0f3460;
    border-radius: 6px;
    margin-bottom: 8px;
}

.agent-item:last-child {
    margin-bottom: 0;
}

.agent-name {
    font-weight: bold;
    color: #00d9ff;
}

.agent-model {
    font-size: 12px;
    color: #888;
    margin-top: 4px;
}

.chat-panel {
    height: calc(100vh - 200px);
    display: flex;
    flex-direction: column;
}

.message-log {
    flex: 1;
    overflow-y: auto;
    background: #0a0e27;
    border-radius: 8px;
    padding: 15px;
    margin-bottom: 15px;
    font-family: monospace;
    font-size: 13px;
}

.message-item {
    margin-bottom: 10px;
    padding: 10px;
    border-radius: 6px;
    animation: fadeIn 0.3s ease;
}

@keyframes fadeIn {
    from { opacity: 0; transform: translateY(-10px); }
    to { opacity: 1; transform: translateY(0); }
}

.message-item.user {
    background: #0f3460;
    border-left: 3px solid #00d9ff;
}

.message-item.assistant {
    background: #1a472a;
    border-left: 3px solid #00ff88;
}

.message-item.system {
    background: #3d2817;
    border-left: 3px solid #ffa502;
}

.message-item.error {
    background: #3d1717;
    border-left: 3px solid #ff4757;
}

.message-header {
    display: flex;
    justify-content: space-between;
    margin-bottom: 5px;
    font-size: 11px;
    color: #888;
}

.message-content {
    white-space: pre-wrap;
    word-break: break-word;
    line-height: 1.5;
}

.input-area {
    display: flex;
    gap: 10px;
}

#agent-select {
    width: 120px;
    padding: 10px;
    background: #0f3460;
    border: 1px solid #16213e;
    border-radius: 6px;
    color: #eee;
}

#message-input {
    flex: 1;
    padding: 10px;
    background: #0f3460;
    border: 1px solid #16213e;
    border-radius: 6px;
    color: #eee;
    font-size: 14px;
}

#message-input:focus {
    outline: none;
    border-color: #00d9ff;
}

#send-btn {
    padding: 10px 20px;
    background: #00d9ff;
    color: #1a1a2e;
    border: none;
    border-radius: 6px;
    cursor: pointer;
    font-weight: bold;
    transition: background 0.2s;
}

#send-btn:hover {
    background: #00b8d9;
}

#send-btn:disabled {
    background: #555;
    cursor: not-allowed;
}

footer {
    text-align: center;
    padding: 20px;
    color: #666;
    font-size: 12px;
}

footer a {
    color: #00d9ff;
    text-decoration: none;
}

footer a:hover {
    text-decoration: underline;
}

/* 滚动条样式 */
::-webkit-scrollbar {
    width: 8px;
}

::-webkit-scrollbar-track {
    background: #0a0e27;
}

::-webkit-scrollbar-thumb {
    background: #16213e;
    border-radius: 4px;
}

::-webkit-scrollbar-thumb:hover {
    background: #0f3460;
}

/* 响应式 */
@media (max-width: 900px) {
    .main-content {
        grid-template-columns: 1fr;
    }
    
    .left-panel {
        order: 2;
    }
    
    .right-panel {
        order: 1;
    }
}
`

// appJS JavaScript
const appJS = `
let eventSource = null;
let agents = [];

function init() {
    connectEventStream();
    loadStatus();
    loadConfig();
    loadAgents();
    setInterval(loadStatus, 5000);
    document.getElementById('send-btn').addEventListener('click', sendMessage);
    document.getElementById('message-input').addEventListener('keypress', function(e) {
        if (e.key === 'Enter') sendMessage();
    });
}

function connectEventStream() {
    eventSource = new EventSource('/api/messages/stream');
    eventSource.onopen = function() { updateStatus('connected'); };
    eventSource.onmessage = function(event) {
        var msg = JSON.parse(event.data);
        addMessageToLog(msg);
    };
    eventSource.onerror = function() {
        updateStatus('disconnected');
        setTimeout(connectEventStream, 3000);
    };
}

function updateStatus(status) {
    var indicator = document.getElementById('status');
    if (status === 'connected') {
        indicator.textContent = '● 已连接';
        indicator.className = 'status-indicator connected';
    } else {
        indicator.textContent = '● 已断开';
        indicator.className = 'status-indicator disconnected';
    }
}

function loadStatus() {
    fetch('/api/status').then(function(resp) { return resp.json(); }).then(function(data) {
        document.getElementById('memory').textContent = formatBytes(data.memory.heap_alloc);
        document.getElementById('goroutines').textContent = data.goroutines;
        document.getElementById('sessions').textContent = data.sessions.total_sessions;
    }).catch(function(err) { console.error('Failed to load status:', err); });
}

function loadConfig() {
    fetch('/api/config').then(function(resp) { return resp.json(); }).then(function(data) {
        var configHtml = '<div class="config-item"><span class="config-key">服务器端口:</span>' +
            '<span class="config-value">' + data.server.port + '</span></div>' +
            '<div class="config-item"><span class="config-key">LLM提供商:</span>' +
            '<span class="config-value">' + data.llm.provider + '</span></div>' +
            '<div class="config-item"><span class="config-key">模型:</span>' +
            '<span class="config-value">' + data.llm.model + '</span></div>' +
            '<div class="config-item"><span class="config-key">智能体数量:</span>' +
            '<span class="config-value">' + data.agents + '</span></div>';
        document.getElementById('config-info').innerHTML = configHtml;
    }).catch(function(err) { console.error('Failed to load config:', err); });
}

function loadAgents() {
    fetch('/api/agents').then(function(resp) { return resp.json(); }).then(function(data) {
        agents = data;
        var agentListHtml = agents.map(function(a) {
            return '<div class="agent-item"><div class="agent-name">' + a.name + '</div>' +
                '<div class="agent-model">' + a.model + '</div></div>';
        }).join('');
        document.getElementById('agent-list').innerHTML = agentListHtml || '<div class="agent-item">暂无智能体</div>';
        var select = document.getElementById('agent-select');
        select.innerHTML = '<option value="">默认智能体</option>';
        agents.forEach(function(a) {
            var option = document.createElement('option');
            option.value = a.id;
            option.textContent = a.name;
            select.appendChild(option);
        });
    }).catch(function(err) { console.error('Failed to load agents:', err); });
}

function sendMessage() {
    var input = document.getElementById('message-input');
    var btn = document.getElementById('send-btn');
    var agentSelect = document.getElementById('agent-select');
    var message = input.value.trim();
    if (!message) return;
    btn.disabled = true;
    input.value = '';
    fetch('/api/send', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ message: message, agent_id: agentSelect.value })
    }).then(function(resp) {
        if (!resp.ok) throw new Error('Failed to send message');
    }).catch(function(err) {
        console.error('Failed to send message:', err);
        addMessageToLog({ type: 'error', time: new Date().toLocaleTimeString(), content: '发送失败: ' + err.message });
    }).finally(function() { btn.disabled = false; });
}

function addMessageToLog(msg) {
    var log = document.getElementById('message-log');
    var item = document.createElement('div');
    item.className = 'message-item ' + msg.type;
    var header = document.createElement('div');
    header.className = 'message-header';
    var userIdText = msg.user_id ? '(' + msg.user_id + ')' : '';
    header.innerHTML = '<span>' + (msg.source || msg.type) + ' ' + userIdText + '</span><span>' + msg.time + '</span>';
    var content = document.createElement('div');
    content.className = 'message-content';
    content.textContent = msg.content;
    item.appendChild(header);
    item.appendChild(content);
    log.appendChild(item);
    log.scrollTop = log.scrollHeight;
}

function formatBytes(bytes) {
    if (bytes === 0) return '0 B';
    var k = 1024;
    var sizes = ['B', 'KB', 'MB', 'GB'];
    var i = Math.floor(Math.log(bytes) / Math.log(k));
    return parseFloat((bytes / Math.pow(k, i)).toFixed(2)) + ' ' + sizes[i];
}

init();
`

// formatBytes 格式化字节
func formatBytes(bytes uint64) string {
	if bytes == 0 {
		return "0 B"
	}
	const k = 1024
	sizes := []string{"B", "KB", "MB", "GB"}
	i := 0
	for bytes >= k && i < len(sizes)-1 {
		bytes /= k
		i++
	}
	return fmt.Sprintf("%d %s", bytes, sizes[i])
}

// parseInt 解析整数
func parseInt(s string, defaultVal int) int {
	v, err := strconv.Atoi(s)
	if err != nil {
		return defaultVal
	}
	return v
}

// StringSliceContains 检查字符串切片是否包含元素
func StringSliceContains(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}
