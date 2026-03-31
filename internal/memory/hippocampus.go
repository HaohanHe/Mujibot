package memory

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"time"

	"github.com/HaohanHe/mujibot/internal/logger"
)

// MemoryItem 记忆项
type MemoryItem struct {
	ID           string    `json:"id"`
	Category     string    `json:"category"`     // preference, fact, event, contact
	Content      string    `json:"content"`      // 原始内容
	Keywords     []string  `json:"keywords"`     // 关键词
	Importance   int       `json:"importance"`   // 重要性 1-10
	CreatedAt    time.Time `json:"createdAt"`
	LastAccessed time.Time `json:"lastAccessed"`
	AccessCount  int       `json:"accessCount"`  // 访问次数
}

// Hippocampus 海马体记忆系统
type Hippocampus struct {
	LongTermMemory  map[string]MemoryItem  // 按ID索引的长时记忆
	RecentFacts     []MemoryItem           // 最近事实
	UserPreferences map[string]string      // 用户偏好
	memoryDir       string
	log             *logger.Logger
}

// NewHippocampus 创建海马体记忆系统
func NewHippocampus(memoryDir string, log *logger.Logger) *Hippocampus {
	h := &Hippocampus{
		LongTermMemory:  make(map[string]MemoryItem),
		RecentFacts:     make([]MemoryItem, 0),
		UserPreferences: make(map[string]string),
		memoryDir:       memoryDir,
		log:             log,
	}

	// 加载记忆数据
	h.load()

	return h
}

// AddMemory 添加记忆项
func (h *Hippocampus) AddMemory(category, content string, keywords []string, importance int) string {
	id := fmt.Sprintf("%d", time.Now().UnixNano())

	item := MemoryItem{
		ID:           id,
		Category:     category,
		Content:      content,
		Keywords:     keywords,
		Importance:   importance,
		CreatedAt:    time.Now(),
		LastAccessed: time.Now(),
		AccessCount:  1,
	}

	// 添加到长时记忆
	h.LongTermMemory[id] = item

	// 添加到最近事实（最多保存100条）
	h.RecentFacts = append(h.RecentFacts, item)
	if len(h.RecentFacts) > 100 {
		h.RecentFacts = h.RecentFacts[1:]
	}

	// 如果是偏好，添加到用户偏好
	if category == "preference" {
		for _, keyword := range keywords {
			h.UserPreferences[keyword] = content
		}
	}

	// 保存到文件
	h.save()

	return id
}

// GetMemory 获取记忆项
func (h *Hippocampus) GetMemory(id string) (MemoryItem, bool) {
	item, exists := h.LongTermMemory[id]
	if exists {
		// 更新访问信息
		item.LastAccessed = time.Now()
		item.AccessCount++
		h.LongTermMemory[id] = item
		h.save()
	}
	return item, exists
}

// SearchMemory 搜索记忆
func (h *Hippocampus) SearchMemory(query string, limit int) []MemoryItem {
	var results []MemoryItem

	// 搜索长时记忆
	for _, item := range h.LongTermMemory {
		if containsKeyword(item, query) {
			results = append(results, item)
		}
	}

	// 按重要性和访问次数排序
	sort.Slice(results, func(i, j int) bool {
		if results[i].Importance != results[j].Importance {
			return results[i].Importance > results[j].Importance
		}
		return results[i].AccessCount > results[j].AccessCount
	})

	// 限制返回数量
	if len(results) > limit {
		results = results[:limit]
	}

	return results
}

// GetUserPreferences 获取用户偏好
func (h *Hippocampus) GetUserPreferences() map[string]string {
	return h.UserPreferences
}

// GetRecentFacts 获取最近事实
func (h *Hippocampus) GetRecentFacts(limit int) []MemoryItem {
	if len(h.RecentFacts) <= limit {
		return h.RecentFacts
	}
	return h.RecentFacts[len(h.RecentFacts)-limit:]
}

// DeleteMemory 删除记忆项
func (h *Hippocampus) DeleteMemory(id string) bool {
	if _, exists := h.LongTermMemory[id]; exists {
		delete(h.LongTermMemory, id)

		// 从最近事实中删除
		for i, item := range h.RecentFacts {
			if item.ID == id {
				h.RecentFacts = append(h.RecentFacts[:i], h.RecentFacts[i+1:]...)
				break
			}
		}

		h.save()
		return true
	}
	return false
}

// UpdateMemory 更新记忆项
func (h *Hippocampus) UpdateMemory(id, content string, keywords []string, importance int) bool {
	if item, exists := h.LongTermMemory[id]; exists {
		item.Content = content
		item.Keywords = keywords
		item.Importance = importance
		item.LastAccessed = time.Now()
		h.LongTermMemory[id] = item

		// 更新最近事实
		for i, recentItem := range h.RecentFacts {
			if recentItem.ID == id {
				h.RecentFacts[i] = item
				break
			}
		}

		h.save()
		return true
	}
	return false
}

// save 保存记忆到文件
func (h *Hippocampus) save() {
	// 确保目录存在
	if err := os.MkdirAll(h.memoryDir, 0755); err != nil {
		h.log.Error("failed to create memory directory", "error", err)
		return
	}

	// 保存长时记忆
	longTermPath := filepath.Join(h.memoryDir, "long_term_memory.json")
	data, err := json.MarshalIndent(h.LongTermMemory, "", "  ")
	if err != nil {
		h.log.Error("failed to marshal long term memory", "error", err)
		return
	}

	if err := os.WriteFile(longTermPath, data, 0644); err != nil {
		h.log.Error("failed to write long term memory", "error", err)
	}

	// 保存最近事实
	recentPath := filepath.Join(h.memoryDir, "recent_facts.json")
	data, err = json.MarshalIndent(h.RecentFacts, "", "  ")
	if err != nil {
		h.log.Error("failed to marshal recent facts", "error", err)
		return
	}

	if err := os.WriteFile(recentPath, data, 0644); err != nil {
		h.log.Error("failed to write recent facts", "error", err)
	}

	// 保存用户偏好
	prefsPath := filepath.Join(h.memoryDir, "user_preferences.json")
	data, err = json.MarshalIndent(h.UserPreferences, "", "  ")
	if err != nil {
		h.log.Error("failed to marshal user preferences", "error", err)
		return
	}

	if err := os.WriteFile(prefsPath, data, 0644); err != nil {
		h.log.Error("failed to write user preferences", "error", err)
	}
}

// load 从文件加载记忆
func (h *Hippocampus) load() {
	// 加载长时记忆
	longTermPath := filepath.Join(h.memoryDir, "long_term_memory.json")
	if data, err := os.ReadFile(longTermPath); err == nil {
		if err := json.Unmarshal(data, &h.LongTermMemory); err != nil {
			h.log.Error("failed to unmarshal long term memory", "error", err)
		}
	}

	// 加载最近事实
	recentPath := filepath.Join(h.memoryDir, "recent_facts.json")
	if data, err := os.ReadFile(recentPath); err == nil {
		if err := json.Unmarshal(data, &h.RecentFacts); err != nil {
			h.log.Error("failed to unmarshal recent facts", "error", err)
		}
	}

	// 加载用户偏好
	prefsPath := filepath.Join(h.memoryDir, "user_preferences.json")
	if data, err := os.ReadFile(prefsPath); err == nil {
		if err := json.Unmarshal(data, &h.UserPreferences); err != nil {
			h.log.Error("failed to unmarshal user preferences", "error", err)
		}
	}
}

// containsKeyword 检查记忆项是否包含关键词
func containsKeyword(item MemoryItem, query string) bool {
	// 检查内容
	if containsString(item.Content, query) {
		return true
	}

	// 检查关键词
	for _, keyword := range item.Keywords {
		if containsString(keyword, query) {
			return true
		}
	}

	return false
}

// containsString 检查字符串是否包含子字符串（不区分大小写）
func containsString(s, substr string) bool {
	return len(s) >= len(substr) && containsIgnoreCase(s, substr)
}

// containsIgnoreCase 不区分大小写的字符串包含检查
func containsIgnoreCase(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if equalFold(s[i:i+len(substr)], substr) {
			return true
		}
	}
	return false
}

// equalFold 不区分大小写的字符串比较
func equalFold(s1, s2 string) bool {
	if len(s1) != len(s2) {
		return false
	}
	for i := 0; i < len(s1); i++ {
		c1 := s1[i]
		c2 := s2[i]
		if c1 >= 'A' && c1 <= 'Z' {
			c1 += 'a' - 'A'
		}
		if c2 >= 'A' && c2 <= 'Z' {
			c2 += 'a' - 'A'
		}
		if c1 != c2 {
			return false
		}
	}
	return true
}
