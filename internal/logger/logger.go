package logger

import (
	"compress/gzip"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sync"
	"time"
)

// Level 日志级别
type Level int

const (
	DEBUG Level = iota
	INFO
	WARN
	ERROR
)

func (l Level) String() string {
	switch l {
	case DEBUG:
		return "DEBUG"
	case INFO:
		return "INFO"
	case WARN:
		return "WARN"
	case ERROR:
		return "ERROR"
	default:
		return "UNKNOWN"
	}
}

// ParseLevel 解析日志级别
func ParseLevel(s string) Level {
	switch s {
	case "debug":
		return DEBUG
	case "info":
		return INFO
	case "warn":
		return WARN
	case "error":
		return ERROR
	default:
		return INFO
	}
}

// LogEntry 日志条目
type LogEntry struct {
	Time    string                 `json:"time"`
	Level   string                 `json:"level"`
	Message string                 `json:"message"`
	Fields  map[string]interface{} `json:"fields,omitempty"`
}

// Logger 日志记录器
type Logger struct {
	level      Level
	output     io.Writer
	file       *os.File
	filePath   string
	maxSize    int64
	format     string
	mu         sync.Mutex
	buffer     []LogEntry
	bufferSize int
	stopCh     chan struct{}
}

// Config 日志配置
type Config struct {
	Level   string
	File    string
	MaxSize int
	Format  string
}

// New 创建日志记录器
func New(cfg Config) (*Logger, error) {
	l := &Logger{
		level:      ParseLevel(cfg.Level),
		filePath:   cfg.File,
		maxSize:    int64(cfg.MaxSize) * 1024 * 1024,
		format:     cfg.Format,
		buffer:     make([]LogEntry, 0, 100),
		bufferSize: 100,
		stopCh:     make(chan struct{}),
	}

	if cfg.File != "" {
		if err := l.openFile(); err != nil {
			return nil, err
		}
	} else {
		l.output = os.Stdout
	}

	go l.flushLoop()

	return l, nil
}

// openFile 打开日志文件
func (l *Logger) openFile() error {
	// 确保目录存在
	dir := filepath.Dir(l.filePath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}

	file, err := os.OpenFile(l.filePath, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0644)
	if err != nil {
		return err
	}

	l.file = file
	l.output = file
	return nil
}

// Debug 记录调试日志
func (l *Logger) Debug(msg string, fields ...interface{}) {
	l.log(DEBUG, msg, fields...)
}

// Info 记录信息日志
func (l *Logger) Info(msg string, fields ...interface{}) {
	l.log(INFO, msg, fields...)
}

// Warn 记录警告日志
func (l *Logger) Warn(msg string, fields ...interface{}) {
	l.log(WARN, msg, fields...)
}

// Error 记录错误日志
func (l *Logger) Error(msg string, fields ...interface{}) {
	l.log(ERROR, msg, fields...)
}

// log 记录日志
func (l *Logger) log(level Level, msg string, fields ...interface{}) {
	if level < l.level {
		return
	}

	entry := LogEntry{
		Time:    time.Now().Format(time.RFC3339),
		Level:   level.String(),
		Message: msg,
		Fields:  l.parseFields(fields...),
	}

	l.mu.Lock()
	defer l.mu.Unlock()

	l.buffer = append(l.buffer, entry)

	// 如果缓冲区满了，立即刷新
	if len(l.buffer) >= l.bufferSize {
		l.flush()
	}
}

// parseFields 解析字段
func (l *Logger) parseFields(fields ...interface{}) map[string]interface{} {
	if len(fields) == 0 {
		return nil
	}

	result := make(map[string]interface{})
	for i := 0; i < len(fields)-1; i += 2 {
		key, ok := fields[i].(string)
		if !ok {
			continue
		}
		// 隐藏敏感信息
		if l.isSensitive(key) {
			result[key] = "***"
		} else {
			result[key] = fields[i+1]
		}
	}
	return result
}

// isSensitive 检查是否为敏感字段
func (l *Logger) isSensitive(key string) bool {
	sensitive := []string{"token", "apiKey", "secret", "password", "credential"}
	for _, s := range sensitive {
		if containsIgnoreCase(key, s) {
			return true
		}
	}
	return false
}

// flushLoop 定期刷新日志
func (l *Logger) flushLoop() {
	ticker := time.NewTicker(time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			l.mu.Lock()
			if len(l.buffer) > 0 {
				l.flush()
			}
			l.mu.Unlock()
		case <-l.stopCh:
			return
		}
	}
}

// flush 刷新日志到输出
func (l *Logger) flush() {
	if len(l.buffer) == 0 {
		return
	}

	// 检查是否需要轮转
	if l.file != nil && l.maxSize > 0 {
		if info, err := l.file.Stat(); err == nil && info.Size() > l.maxSize {
			l.rotate()
		}
	}

	for _, entry := range l.buffer {
		var line string
		if l.format == "json" {
			data, _ := json.Marshal(entry)
			line = string(data) + "\n"
		} else {
			line = fmt.Sprintf("[%s] %s: %s", entry.Time, entry.Level, entry.Message)
			if len(entry.Fields) > 0 {
				data, _ := json.Marshal(entry.Fields)
				line += " " + string(data)
			}
			line += "\n"
		}
		l.output.Write([]byte(line))
	}

	// 清空缓冲区
	l.buffer = l.buffer[:0]
}

// rotate 轮转日志文件
func (l *Logger) rotate() {
	if l.file == nil {
		return
	}

	// 关闭当前文件
	l.file.Close()

	// 重命名旧文件
	timestamp := time.Now().Format("20060102-150405")
	backupPath := l.filePath + "." + timestamp + ".gz"

	// 压缩旧文件
	go func() {
		oldFile, err := os.Open(l.filePath)
		if err != nil {
			return
		}
		defer oldFile.Close()

		gzipFile, err := os.Create(backupPath)
		if err != nil {
			return
		}
		defer gzipFile.Close()

		gzipWriter := gzip.NewWriter(gzipFile)
		defer gzipWriter.Close()

		io.Copy(gzipWriter, oldFile)
		os.Remove(l.filePath)
	}()

	// 打开新文件
	l.openFile()
}

// Close 关闭日志记录器
func (l *Logger) Close() error {
	close(l.stopCh)

	l.mu.Lock()
	defer l.mu.Unlock()

	l.flush()

	if l.file != nil {
		return l.file.Close()
	}
	return nil
}

// GetLevel 获取当前日志级别
func (l *Logger) GetLevel() Level {
	return l.level
}

// SetLevel 设置日志级别
func (l *Logger) SetLevel(level Level) {
	l.level = level
}

// GetRecentLogs 获取最近的日志条目（用于Web调试界面）
func (l *Logger) GetRecentLogs(count int) []LogEntry {
	l.mu.Lock()
	defer l.mu.Unlock()

	if len(l.buffer) == 0 {
		return nil
	}

	if count > len(l.buffer) {
		count = len(l.buffer)
	}

	// 返回最近的日志
	start := len(l.buffer) - count
	result := make([]LogEntry, count)
	copy(result, l.buffer[start:])
	return result
}

// containsIgnoreCase 检查字符串是否包含子串（忽略大小写）
func containsIgnoreCase(s, substr string) bool {
	return len(s) >= len(substr) && containsIgnoreCaseHelper(s, substr)
}

func containsIgnoreCaseHelper(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		match := true
		for j := 0; j < len(substr); j++ {
			if toLower(s[i+j]) != toLower(substr[j]) {
				match = false
				break
			}
		}
		if match {
			return true
		}
	}
	return false
}

func toLower(c byte) byte {
	if c >= 'A' && c <= 'Z' {
		return c + ('a' - 'A')
	}
	return c
}
