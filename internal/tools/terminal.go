package tools

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"os/exec"
	"runtime"
	"strings"
	"sync"
	"time"

	"github.com/HaohanHe/mujibot/internal/confirmation"
)

type TerminalSession struct {
	ID        string
	Cmd       *exec.Cmd
	Stdin     io.WriteCloser
	Stdout    io.Reader
	Stderr    io.Reader
	Output    strings.Builder
	StartTime time.Time
	Running   bool
	mu        sync.RWMutex
}

type TerminalTool struct {
	manager   *Manager
	sessions  map[string]*TerminalSession
	mu        sync.RWMutex
	confirmMgr *confirmation.ConfirmationManager
}

func NewTerminalTool(manager *Manager, confirmMgr *confirmation.ConfirmationManager) *TerminalTool {
	return &TerminalTool{
		manager:    manager,
		sessions:   make(map[string]*TerminalSession),
		confirmMgr: confirmMgr,
	}
}

func (t *TerminalTool) Name() string {
	return "terminal"
}

func (t *TerminalTool) Description() string {
	return "执行终端命令并获取实时输出。支持交互式会话、后台运行、命令取消。"
}

func (t *TerminalTool) Parameters() map[string]interface{} {
	return map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"command": map[string]interface{}{
				"type":        "string",
				"description": "要执行的命令",
			},
			"action": map[string]interface{}{
				"type":        "string",
				"description": "操作类型: run(执行), cancel(取消), list(列出会话)",
				"enum":        []string{"run", "cancel", "list", "output"},
			},
			"sessionId": map[string]interface{}{
				"type":        "string",
				"description": "会话ID（用于cancel/output操作）",
			},
			"timeout": map[string]interface{}{
				"type":        "number",
				"description": "超时时间（秒），默认30",
			},
			"background": map[string]interface{}{
				"type":        "boolean",
				"description": "是否后台运行",
			},
		},
		"required": []string{"action"},
	}
}

func (t *TerminalTool) Execute(args map[string]interface{}) (string, error) {
	action, _ := args["action"].(string)

	switch action {
	case "list":
		return t.listSessions()
	case "cancel":
		sessionID, _ := args["sessionId"].(string)
		return t.cancelSession(sessionID)
	case "output":
		sessionID, _ := args["sessionId"].(string)
		return t.getSessionOutput(sessionID)
	case "run":
		command, _ := args["command"].(string)
		if command == "" {
			return "", fmt.Errorf("command is required for run action")
		}
		timeout := 30
		if t, ok := args["timeout"].(float64); ok {
			timeout = int(t)
		}
		background := false
		if b, ok := args["background"].(bool); ok {
			background = b
		}
		return t.runCommand(command, timeout, background)
	default:
		return "", fmt.Errorf("unknown action: %s", action)
	}
}

func (t *TerminalTool) runCommand(command string, timeout int, background bool) (string, error) {
	cfg := t.manager.GetConfig()
	if !cfg.TerminalEnabled {
		return "", fmt.Errorf("terminal is disabled in config")
	}

	var blockedCommand string
	for _, blocked := range cfg.BlockedCommands {
		if strings.Contains(command, blocked) {
			blockedCommand = blocked
			break
		}
	}

	isDangerous := confirmation.IsDangerousOperation(command)
	needsConfirmation := false
	confirmationDetails := ""

	if blockedCommand != "" {
		needsConfirmation = true
		confirmationDetails = fmt.Sprintf("命令包含黑名单命令: %s，需要用户确认", blockedCommand)
	} else if isDangerous {
		needsConfirmation = true
		confirmationDetails = "危险命令需要用户确认"
	}

	if needsConfirmation {
		if cfg.ConfirmDangerous && !cfg.UnattendedMode {
			riskLevel := "high"
			if blockedCommand != "" {
				riskLevel = "critical"
			}
			approved, err := t.confirmMgr.RequestConfirmation(
				context.Background(),
				"terminal",
				command,
				confirmationDetails,
				riskLevel,
			)
			if err != nil {
				return "", fmt.Errorf("confirmation failed: %w", err)
			}
			if !approved {
				return "", fmt.Errorf("operation rejected by user")
			}
		}
	}

	sessionID := fmt.Sprintf("term_%d", time.Now().UnixNano())

	var cmd *exec.Cmd
	if runtime.GOOS == "windows" {
		cmd = exec.Command("cmd", "/c", command)
	} else {
		cmd = exec.Command("sh", "-c", command)
	}

	cmd.Dir = cfg.WorkDir

	stdin, err := cmd.StdinPipe()
	if err != nil {
		return "", fmt.Errorf("failed to create stdin pipe: %w", err)
	}

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return "", fmt.Errorf("failed to create stdout pipe: %w", err)
	}

	stderr, err := cmd.StderrPipe()
	if err != nil {
		return "", fmt.Errorf("failed to create stderr pipe: %w", err)
	}

	session := &TerminalSession{
		ID:        sessionID,
		Cmd:       cmd,
		Stdin:     stdin,
		Stdout:    stdout,
		Stderr:    stderr,
		StartTime: time.Now(),
		Running:   true,
	}

	t.mu.Lock()
	t.sessions[sessionID] = session
	t.mu.Unlock()

	if err := cmd.Start(); err != nil {
		t.mu.Lock()
		delete(t.sessions, sessionID)
		t.mu.Unlock()
		return "", fmt.Errorf("failed to start command: %w", err)
	}

	if background {
		go t.monitorSession(session)
		return fmt.Sprintf("Session started: %s\nUse 'output' action with sessionId to get output.", sessionID), nil
	}

	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(timeout)*time.Second)
	defer cancel()

	done := make(chan error, 1)
	go func() {
		scanner := bufio.NewScanner(io.MultiReader(stdout, stderr))
		for scanner.Scan() {
			session.mu.Lock()
			session.Output.WriteString(scanner.Text() + "\n")
			session.mu.Unlock()
		}
		done <- cmd.Wait()
	}()

	select {
	case <-ctx.Done():
		cmd.Process.Kill()
		session.mu.Lock()
		session.Running = false
		session.mu.Unlock()
		return session.Output.String() + "\n[TIMEOUT]", nil
	case err := <-done:
		session.mu.Lock()
		session.Running = false
		session.mu.Unlock()
		output := session.Output.String()
		if err != nil {
			output += fmt.Sprintf("\n[EXIT ERROR: %v]", err)
		}
		t.mu.Lock()
		delete(t.sessions, sessionID)
		t.mu.Unlock()
		return output, nil
	}
}

func (t *TerminalTool) monitorSession(session *TerminalSession) {
	scanner := bufio.NewScanner(io.MultiReader(session.Stdout, session.Stderr))
	for scanner.Scan() {
		session.mu.Lock()
		session.Output.WriteString(scanner.Text() + "\n")
		session.mu.Unlock()
	}

	session.Cmd.Wait()
	session.mu.Lock()
	session.Running = false
	session.mu.Unlock()
}

func (t *TerminalTool) cancelSession(sessionID string) (string, error) {
	t.mu.Lock()
	defer t.mu.Unlock()

	session, ok := t.sessions[sessionID]
	if !ok {
		return "", fmt.Errorf("session not found: %s", sessionID)
	}

	if !session.Running {
		return "Session already completed", nil
	}

	if session.Cmd.Process != nil {
		session.Cmd.Process.Kill()
	}

	session.Running = false
	output := session.Output.String()
	delete(t.sessions, sessionID)

	return output + "\n[SESSION CANCELLED]", nil
}

func (t *TerminalTool) getSessionOutput(sessionID string) (string, error) {
	t.mu.RLock()
	defer t.mu.RUnlock()

	session, ok := t.sessions[sessionID]
	if !ok {
		return "", fmt.Errorf("session not found: %s", sessionID)
	}

	session.mu.RLock()
	defer session.mu.RUnlock()

	status := "running"
	if !session.Running {
		status = "completed"
	}

	return fmt.Sprintf("Status: %s\nDuration: %s\nOutput:\n%s",
		status,
		time.Since(session.StartTime).Round(time.Second),
		session.Output.String()), nil
}

func (t *TerminalTool) listSessions() (string, error) {
	t.mu.RLock()
	defer t.mu.RUnlock()

	if len(t.sessions) == 0 {
		return "No active sessions", nil
	}

	var sb strings.Builder
	sb.WriteString("Active sessions:\n")
	for id, session := range t.sessions {
		session.mu.RLock()
		status := "running"
		if !session.Running {
			status = "completed"
		}
		sb.WriteString(fmt.Sprintf("- %s: %s (started %s ago)\n",
			id,
			status,
			time.Since(session.StartTime).Round(time.Second)))
		session.mu.RUnlock()
	}
	return sb.String(), nil
}

func (t *TerminalTool) Cleanup() {
	t.mu.Lock()
	defer t.mu.Unlock()

	for _, session := range t.sessions {
		if session.Running && session.Cmd.Process != nil {
			session.Cmd.Process.Kill()
		}
	}
	t.sessions = make(map[string]*TerminalSession)
}

func (t *TerminalTool) SendInput(sessionID, input string) error {
	t.mu.RLock()
	session, ok := t.sessions[sessionID]
	t.mu.RUnlock()

	if !ok {
		return fmt.Errorf("session not found: %s", sessionID)
	}

	if !session.Running {
		return fmt.Errorf("session not running")
	}

	_, err := session.Stdin.Write([]byte(input + "\n"))
	return err
}
