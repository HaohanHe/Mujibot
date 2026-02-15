package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/HaohanHe/mujibot/internal/logger"
	"github.com/HaohanHe/mujibot/internal/memory"
)

// Tool 工具接口
type Tool interface {
	Name() string
	Description() string
	Parameters() map[string]interface{}
	Execute(args map[string]interface{}) (string, error)
}

// Manager 工具管理器
type Manager struct {
	tools            map[string]Tool
	workDir          string
	timeout          time.Duration
	confirmDangerous bool
	blockedCommands  []string
	memoryMgr        *memory.Manager
	log              *logger.Logger
}

// Config 工具配置
type Config struct {
	WorkDir          string
	Timeout          int
	ConfirmDangerous bool
	AllowedCommands  []string
	BlockedCommands  []string
	MemoryMgr        *memory.Manager
}

// NewManager 创建工具管理器
func NewManager(cfg Config, log *logger.Logger) (*Manager, error) {
	// 创建工作目录
	if err := os.MkdirAll(cfg.WorkDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create work directory: %w", err)
	}

	m := &Manager{
		tools:            make(map[string]Tool),
		workDir:          cfg.WorkDir,
		timeout:          time.Duration(cfg.Timeout) * time.Second,
		confirmDangerous: cfg.ConfirmDangerous,
		blockedCommands:  cfg.BlockedCommands,
		memoryMgr:        cfg.MemoryMgr,
		log:              log,
	}

	// 注册内置工具
	m.registerBuiltinTools()

	return m, nil
}

// Register 注册工具
func (m *Manager) Register(tool Tool) {
	m.tools[tool.Name()] = tool
	m.log.Info("tool registered", "name", tool.Name())
}

// Get 获取工具
func (m *Manager) Get(name string) (Tool, bool) {
	tool, ok := m.tools[name]
	return tool, ok
}

// GetAll 获取所有工具
func (m *Manager) GetAll() []Tool {
	result := make([]Tool, 0, len(m.tools))
	for _, tool := range m.tools {
		result = append(result, tool)
	}
	return result
}

// Execute 执行工具
func (m *Manager) Execute(name string, args map[string]interface{}) (string, error) {
	tool, ok := m.tools[name]
	if !ok {
		return "", fmt.Errorf("tool not found: %s", name)
	}

	m.log.Info("executing tool", "name", name, "args", args)

	result, err := tool.Execute(args)
	if err != nil {
		m.log.Error("tool execution failed", "name", name, "error", err)
		return "", err
	}

	m.log.Info("tool executed successfully", "name", name)
	return result, nil
}

// GetToolDefinitions 获取工具定义（用于LLM）
func (m *Manager) GetToolDefinitions() []map[string]interface{} {
	defs := make([]map[string]interface{}, 0, len(m.tools))
	for _, tool := range m.tools {
		defs = append(defs, map[string]interface{}{
			"type": "function",
			"function": map[string]interface{}{
				"name":        tool.Name(),
				"description": tool.Description(),
				"parameters":  tool.Parameters(),
			},
		})
	}
	return defs
}

// registerBuiltinTools 注册内置工具
func (m *Manager) registerBuiltinTools() {
	m.Register(&ReadFileTool{manager: m})
	m.Register(&WriteFileTool{manager: m})
	m.Register(&ListDirectoryTool{manager: m})
	m.Register(&ExecuteCommandTool{manager: m})
	m.Register(&GetSystemInfoTool{manager: m})
	m.Register(&ApplyPatchTool{manager: m})
	m.Register(&WebSearchTool{manager: m})
	m.Register(&HTTPRequestTool{manager: m})
	m.Register(&GrepTool{manager: m})
	m.Register(&MemoryReadTool{manager: m})
	m.Register(&MemoryWriteTool{manager: m})
}

// sanitizePath 清理路径，确保在工作目录内
func (m *Manager) sanitizePath(path string) (string, error) {
	// 转换为绝对路径
	if !filepath.IsAbs(path) {
		path = filepath.Join(m.workDir, path)
	}

	// 清理路径
	path = filepath.Clean(path)

	// 检查是否在工作目录内
	if !strings.HasPrefix(path, m.workDir) {
		return "", fmt.Errorf("path is outside work directory: %s", path)
	}

	return path, nil
}

// isDangerousCommand 检查是否为危险命令
func (m *Manager) isDangerousCommand(cmd string) bool {
	dangerousPatterns := []string{
		`\brm\s+-rf\b`,
		`\bdd\s+`,
		`\bmkfs\b`,
		`\bchmod\s+777\b`,
		`\b>\s*/dev/`,
		`\b:\(\)\{\s*:\|\:&\};`,
	}

	for _, pattern := range dangerousPatterns {
		if matched, _ := regexp.MatchString(pattern, cmd); matched {
			return true
		}
	}

	return false
}

// isBlockedCommand 检查是否为禁止命令
func (m *Manager) isBlockedCommand(cmd string) bool {
	parts := strings.Fields(cmd)
	if len(parts) == 0 {
		return false
	}

	baseCmd := filepath.Base(parts[0])
	for _, blocked := range m.blockedCommands {
		if baseCmd == blocked {
			return true
		}
	}

	return false
}

// ReadFileTool 读取文件工具
type ReadFileTool struct {
	manager *Manager
}

func (t *ReadFileTool) Name() string {
	return "read_file"
}

func (t *ReadFileTool) Description() string {
	return "读取文件内容。支持文本文件，限制1MB以内。"
}

func (t *ReadFileTool) Parameters() map[string]interface{} {
	return map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"path": map[string]interface{}{
				"type":        "string",
				"description": "文件路径（相对workDir或绝对路径）",
			},
		},
		"required": []string{"path"},
	}
}

func (t *ReadFileTool) Execute(args map[string]interface{}) (string, error) {
	path, ok := args["path"].(string)
	if !ok {
		return "", fmt.Errorf("path is required")
	}

	safePath, err := t.manager.sanitizePath(path)
	if err != nil {
		return "", err
	}

	// 检查文件大小
	info, err := os.Stat(safePath)
	if err != nil {
		return "", fmt.Errorf("failed to stat file: %w", err)
	}

	if info.Size() > 1024*1024 {
		return "", fmt.Errorf("file too large (max 1MB)")
	}

	content, err := os.ReadFile(safePath)
	if err != nil {
		return "", fmt.Errorf("failed to read file: %w", err)
	}

	return string(content), nil
}

// WriteFileTool 写入文件工具
type WriteFileTool struct {
	manager *Manager
}

func (t *WriteFileTool) Name() string {
	return "write_file"
}

func (t *WriteFileTool) Description() string {
	return "写入内容到文件。如果文件不存在则创建，存在则覆盖。"
}

func (t *WriteFileTool) Parameters() map[string]interface{} {
	return map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"path": map[string]interface{}{
				"type":        "string",
				"description": "文件路径（相对workDir或绝对路径）",
			},
			"content": map[string]interface{}{
				"type":        "string",
				"description": "要写入的内容",
			},
		},
		"required": []string{"path", "content"},
	}
}

func (t *WriteFileTool) Execute(args map[string]interface{}) (string, error) {
	path, ok := args["path"].(string)
	if !ok {
		return "", fmt.Errorf("path is required")
	}

	content, ok := args["content"].(string)
	if !ok {
		return "", fmt.Errorf("content is required")
	}

	safePath, err := t.manager.sanitizePath(path)
	if err != nil {
		return "", err
	}

	// 确保目录存在
	dir := filepath.Dir(safePath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return "", fmt.Errorf("failed to create directory: %w", err)
	}

	if err := os.WriteFile(safePath, []byte(content), 0644); err != nil {
		return "", fmt.Errorf("failed to write file: %w", err)
	}

	return fmt.Sprintf("File written successfully: %s", safePath), nil
}

// ListDirectoryTool 列出目录工具
type ListDirectoryTool struct {
	manager *Manager
}

func (t *ListDirectoryTool) Name() string {
	return "list_directory"
}

func (t *ListDirectoryTool) Description() string {
	return "列出目录中的文件和子目录。"
}

func (t *ListDirectoryTool) Parameters() map[string]interface{} {
	return map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"path": map[string]interface{}{
				"type":        "string",
				"description": "目录路径（相对workDir或绝对路径），默认为workDir",
			},
		},
	}
}

func (t *ListDirectoryTool) Execute(args map[string]interface{}) (string, error) {
	path := "."
	if p, ok := args["path"].(string); ok && p != "" {
		path = p
	}

	safePath, err := t.manager.sanitizePath(path)
	if err != nil {
		return "", err
	}

	entries, err := os.ReadDir(safePath)
	if err != nil {
		return "", fmt.Errorf("failed to read directory: %w", err)
	}

	var result strings.Builder
	for _, entry := range entries {
		prefix := "[FILE]"
		if entry.IsDir() {
			prefix = "[DIR]"
		}
		result.WriteString(fmt.Sprintf("%s %s\n", prefix, entry.Name()))
	}

	return result.String(), nil
}

// ExecuteCommandTool 执行命令工具
type ExecuteCommandTool struct {
	manager *Manager
}

func (t *ExecuteCommandTool) Name() string {
	return "execute_command"
}

func (t *ExecuteCommandTool) Description() string {
	return "执行shell命令并返回输出。危险命令需要确认。"
}

func (t *ExecuteCommandTool) Parameters() map[string]interface{} {
	return map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"command": map[string]interface{}{
				"type":        "string",
				"description": "要执行的命令",
			},
			"confirm": map[string]interface{}{
				"type":        "boolean",
				"description": "危险命令确认",
			},
		},
		"required": []string{"command"},
	}
}

func (t *ExecuteCommandTool) Execute(args map[string]interface{}) (string, error) {
	command, ok := args["command"].(string)
	if !ok {
		return "", fmt.Errorf("command is required")
	}

	// 检查禁止命令
	if t.manager.isBlockedCommand(command) {
		return "", fmt.Errorf("command is blocked for security reasons")
	}

	// 检查危险命令
	if t.manager.isDangerousCommand(command) {
		if t.manager.confirmDangerous {
			confirmed, _ := args["confirm"].(bool)
			if !confirmed {
				return "", fmt.Errorf("dangerous command detected, set confirm=true to execute")
			}
		}
	}

	ctx, cancel := context.WithTimeout(context.Background(), t.manager.timeout)
	defer cancel()

	cmd := exec.CommandContext(ctx, "sh", "-c", command)
	cmd.Dir = t.manager.workDir

	output, err := cmd.CombinedOutput()
	if ctx.Err() == context.DeadlineExceeded {
		return "", fmt.Errorf("command timed out after %v", t.manager.timeout)
	}

	result := string(output)
	if err != nil {
		return result, fmt.Errorf("command failed: %w", err)
	}

	return result, nil
}

// GetSystemInfoTool 获取系统信息工具
type GetSystemInfoTool struct {
	manager *Manager
}

func (t *GetSystemInfoTool) Name() string {
	return "get_system_info"
}

func (t *GetSystemInfoTool) Description() string {
	return "获取系统信息，包括内存使用、磁盘空间等。"
}

func (t *GetSystemInfoTool) Parameters() map[string]interface{} {
	return map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{},
	}
}

func (t *GetSystemInfoTool) Execute(args map[string]interface{}) (string, error) {
	info := make(map[string]interface{})

	// 内存信息
	memInfo, err := exec.Command("free", "-h").Output()
	if err == nil {
		info["memory"] = string(memInfo)
	}

	// 磁盘信息
	diskInfo, err := exec.Command("df", "-h").Output()
	if err == nil {
		info["disk"] = string(diskInfo)
	}

	// 负载信息
	loadInfo, err := exec.Command("uptime").Output()
	if err == nil {
		info["uptime"] = string(loadInfo)
	}

	// 工作目录
	info["work_dir"] = t.manager.workDir

	result, _ := json.MarshalIndent(info, "", "  ")
	return string(result), nil
}

// ApplyPatchTool 应用代码补丁工具
type ApplyPatchTool struct {
	manager *Manager
}

func (t *ApplyPatchTool) Name() string {
	return "apply_patch"
}

func (t *ApplyPatchTool) Description() string {
	return "应用代码补丁到文件。支持统一diff格式，可以精确修改文件内容。"
}

func (t *ApplyPatchTool) Parameters() map[string]interface{} {
	return map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"path": map[string]interface{}{
				"type":        "string",
				"description": "要修改的文件路径",
			},
			"old_string": map[string]interface{}{
				"type":        "string",
				"description": "要被替换的旧字符串（必须精确匹配）",
			},
			"new_string": map[string]interface{}{
				"type":        "string",
				"description": "用于替换的新字符串",
			},
		},
		"required": []string{"path", "old_string", "new_string"},
	}
}

func (t *ApplyPatchTool) Execute(args map[string]interface{}) (string, error) {
	path, ok := args["path"].(string)
	if !ok {
		return "", fmt.Errorf("path is required")
	}

	oldStr, ok := args["old_string"].(string)
	if !ok {
		return "", fmt.Errorf("old_string is required")
	}

	newStr, ok := args["new_string"].(string)
	if !ok {
		return "", fmt.Errorf("new_string is required")
	}

	safePath, err := t.manager.sanitizePath(path)
	if err != nil {
		return "", err
	}

	// 读取文件内容
	content, err := os.ReadFile(safePath)
	if err != nil {
		return "", fmt.Errorf("failed to read file: %w", err)
	}

	oldContent := string(content)

	// 检查old_string是否存在
	if !strings.Contains(oldContent, oldStr) {
		return "", fmt.Errorf("old_string not found in file")
	}

	// 替换内容
	newContent := strings.Replace(oldContent, oldStr, newStr, 1)

	// 写回文件
	if err := os.WriteFile(safePath, []byte(newContent), 0644); err != nil {
		return "", fmt.Errorf("failed to write file: %w", err)
	}

	return fmt.Sprintf("Patch applied successfully to %s", safePath), nil
}

// WebSearchTool 网页搜索工具
type WebSearchTool struct {
	manager *Manager
}

func (t *WebSearchTool) Name() string {
	return "web_search"
}

func (t *WebSearchTool) Description() string {
	return "使用DuckDuckGo搜索网页。返回搜索结果标题和链接。"
}

func (t *WebSearchTool) Parameters() map[string]interface{} {
	return map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"query": map[string]interface{}{
				"type":        "string",
				"description": "搜索查询",
			},
			"num_results": map[string]interface{}{
				"type":        "integer",
				"description": "返回结果数量（默认5，最大10）",
			},
		},
		"required": []string{"query"},
	}
}

func (t *WebSearchTool) Execute(args map[string]interface{}) (string, error) {
	query, ok := args["query"].(string)
	if !ok || query == "" {
		return "", fmt.Errorf("query is required")
	}

	numResults := 5
	if n, ok := args["num_results"].(float64); ok {
		numResults = int(n)
		if numResults > 10 {
			numResults = 10
		}
		if numResults < 1 {
			numResults = 5
		}
	}

	// 使用DuckDuckGo HTML版本搜索
	searchURL := fmt.Sprintf("https://html.duckduckgo.com/html/?q=%s", strings.ReplaceAll(query, " ", "+"))

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Get(searchURL)
	if err != nil {
		return "", fmt.Errorf("search request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("search returned status %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read response: %w", err)
	}

	// 简单解析HTML提取结果
	content := string(body)
	var results []map[string]string

	// 提取搜索结果
	re := regexp.MustCompile(`<a[^>]*class="result__a"[^>]*href="([^"]*)"[^>]*>(.*?)</a>`)
	matches := re.FindAllStringSubmatch(content, numResults)

	for _, match := range matches {
		if len(match) >= 3 {
			title := stripHTMLTags(match[2])
			link := match[1]
			// 处理DuckDuckGo重定向链接
			if strings.HasPrefix(link, "//") {
				link = "https:" + link
			}
			results = append(results, map[string]string{
				"title": title,
				"link":  link,
			})
		}
	}

	if len(results) == 0 {
		return "No search results found", nil
	}

	var output strings.Builder
	output.WriteString(fmt.Sprintf("Search results for: %s\n\n", query))
	for i, result := range results {
		output.WriteString(fmt.Sprintf("%d. %s\n   %s\n\n", i+1, result["title"], result["link"]))
	}

	return output.String(), nil
}

type HTTPRequestTool struct {
	manager *Manager
}

func (t *HTTPRequestTool) Name() string {
	return "http_request"
}

func (t *HTTPRequestTool) Description() string {
	return "发送HTTP请求获取网页内容。用于获取搜索结果的详细内容。"
}

func (t *HTTPRequestTool) Parameters() map[string]interface{} {
	return map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"url": map[string]interface{}{
				"type":        "string",
				"description": "请求的URL",
			},
			"method": map[string]interface{}{
				"type":        "string",
				"description": "HTTP方法（GET/POST，默认GET）",
			},
		},
		"required": []string{"url"},
	}
}

func (t *HTTPRequestTool) Execute(args map[string]interface{}) (string, error) {
	url, ok := args["url"].(string)
	if !ok || url == "" {
		return "", fmt.Errorf("url is required")
	}

	method := "GET"
	if m, ok := args["method"].(string); ok {
		method = strings.ToUpper(m)
	}

	client := &http.Client{Timeout: 15 * time.Second}
	var req *http.Request
	var err error

	if method == "POST" {
		req, err = http.NewRequest("POST", url, nil)
	} else {
		req, err = http.NewRequest("GET", url, nil)
	}
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("User-Agent", "Mozilla/5.0 (compatible; Mujibot/1.0)")

	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read response: %w", err)
	}

	content := string(body)

	content = stripHTMLTags(content)

	if len(content) > 5000 {
		content = content[:5000] + "\n... (truncated)"
	}

	content = strings.TrimSpace(content)
	if len(content) == 0 {
		return "Empty response", nil
	}

	return content, nil
}

type GrepTool struct {
	manager *Manager
}

func (t *GrepTool) Name() string {
	return "grep"
}

func (t *GrepTool) Description() string {
	return "在工作目录中搜索文件内容。支持正则表达式。"
}

func (t *GrepTool) Parameters() map[string]interface{} {
	return map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"pattern": map[string]interface{}{
				"type":        "string",
				"description": "搜索模式（正则表达式）",
			},
			"path": map[string]interface{}{
				"type":        "string",
				"description": "搜索路径（默认为workDir）",
			},
			"include": map[string]interface{}{
				"type":        "string",
				"description": "文件匹配模式（如 *.go）",
			},
		},
		"required": []string{"pattern"},
	}
}

func (t *GrepTool) Execute(args map[string]interface{}) (string, error) {
	pattern, ok := args["pattern"].(string)
	if !ok || pattern == "" {
		return "", fmt.Errorf("pattern is required")
	}

	searchPath := "."
	if p, ok := args["path"].(string); ok && p != "" {
		searchPath = p
	}

	include := "*"
	if i, ok := args["include"].(string); ok && i != "" {
		include = i
	}

	safePath, err := t.manager.sanitizePath(searchPath)
	if err != nil {
		return "", err
	}

	// 编译正则表达式
	re, err := regexp.Compile(pattern)
	if err != nil {
		return "", fmt.Errorf("invalid pattern: %w", err)
	}

	var matches []string
	var matchCount int

	// 遍历目录
	err = filepath.Walk(safePath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil // 跳过错误
		}

		if info.IsDir() {
			return nil
		}

		// 检查文件匹配模式
		matched, _ := filepath.Match(include, filepath.Base(path))
		if !matched {
			return nil
		}

		// 跳过二进制文件和大文件
		if info.Size() > 1024*1024 {
			return nil
		}

		content, err := os.ReadFile(path)
		if err != nil {
			return nil
		}

		lines := strings.Split(string(content), "\n")
		for i, line := range lines {
			if re.MatchString(line) {
				relPath, _ := filepath.Rel(t.manager.workDir, path)
				matches = append(matches, fmt.Sprintf("%s:%d: %s", relPath, i+1, strings.TrimSpace(line)))
				matchCount++
				if matchCount >= 50 { // 限制结果数量
					return filepath.SkipAll
				}
			}
		}

		return nil
	})

	if err != nil && err != filepath.SkipAll {
		return "", err
	}

	if len(matches) == 0 {
		return "No matches found", nil
	}

	return strings.Join(matches, "\n"), nil
}

// stripHTMLTags 去除HTML标签
func stripHTMLTags(html string) string {
	re := regexp.MustCompile(`<[^>]*>`)
	return re.ReplaceAllString(html, "")
}

// MemoryReadTool 读取记忆工具
type MemoryReadTool struct {
	manager *Manager
}

func (t *MemoryReadTool) Name() string {
	return "memory_read"
}

func (t *MemoryReadTool) Description() string {
	return "读取长期记忆或每日笔记。用于回顾之前保存的信息。"
}

func (t *MemoryReadTool) Parameters() map[string]interface{} {
	return map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"type": map[string]interface{}{
				"type":        "string",
				"description": "记忆类型: 'longterm' 或 'daily'",
				"enum":        []string{"longterm", "daily"},
			},
			"date": map[string]interface{}{
				"type":        "string",
				"description": "日期 (YYYY-MM-DD格式)，仅用于daily类型，默认为今天",
			},
		},
		"required": []string{"type"},
	}
}

func (t *MemoryReadTool) Execute(args map[string]interface{}) (string, error) {
	if t.manager.memoryMgr == nil || !t.manager.memoryMgr.IsEnabled() {
		return "", fmt.Errorf("memory feature is not enabled")
	}

	memType, ok := args["type"].(string)
	if !ok {
		return "", fmt.Errorf("type is required")
	}

	switch memType {
	case "longterm":
		content, err := t.manager.memoryMgr.ReadLongTermMemory()
		if err != nil {
			return "", fmt.Errorf("failed to read long-term memory: %w", err)
		}
		if content == "" {
			return "No long-term memory found", nil
		}
		return content, nil

	case "daily":
		date := time.Now().Format("2006-01-02")
		if d, ok := args["date"].(string); ok && d != "" {
			date = d
		}
		content, err := t.manager.memoryMgr.ReadDailyNote(date)
		if err != nil {
			return "", fmt.Errorf("failed to read daily note: %w", err)
		}
		if content == "" {
			return fmt.Sprintf("No daily note found for %s", date), nil
		}
		return content, nil

	default:
		return "", fmt.Errorf("invalid memory type: %s", memType)
	}
}

// MemoryWriteTool 写入记忆工具
type MemoryWriteTool struct {
	manager *Manager
}

func (t *MemoryWriteTool) Name() string {
	return "memory_write"
}

func (t *MemoryWriteTool) Description() string {
	return "写入长期记忆或每日笔记。用于保存重要信息供将来参考。"
}

func (t *MemoryWriteTool) Parameters() map[string]interface{} {
	return map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"type": map[string]interface{}{
				"type":        "string",
				"description": "记忆类型: 'longterm' 或 'daily'",
				"enum":        []string{"longterm", "daily"},
			},
			"content": map[string]interface{}{
				"type":        "string",
				"description": "要保存的内容",
			},
			"append": map[string]interface{}{
				"type":        "boolean",
				"description": "是否追加到现有内容（仅用于longterm），默认为true",
			},
		},
		"required": []string{"type", "content"},
	}
}

func (t *MemoryWriteTool) Execute(args map[string]interface{}) (string, error) {
	if t.manager.memoryMgr == nil || !t.manager.memoryMgr.IsEnabled() {
		return "", fmt.Errorf("memory feature is not enabled")
	}

	memType, ok := args["type"].(string)
	if !ok {
		return "", fmt.Errorf("type is required")
	}

	content, ok := args["content"].(string)
	if !ok || content == "" {
		return "", fmt.Errorf("content is required")
	}

	switch memType {
	case "longterm":
		append := true
		if a, ok := args["append"].(bool); ok {
			append = a
		}

		var err error
		if append {
			err = t.manager.memoryMgr.AppendToLongTermMemory(content)
		} else {
			err = t.manager.memoryMgr.WriteLongTermMemory(content)
		}

		if err != nil {
			return "", fmt.Errorf("failed to write long-term memory: %w", err)
		}
		return "Long-term memory updated successfully", nil

	case "daily":
		date := time.Now().Format("2006-01-02")
		if err := t.manager.memoryMgr.WriteDailyNote(date, content); err != nil {
			return "", fmt.Errorf("failed to write daily note: %w", err)
		}
		return fmt.Sprintf("Daily note for %s updated successfully", date), nil

	default:
		return "", fmt.Errorf("invalid memory type: %s", memType)
	}
}
