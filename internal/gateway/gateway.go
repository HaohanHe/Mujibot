package gateway

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"runtime"
	"runtime/debug"
	"sync"
	"syscall"
	"time"

	"github.com/HaohanHe/mujibot/internal/agent"
	"github.com/HaohanHe/mujibot/internal/channel/discord"
	"github.com/HaohanHe/mujibot/internal/channel/feishu"
	"github.com/HaohanHe/mujibot/internal/channel/telegram"
	"github.com/HaohanHe/mujibot/internal/config"
	"github.com/HaohanHe/mujibot/internal/health"
	"github.com/HaohanHe/mujibot/internal/llm"
	"github.com/HaohanHe/mujibot/internal/logger"
	"github.com/HaohanHe/mujibot/internal/memory"
	"github.com/HaohanHe/mujibot/internal/session"
	"github.com/HaohanHe/mujibot/internal/tools"
	"github.com/HaohanHe/mujibot/internal/web"
)

// Gateway 网关
type Gateway struct {
	config      *config.Manager
	log         *logger.Logger
	sessionMgr  *session.Manager
	memoryMgr   *memory.Manager
	toolMgr     *tools.Manager
	llmProvider llm.Provider
	agentRouter *agent.Router
	healthCheck *health.Checker
	webServer   *web.Server

	// 渠道
	telegramBot *telegram.Bot
	discordBot  *discord.Bot
	feishuBot   *feishu.Bot

	// 控制
	ctx    context.Context
	cancel context.CancelFunc
	wg     sync.WaitGroup
	running bool
	mu     sync.RWMutex
}

// NewGateway 创建网关
func NewGateway(configPath string) (*Gateway, error) {
	// 创建临时日志记录器
	tempLog, err := logger.New(logger.Config{Level: "info", Format: "json"})
	if err != nil {
		return nil, fmt.Errorf("failed to create temp logger: %w", err)
	}

	// 加载配置
	cfg, err := config.NewManager(configPath, tempLog)
	if err != nil {
		return nil, fmt.Errorf("failed to load config: %w", err)
	}

	// 使用配置创建正式日志记录器
	logConfig := cfg.Get().Logging
	log, err := logger.New(logger.Config{
		Level:   logConfig.Level,
		File:    logConfig.File,
		MaxSize: logConfig.MaxSize,
		Format:  logConfig.Format,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create logger: %w", err)
	}

	// 更新配置管理器的日志
	cfg.Close()
	cfg, err = config.NewManager(configPath, log)
	if err != nil {
		return nil, err
	}

	g := &Gateway{
		config: cfg,
		log:    log,
	}

	// 初始化组件
	if err := g.initComponents(); err != nil {
		return nil, err
	}

	return g, nil
}

// initComponents 初始化组件
func (g *Gateway) initComponents() error {
	cfg := g.config.Get()

	// 创建会话管理器
	g.sessionMgr = session.NewManager(
		cfg.Session.MaxMessages,
		cfg.Session.IdleTimeout,
		cfg.Session.MaxSessions,
		g.log,
	)

	// 创建记忆管理器
	memCfg := memory.Config{
		Enabled:     cfg.Memory.Enabled,
		MemoryDir:   cfg.Memory.MemoryDir,
		MaxFileSize: cfg.Memory.MaxFileSize,
	}
	memoryMgr, err := memory.NewManager(memCfg, g.log)
	if err != nil {
		return fmt.Errorf("failed to create memory manager: %w", err)
	}
	g.memoryMgr = memoryMgr

	// 创建工具管理器
	toolCfg := tools.Config{
		WorkDir:          cfg.Tools.WorkDir,
		Timeout:          cfg.Tools.Timeout,
		ConfirmDangerous: cfg.Tools.ConfirmDangerous,
		BlockedCommands:  cfg.Tools.BlockedCommands,
		MemoryMgr:        memoryMgr,
	}
	toolMgr, err := tools.NewManager(toolCfg, g.log)
	if err != nil {
		return fmt.Errorf("failed to create tool manager: %w", err)
	}
	g.toolMgr = toolMgr

	// 创建LLM提供商
	llmProvider, err := llm.NewProvider(
		cfg.LLM.Provider,
		cfg.LLM.APIKey,
		cfg.LLM.BaseURL,
		cfg.LLM.Model,
		cfg.LLM.Timeout,
		cfg.LLM.MaxRetries,
		g.log,
	)
	if err != nil {
		return fmt.Errorf("failed to create llm provider: %w", err)
	}
	g.llmProvider = llmProvider

	// 创建智能体路由器
	g.agentRouter = agent.NewRouter(g.log)

	// 注册智能体
	for agentID, agentCfg := range cfg.Agents {
		a := agent.CreateAgent(agentID, agentCfg, llmProvider, g.toolMgr, g.sessionMgr, g.memoryMgr, g.log)
		g.agentRouter.RegisterAgent(agentID, a)
	}

	// 创建健康检查器
	g.healthCheck = health.NewChecker(g.log)

	// 创建Web服务器
	g.webServer = web.NewServer(
		cfg.Server.Port,
		g.config,
		g.sessionMgr,
		g.agentRouter,
		g.healthCheck,
		g.log,
	)

	return nil
}

// Start 启动网关
func (g *Gateway) Start() error {
	g.mu.Lock()
	if g.running {
		g.mu.Unlock()
		return fmt.Errorf("gateway already running")
	}
	g.running = true
	g.ctx, g.cancel = context.WithCancel(context.Background())
	g.mu.Unlock()

	g.log.Info("gateway starting", "version", "1.0.0")

	cfg := g.config.Get()

	// 启动Web服务器
	if err := g.webServer.Start(); err != nil {
		return fmt.Errorf("failed to start web server: %w", err)
	}

	// 启动Telegram Bot
	if cfg.Channels.Telegram.Enabled {
		if err := g.startTelegram(); err != nil {
			g.log.Error("failed to start telegram", "error", err)
		}
	}

	// 启动Discord Bot
	if cfg.Channels.Discord.Enabled {
		if err := g.startDiscord(); err != nil {
			g.log.Error("failed to start discord", "error", err)
		}
	}

	// 启动飞书Bot
	if cfg.Channels.Feishu.Enabled {
		if err := g.startFeishu(); err != nil {
			g.log.Error("failed to start feishu", "error", err)
		} else {
			g.webServer.SetFeishuHandler(g.GetFeishuWebhookHandler())
		}
	}

	// 启动监控协程
	g.wg.Add(1)
	go g.monitorLoop()

	// 等待退出信号
	g.waitForShutdown()

	return nil
}

// Stop 停止网关
func (g *Gateway) Stop() {
	g.mu.Lock()
	if !g.running {
		g.mu.Unlock()
		return
	}
	g.running = false
	g.mu.Unlock()

	g.log.Info("gateway stopping")

	// 取消上下文
	if g.cancel != nil {
		g.cancel()
	}

	// 停止渠道
	if g.telegramBot != nil {
		g.telegramBot.Stop()
	}
	if g.discordBot != nil {
		g.discordBot.Stop()
	}
	if g.feishuBot != nil {
		g.feishuBot.Stop()
	}

	// 等待协程结束
	g.wg.Wait()

	// 关闭组件
	if g.log != nil {
		g.log.Close()
	}
	if g.config != nil {
		g.config.Close()
	}

	g.log.Info("gateway stopped")
}

// IsRunning 检查是否运行中
func (g *Gateway) IsRunning() bool {
	g.mu.RLock()
	defer g.mu.RUnlock()
	return g.running
}

// startTelegram 启动Telegram
func (g *Gateway) startTelegram() error {
	cfg := g.config.Get()
	g.telegramBot = telegram.NewBot(cfg.Channels.Telegram, g.log)

	// 注册消息处理器
	g.telegramBot.OnMessage(func(userID int64, username, text string, chatID int64) (string, error) {
		return g.handleMessage("telegram", fmt.Sprintf("%d", userID), username, text)
	})

	if err := g.telegramBot.Start(); err != nil {
		return err
	}

	g.log.Info("telegram bot started")
	return nil
}

// startDiscord 启动Discord
func (g *Gateway) startDiscord() error {
	cfg := g.config.Get()
	g.discordBot = discord.NewBot(cfg.Channels.Discord, g.log)

	// 注册消息处理器
	g.discordBot.OnMessage(func(userID, username, content, channelID string) (string, error) {
		return g.handleMessage("discord", userID, username, content)
	})

	if err := g.discordBot.Start(); err != nil {
		return err
	}

	g.log.Info("discord bot started")
	return nil
}

// startFeishu 启动飞书
func (g *Gateway) startFeishu() error {
	cfg := g.config.Get()
	g.feishuBot = feishu.NewBot(cfg.Channels.Feishu, g.log)

	g.feishuBot.OnMessage(func(userID, username, content string) (string, error) {
		return g.handleMessage("feishu", userID, username, content)
	})

	if err := g.feishuBot.Start(); err != nil {
		return err
	}

	g.log.Info("feishu bot started")
	return nil
}

// GetFeishuWebhookHandler 获取飞书Webhook处理器
func (g *Gateway) GetFeishuWebhookHandler() http.HandlerFunc {
	if g.feishuBot == nil {
		return func(w http.ResponseWriter, r *http.Request) {
			http.Error(w, "Feishu not enabled", http.StatusServiceUnavailable)
		}
	}
	return g.feishuBot.GetWebhookHandler()
}

// handleMessage 处理消息
func (g *Gateway) handleMessage(channel, userID, username, content string) (string, error) {
	defer func() {
		if r := recover(); r != nil {
			g.log.Error("message handler panic", "error", r, "stack", string(debug.Stack()))
		}
	}()

	g.log.Info("message received",
		"channel", channel,
		"user_id", userID,
		"username", username,
		"content", truncate(content, 100),
	)

	// 记录消息统计
	g.healthCheck.RecordMessage()

	// 记录调试消息
	g.webServer.LogMessage("user", channel, content, userID, channel)

	// 路由到智能体
	agent, err := g.agentRouter.Route(userID, channel, "")
	if err != nil {
		g.log.Error("failed to route message", "error", err)
		return "", err
	}

	// 处理消息
	response, err := g.agentRouter.ProcessMessage(agent, userID, channel, content)
	if err != nil {
		g.log.Error("failed to process message", "error", err)
		g.healthCheck.RecordLLMFailed()
		g.webServer.LogMessage("error", channel, err.Error(), userID, channel)
		return "", err
	}

	// 记录成功
	g.healthCheck.RecordLLMSuccess()
	g.webServer.LogMessage("assistant", channel, response, userID, channel)

	return response, nil
}

// monitorLoop 监控循环
func (g *Gateway) monitorLoop() {
	defer g.wg.Done()

	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-g.ctx.Done():
			return
		case <-ticker.C:
			g.checkHealth()
		}
	}
}

// checkHealth 检查健康状态
func (g *Gateway) checkHealth() {
	var m runtime.MemStats
	runtime.ReadMemStats(&m)

	// 检查内存使用
	heapMB := m.HeapAlloc / 1024 / 1024
	if heapMB > 80 {
		g.log.Warn("high memory usage, triggering GC", "heap_mb", heapMB)
		runtime.GC()
		debug.FreeOSMemory()
	}

	// 检查磁盘空间
	if g.checkDiskSpace() {
		g.log.Warn("low disk space detected")
	}
}

// checkDiskSpace 检查磁盘空间
func (g *Gateway) checkDiskSpace() bool {
	// 简化实现：在Windows上跳过磁盘检查
	// 实际部署时在Linux上运行，此代码不会执行
	return false
}

// waitForShutdown 等待关闭信号
func (g *Gateway) waitForShutdown() {
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)

	select {
	case sig := <-sigCh:
		g.log.Info("received signal", "signal", sig)
	case <-g.ctx.Done():
	}

	g.Stop()
}

// truncate 截断字符串
func truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "..."
}
