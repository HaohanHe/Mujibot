package agent

import (
	"encoding/json"
	"fmt"
	"runtime/debug"
	"strings"
	"sync"

	"github.com/HaohanHe/mujibot/internal/config"
	"github.com/HaohanHe/mujibot/internal/i18n"
	"github.com/HaohanHe/mujibot/internal/llm"
	"github.com/HaohanHe/mujibot/internal/logger"
	"github.com/HaohanHe/mujibot/internal/memory"
	"github.com/HaohanHe/mujibot/internal/session"
	"github.com/HaohanHe/mujibot/internal/system"
	"github.com/HaohanHe/mujibot/internal/tools"
)

// Agent 智能体实例
type Agent struct {
	ID           string
	Name         string
	SystemPrompt string
	Provider     llm.Provider
	ToolManager  *tools.Manager
	SessionMgr   *session.Manager
	MemoryMgr    *memory.Manager
	Config       config.AgentConfig
	I18n         *i18n.I18n
	log          *logger.Logger
}

// Router 智能体路由器
type Router struct {
	agents   map[string]*Agent
	defaultAgent string
	mu       sync.RWMutex
	log      *logger.Logger
}

// NewRouter 创建智能体路由器
func NewRouter(log *logger.Logger) *Router {
	return &Router{
		agents: make(map[string]*Agent),
		log:    log,
	}
}

// RegisterAgent 注册智能体
func (r *Router) RegisterAgent(id string, agent *Agent) {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.agents[id] = agent
	if r.defaultAgent == "" {
		r.defaultAgent = id
	}

	r.log.Info("agent registered", "id", id, "name", agent.Name)
}

// GetAgent 获取智能体
func (r *Router) GetAgent(id string) (*Agent, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	agent, ok := r.agents[id]
	return agent, ok
}

// GetDefaultAgent 获取默认智能体
func (r *Router) GetDefaultAgent() (*Agent, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	agent, ok := r.agents[r.defaultAgent]
	return agent, ok
}

// Route 路由消息到智能体
func (r *Router) Route(userID, channel, agentID string) (*Agent, error) {
	// 如果指定了智能体ID，使用指定的
	if agentID != "" {
		if agent, ok := r.GetAgent(agentID); ok {
			return agent, nil
		}
		return nil, fmt.Errorf("agent not found: %s", agentID)
	}

	// 使用默认智能体
	if agent, ok := r.GetDefaultAgent(); ok {
		return agent, nil
	}

	return nil, fmt.Errorf("no agent available")
}

// GetAllAgents 获取所有智能体
func (r *Router) GetAllAgents() map[string]*Agent {
	r.mu.RLock()
	defer r.mu.RUnlock()

	result := make(map[string]*Agent, len(r.agents))
	for k, v := range r.agents {
		result[k] = v
	}
	return result
}

// ProcessMessage 处理消息（带panic恢复）
func (r *Router) ProcessMessage(agent *Agent, userID, channel, content string) (string, error) {
	defer func() {
		if rec := recover(); rec != nil {
			r.log.Error("agent panic recovered", "error", rec, "stack", string(debug.Stack()))
		}
	}()

	return agent.ProcessMessage(userID, channel, content)
}

// ProcessMessageStream 流式处理消息
func (r *Router) ProcessMessageStream(agent *Agent, userID, channel, content string, callback func(chunk string)) (string, error) {
	defer func() {
		if rec := recover(); rec != nil {
			r.log.Error("agent panic recovered", "error", rec, "stack", string(debug.Stack()))
		}
	}()

	return agent.ProcessMessageStream(userID, channel, content, callback)
}

// ProcessMessage 处理消息
func (a *Agent) ProcessMessage(userID, channel, content string) (string, error) {
	// 获取或创建会话
	sess := a.SessionMgr.GetOrCreate(userID, channel, a.ID)

	// 添加用户消息
	a.SessionMgr.AddMessage(sess, "user", content)

	// 构建消息历史
	messages := a.buildMessages(sess)

	// 获取工具定义
	toolDefs := a.ToolManager.GetToolDefinitions()
	tools := make([]llm.Tool, 0, len(toolDefs))
	for _, def := range toolDefs {
		fn, ok := def["function"].(map[string]interface{})
		if !ok {
			continue
		}

		name, _ := fn["name"].(string)
		desc, _ := fn["description"].(string)
		params, _ := fn["parameters"].(map[string]interface{})

		if name == "" {
			continue
		}

		tools = append(tools, llm.Tool{
			Type: "function",
			Function: llm.Function{
				Name:        name,
				Description: desc,
				Parameters:  params,
			},
		})
	}

	// 调用LLM
	resp, err := a.Provider.Chat(messages, tools)
	if err != nil {
		return "", fmt.Errorf("llm error: %w", err)
	}

	// 处理工具调用
	if len(resp.ToolCalls) > 0 {
		// 添加助手消息（带工具调用）
		a.SessionMgr.AddToolCallMessage(sess, "assistant", resp.Content, resp.ToolCalls)

		// 执行工具
		for _, tc := range resp.ToolCalls {
			result, err := a.executeToolCall(tc)
			if err != nil {
				result = fmt.Sprintf("Error: %v", err)
			}

			// 添加工具结果
			toolResult := fmt.Sprintf("Tool: %s\nResult: %s", tc.Function.Name, result)
			a.SessionMgr.AddMessage(sess, "tool", toolResult)
		}

		// 再次调用LLM获取最终响应
		messages = a.buildMessages(sess)
		resp, err = a.Provider.Chat(messages, nil)
		if err != nil {
			return "", fmt.Errorf("llm error: %w", err)
		}
	}

	// 添加助手响应
	a.SessionMgr.AddMessage(sess, "assistant", resp.Content)

	return resp.Content, nil
}

// ProcessMessageStream 流式处理消息
func (a *Agent) ProcessMessageStream(userID, channel, content string, callback func(chunk string)) (string, error) {
	// 获取或创建会话
	sess := a.SessionMgr.GetOrCreate(userID, channel, a.ID)

	// 添加用户消息
	a.SessionMgr.AddMessage(sess, "user", content)

	// 构建消息历史
	messages := a.buildMessages(sess)

	// 获取工具定义
	toolDefs := a.ToolManager.GetToolDefinitions()
	tools := make([]llm.Tool, len(toolDefs))
	for i, def := range toolDefs {
		tools[i] = llm.Tool{
			Type: "function",
			Function: llm.Function{
				Name:        def["function"].(map[string]interface{})["name"].(string),
				Description: def["function"].(map[string]interface{})["description"].(string),
				Parameters:  def["function"].(map[string]interface{})["parameters"].(map[string]interface{}),
			},
		}
	}

	// 调用LLM（流式）
	var fullContent string
	resp, err := a.Provider.ChatStream(messages, tools, func(chunk string) {
		fullContent += chunk
		if callback != nil {
			callback(chunk)
		}
	})
	if err != nil {
		return "", fmt.Errorf("llm error: %w", err)
	}

	// 处理工具调用
	if len(resp.ToolCalls) > 0 {
		// 添加助手消息（带工具调用）
		a.SessionMgr.AddToolCallMessage(sess, "assistant", fullContent, resp.ToolCalls)

		// 执行工具
		for _, tc := range resp.ToolCalls {
			result, err := a.executeToolCall(tc)
			if err != nil {
				result = fmt.Sprintf("Error: %v", err)
			}

			// 添加工具结果
			toolResult := fmt.Sprintf("Tool: %s\nResult: %s", tc.Function.Name, result)
			a.SessionMgr.AddMessage(sess, "tool", toolResult)
		}

		// 再次调用LLM获取最终响应
		messages = a.buildMessages(sess)
		fullContent = ""
		resp, err = a.Provider.ChatStream(messages, nil, func(chunk string) {
			fullContent += chunk
			if callback != nil {
				callback(chunk)
			}
		})
		if err != nil {
			return "", fmt.Errorf("llm error: %w", err)
		}
	}

	// 添加助手响应
	a.SessionMgr.AddMessage(sess, "assistant", fullContent)

	return fullContent, nil
}

// buildMessages 构建消息列表
func (a *Agent) buildMessages(sess *session.Session) []session.Message {
	messages := make([]session.Message, 0)

	// 添加系统提示
	if a.SystemPrompt != "" {
		systemContent := a.buildSystemPrompt()

		messages = append(messages, session.Message{
			Role:    "system",
			Content: systemContent,
		})
	}

	// 添加会话历史
	sessionMessages := a.SessionMgr.GetMessages(sess)
	messages = append(messages, sessionMessages...)

	return messages
}

// buildSystemPrompt 构建完整的系统提示词
func (a *Agent) buildSystemPrompt() string {
	var sb strings.Builder

	sb.WriteString(a.SystemPrompt)

	sb.WriteString("\n\n## 环境信息\n\n")
	sb.WriteString(fmt.Sprintf("- %s: %s\n", a.t("currentTime"), system.GetCurrentTime()))
	sb.WriteString(fmt.Sprintf("- %s: %s\n", a.t("timezone"), system.GetTimezone()))
	sb.WriteString(fmt.Sprintf("- %s: Mujibot AI Assistant\n", a.t("systemType")))

	sysInfo := system.GetInfo()
	sb.WriteString(sysInfo.Format())

	sb.WriteString(fmt.Sprintf("\n## %s\n\n", a.t("availableTools")))
	sb.WriteString(a.t("toolsIntro") + "\n")

	toolDefs := a.ToolManager.GetToolDefinitions()
	for _, tool := range toolDefs {
		sb.WriteString(fmt.Sprintf("- **%s**: %s\n", tool["name"], tool["description"]))
	}

	sb.WriteString("\n" + a.t("toolUsage") + "\n")

	if a.MemoryMgr != nil && a.MemoryMgr.IsEnabled() {
		memoryContext := a.MemoryMgr.GetMemoryContext()
		if memoryContext != "" {
			sb.WriteString(fmt.Sprintf("\n## %s\n\n", a.t("memoryContext")))
			sb.WriteString(memoryContext)
		}
	}

	sb.WriteString("\n## " + a.t("userLanguage") + "\n\n")
	sb.WriteString(a.t("replyInSameLang") + "\n")

	sb.WriteString("\n## " + a.t("memoryRulesTitle") + "\n\n")
	sb.WriteString(a.t("memoryRules") + "\n")
	sb.WriteString("\n" + a.t("memoryCategories") + "\n")

	return sb.String()
}

func (a *Agent) t(key string) string {
	if a.I18n == nil {
		a.I18n = i18n.New("en-US")
	}
	return a.I18n.T(key)
}

// executeToolCall 执行工具调用
func (a *Agent) executeToolCall(tc session.ToolCall) (string, error) {
	// 解析参数
	var args map[string]interface{}
	if err := json.Unmarshal([]byte(tc.Function.Arguments), &args); err != nil {
		return "", fmt.Errorf("failed to parse tool arguments: %w", err)
	}

	// 执行工具
	return a.ToolManager.Execute(tc.Function.Name, args)
}

// CreateAgent 创建智能体实例
func CreateAgent(id string, cfg config.AgentConfig, provider llm.Provider, toolMgr *tools.Manager, sessionMgr *session.Manager, memoryMgr *memory.Manager, i *i18n.I18n, log *logger.Logger) *Agent {
	return &Agent{
		ID:           id,
		Name:         cfg.Name,
		SystemPrompt: cfg.SystemPrompt,
		Provider:     provider,
		ToolManager:  toolMgr,
		SessionMgr:   sessionMgr,
		MemoryMgr:    memoryMgr,
		Config:       cfg,
		I18n:         i,
		log:          log,
	}
}
