package memory

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"time"

	"github.com/HaohanHe/mujibot/internal/logger"
)

// Manager 记忆管理器
type Manager struct {
	memoryDir   string
	maxFileSize int
	log         *logger.Logger
}

// Config 记忆配置
type Config struct {
	Enabled     bool
	MemoryDir   string
	MaxFileSize int
}

// NewManager 创建记忆管理器
func NewManager(cfg Config, log *logger.Logger) (*Manager, error) {
	if !cfg.Enabled {
		return &Manager{
			memoryDir:   "",
			maxFileSize: cfg.MaxFileSize,
			log:         log,
		}, nil
	}

	// 创建记忆目录
	if err := os.MkdirAll(cfg.MemoryDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create memory directory: %w", err)
	}

	// 创建memory子目录
	dailyDir := filepath.Join(cfg.MemoryDir, "memory")
	if err := os.MkdirAll(dailyDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create daily memory directory: %w", err)
	}

	return &Manager{
		memoryDir:   cfg.MemoryDir,
		maxFileSize: cfg.MaxFileSize,
		log:         log,
	}, nil
}

// GetDailyNotes 获取每日笔记内容
func (m *Manager) GetDailyNotes(days int) string {
	if m.memoryDir == "" {
		return ""
	}

	var result strings.Builder
	now := time.Now()

	for i := 0; i < days; i++ {
		date := now.AddDate(0, 0, -i).Format("2006-01-02")
		content, err := m.ReadDailyNote(date)
		if err == nil && content != "" {
			if result.Len() > 0 {
				result.WriteString("\n\n---\n\n")
			}
			result.WriteString(fmt.Sprintf("## %s\n\n%s", date, content))
		}
	}

	return result.String()
}

// ReadDailyNote 读取指定日期的笔记
func (m *Manager) ReadDailyNote(date string) (string, error) {
	if m.memoryDir == "" {
		return "", nil
	}

	filePath := filepath.Join(m.memoryDir, "memory", date+".md")
	content, err := os.ReadFile(filePath)
	if err != nil {
		if os.IsNotExist(err) {
			return "", nil
		}
		return "", err
	}

	return string(content), nil
}

// WriteDailyNote 写入每日笔记
func (m *Manager) WriteDailyNote(date string, content string) error {
	if m.memoryDir == "" {
		return nil
	}

	filePath := filepath.Join(m.memoryDir, "memory", date+".md")

	// 检查文件大小
	if info, err := os.Stat(filePath); err == nil {
		if info.Size() > int64(m.maxFileSize) {
			return fmt.Errorf("daily note file too large (max %d bytes)", m.maxFileSize)
		}
	}

	// 追加内容
	f, err := os.OpenFile(filePath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return fmt.Errorf("failed to open daily note file: %w", err)
	}
	defer f.Close()

	// 添加时间戳
	timestamp := time.Now().Format("15:04:05")
	entry := fmt.Sprintf("\n### %s\n\n%s\n", timestamp, content)

	if _, err := f.WriteString(entry); err != nil {
		return fmt.Errorf("failed to write daily note: %w", err)
	}

	m.log.Info("daily note written", "date", date, "file", filePath)
	return nil
}

// ReadLongTermMemory 读取长期记忆
func (m *Manager) ReadLongTermMemory() (string, error) {
	if m.memoryDir == "" {
		return "", nil
	}

	filePath := filepath.Join(m.memoryDir, "MEMORY.md")
	content, err := os.ReadFile(filePath)
	if err != nil {
		if os.IsNotExist(err) {
			return "", nil
		}
		return "", err
	}

	return string(content), nil
}

// WriteLongTermMemory 写入长期记忆
func (m *Manager) WriteLongTermMemory(content string) error {
	if m.memoryDir == "" {
		return nil
	}

	filePath := filepath.Join(m.memoryDir, "MEMORY.md")

	// 检查文件大小
	if len(content) > m.maxFileSize {
		return fmt.Errorf("memory content too large (max %d bytes)", m.maxFileSize)
	}

	if err := os.WriteFile(filePath, []byte(content), 0644); err != nil {
		return fmt.Errorf("failed to write memory file: %w", err)
	}

	m.log.Info("long-term memory written", "file", filePath)
	return nil
}

// SearchMemory 搜索记忆内容
func (m *Manager) SearchMemory(keyword string) ([]string, error) {
	if m.memoryDir == "" {
		return nil, nil
	}

	var results []string
	keywordLower := strings.ToLower(keyword)

	// 搜索每日笔记
	dailyDir := filepath.Join(m.memoryDir, "memory")
	entries, err := os.ReadDir(dailyDir)
	if err == nil {
		for _, entry := range entries {
			if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".md") {
				continue
			}

			content, err := os.ReadFile(filepath.Join(dailyDir, entry.Name()))
			if err != nil {
				continue
			}

			if strings.Contains(strings.ToLower(string(content)), keywordLower) {
				date := strings.TrimSuffix(entry.Name(), ".md")
				results = append(results, fmt.Sprintf("[Daily Note %s]", date))
			}
		}
	}

	// 搜索长期记忆
	longTerm, err := m.ReadLongTermMemory()
	if err == nil && longTerm != "" {
		if strings.Contains(strings.ToLower(longTerm), keywordLower) {
			results = append(results, "[Long-term Memory]")
		}
	}

	return results, nil
}

// GetMemoryContext 获取记忆上下文（用于LLM提示）
func (m *Manager) GetMemoryContext() string {
	if m.memoryDir == "" {
		return ""
	}

	var context strings.Builder

	// 添加长期记忆
	longTerm, _ := m.ReadLongTermMemory()
	if longTerm != "" {
		context.WriteString("## Long-term Memory\n\n")
		context.WriteString(longTerm)
		context.WriteString("\n\n")
	}

	// 添加最近2天的笔记
	dailyNotes := m.GetDailyNotes(2)
	if dailyNotes != "" {
		context.WriteString("## Recent Daily Notes\n\n")
		context.WriteString(dailyNotes)
	}

	return context.String()
}

// AppendToLongTermMemory 追加内容到长期记忆
func (m *Manager) AppendToLongTermMemory(content string) error {
	if m.memoryDir == "" {
		return nil
	}

	existing, _ := m.ReadLongTermMemory()

	var newContent strings.Builder
	if existing != "" {
		newContent.WriteString(existing)
		newContent.WriteString("\n\n")
	}

	// 添加时间戳
	timestamp := time.Now().Format("2006-01-02 15:04:05")
	newContent.WriteString(fmt.Sprintf("<!-- %s -->\n", timestamp))
	newContent.WriteString(content)

	return m.WriteLongTermMemory(newContent.String())
}

// ListDailyNotes 列出所有每日笔记
func (m *Manager) ListDailyNotes() ([]string, error) {
	if m.memoryDir == "" {
		return nil, nil
	}

	dailyDir := filepath.Join(m.memoryDir, "memory")
	entries, err := os.ReadDir(dailyDir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}

	var dates []string
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		name := entry.Name()
		if strings.HasSuffix(name, ".md") {
			date := strings.TrimSuffix(name, ".md")
			// 验证日期格式
			if matched, _ := regexp.MatchString(`^\d{4}-\d{2}-\d{2}$`, date); matched {
				dates = append(dates, date)
			}
		}
	}

	// 按日期排序（最新的在前）
	sort.Sort(sort.Reverse(sort.StringSlice(dates)))

	return dates, nil
}

// CleanOldNotes 清理旧笔记（保留最近N天）
func (m *Manager) CleanOldNotes(keepDays int) error {
	if m.memoryDir == "" {
		return nil
	}

	dates, err := m.ListDailyNotes()
	if err != nil {
		return err
	}

	if len(dates) <= keepDays {
		return nil
	}

	// 删除旧笔记
	for _, date := range dates[keepDays:] {
		filePath := filepath.Join(m.memoryDir, "memory", date+".md")
		if err := os.Remove(filePath); err != nil {
			m.log.Warn("failed to remove old note", "date", date, "error", err)
		} else {
			m.log.Info("old note removed", "date", date)
		}
	}

	return nil
}

// IsEnabled 检查记忆功能是否启用
func (m *Manager) IsEnabled() bool {
	return m.memoryDir != ""
}
