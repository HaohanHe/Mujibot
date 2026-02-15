package health

import (
	"encoding/json"
	"net/http"
	"runtime"
	"sync"
	"time"

	"github.com/HaohanHe/mujibot/internal/logger"
)

// Checker 健康检查器
type Checker struct {
	startTime    time.Time
	messageCount uint64
	llmSuccess   uint64
	llmFailed    uint64
	mu           sync.RWMutex
	log          *logger.Logger
}

// Status 健康状态
type Status struct {
	Status        string                 `json:"status"`
	Version       string                 `json:"version"`
	Uptime        string                 `json:"uptime"`
	Timestamp     int64                  `json:"timestamp"`
	Memory        MemoryStats            `json:"memory"`
	Goroutines    int                    `json:"goroutines"`
	Messages      MessageStats           `json:"messages"`
	LLM           LLMStats               `json:"llm"`
}

// MemoryStats 内存统计
type MemoryStats struct {
	Alloc        uint64 `json:"alloc"`
	TotalAlloc   uint64 `json:"total_alloc"`
	Sys          uint64 `json:"sys"`
	HeapAlloc    uint64 `json:"heap_alloc"`
	HeapSys      uint64 `json:"heap_sys"`
	HeapObjects  uint64 `json:"heap_objects"`
	NumGC        uint32 `json:"num_gc"`
}

// MessageStats 消息统计
type MessageStats struct {
	Total   uint64 `json:"total"`
	PerHour uint64 `json:"per_hour"`
}

// LLMStats LLM统计
type LLMStats struct {
	Success uint64  `json:"success"`
	Failed  uint64  `json:"failed"`
	Rate    float64 `json:"rate"`
}

// NewChecker 创建健康检查器
func NewChecker(log *logger.Logger) *Checker {
	return &Checker{
		startTime: time.Now(),
		log:       log,
	}
}

// GetStatus 获取健康状态
func (c *Checker) GetStatus() Status {
	c.mu.RLock()
	defer c.mu.RUnlock()

	var m runtime.MemStats
	runtime.ReadMemStats(&m)

	uptime := time.Since(c.startTime)
	hours := int(uptime.Hours())
	minutes := int(uptime.Minutes()) % 60
	seconds := int(uptime.Seconds()) % 60

	llmTotal := c.llmSuccess + c.llmFailed
	llmRate := 0.0
	if llmTotal > 0 {
		llmRate = float64(c.llmSuccess) / float64(llmTotal) * 100
	}

	return Status{
		Status:    "healthy",
		Version:   "1.0.0",
		Uptime:    formatDuration(hours, minutes, seconds),
		Timestamp: time.Now().Unix(),
		Memory: MemoryStats{
			Alloc:       m.Alloc,
			TotalAlloc:  m.TotalAlloc,
			Sys:         m.Sys,
			HeapAlloc:   m.HeapAlloc,
			HeapSys:     m.HeapSys,
			HeapObjects: m.HeapObjects,
			NumGC:       m.NumGC,
		},
		Goroutines: runtime.NumGoroutine(),
		Messages: MessageStats{
			Total:   c.messageCount,
			PerHour: c.calculatePerHour(),
		},
		LLM: LLMStats{
			Success: c.llmSuccess,
			Failed:  c.llmFailed,
			Rate:    llmRate,
		},
	}
}

// RecordMessage 记录消息
func (c *Checker) RecordMessage() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.messageCount++
}

// RecordLLMSuccess 记录LLM成功
func (c *Checker) RecordLLMSuccess() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.llmSuccess++
}

// RecordLLMFailed 记录LLM失败
func (c *Checker) RecordLLMFailed() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.llmFailed++
}

// calculatePerHour 计算每小时消息数
func (c *Checker) calculatePerHour() uint64 {
	uptime := time.Since(c.startTime).Hours()
	if uptime < 1 {
		uptime = 1
	}
	return uint64(float64(c.messageCount) / uptime)
}

// CheckHealth 检查健康状态
func (c *Checker) CheckHealth() map[string]interface{} {
	status := c.GetStatus()

	// 检查内存使用
	memoryMB := status.Memory.HeapAlloc / 1024 / 1024
	if memoryMB > 70 {
		c.log.Warn("high memory usage detected", "heap_mb", memoryMB)
		return map[string]interface{}{
			"status":  "warning",
			"reason":  "high_memory",
			"memory":  memoryMB,
		}
	}

	return map[string]interface{}{
		"status": "healthy",
	}
}

// Handler HTTP处理器
func (c *Checker) Handler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		status := c.GetStatus()

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(status)
	}
}

// HealthHandler 健康检查处理器
func (c *Checker) HealthHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		health := c.CheckHealth()

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(health)
	}
}

// formatDuration 格式化持续时间
func formatDuration(hours, minutes, seconds int) string {
	if hours > 0 {
		return formatInt(hours) + ":" + formatInt(minutes) + ":" + formatInt(seconds)
	}
	return formatInt(minutes) + ":" + formatInt(seconds)
}

func formatInt(n int) string {
	if n < 10 {
		return "0" + string(rune('0'+n))
	}
	return string(rune('0'+n/10)) + string(rune('0'+n%10))
}
