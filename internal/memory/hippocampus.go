package memory

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

type MemoryCategory string

const (
	CategoryPreference MemoryCategory = "preference"
	CategoryFact       MemoryCategory = "fact"
	CategoryEvent      MemoryCategory = "event"
	CategoryContact    MemoryCategory = "contact"
)

type MemoryItem struct {
	ID           string          `json:"id"`
	Category     MemoryCategory  `json:"category"`
	Content      string          `json:"content"`
	Keywords     []string        `json:"keywords"`
	Importance   int             `json:"importance"`
	CreatedAt    time.Time       `json:"createdAt"`
	LastAccessed time.Time       `json:"lastAccessed"`
	AccessCount  int             `json:"accessCount"`
	Source       string          `json:"source"`
}

type Hippocampus struct {
	LongTermMemory  map[string]*MemoryItem `json:"longTermMemory"`
	RecentFacts     []*MemoryItem          `json:"recentFacts"`
	UserPreferences map[string]string      `json:"userPreferences"`
	KeywordsIndex   map[string][]string    `json:"keywordsIndex"`
	mu              sync.RWMutex
	dataDir         string
	maxItems        int
}

func NewHippocampus(dataDir string, maxItems int) (*Hippocampus, error) {
	h := &Hippocampus{
		LongTermMemory:  make(map[string]*MemoryItem),
		RecentFacts:     make([]*MemoryItem, 0),
		UserPreferences: make(map[string]string),
		KeywordsIndex:   make(map[string][]string),
		dataDir:         dataDir,
		maxItems:        maxItems,
	}

	if err := os.MkdirAll(dataDir, 0755); err != nil {
		return nil, err
	}

	if err := h.load(); err != nil {
		return nil, err
	}

	return h, nil
}

func (h *Hippocampus) load() error {
	data, err := os.ReadFile(filepath.Join(h.dataDir, "hippocampus.json"))
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}

	return json.Unmarshal(data, h)
}

func (h *Hippocampus) save() error {
	data, err := json.MarshalIndent(h, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(filepath.Join(h.dataDir, "hippocampus.json"), data, 0644)
}

func (h *Hippocampus) Remember(content string, category MemoryCategory, source string) (*MemoryItem, error) {
	h.mu.Lock()
	defer h.mu.Unlock()

	item := &MemoryItem{
		ID:           generateID(),
		Category:     category,
		Content:      content,
		Keywords:     extractKeywords(content),
		Importance:   5,
		CreatedAt:    time.Now(),
		LastAccessed: time.Now(),
		AccessCount:  1,
		Source:       source,
	}

	h.LongTermMemory[item.ID] = item

	for _, kw := range item.Keywords {
		h.KeywordsIndex[kw] = append(h.KeywordsIndex[kw], item.ID)
	}

	switch category {
	case CategoryPreference:
		h.UserPreferences[strings.Join(item.Keywords, "_")] = content
	default:
		h.RecentFacts = append([]*MemoryItem{item}, h.RecentFacts...)
		if len(h.RecentFacts) > h.maxItems {
			h.RecentFacts = h.RecentFacts[:h.maxItems]
		}
	}

	if err := h.save(); err != nil {
		return nil, err
	}

	return item, nil
}

func (h *Hippocampus) Recall(query string) []*MemoryItem {
	h.mu.RLock()
	defer h.mu.RUnlock()

	keywords := extractKeywords(query)
	matchedIDs := make(map[string]int)

	for _, kw := range keywords {
		if ids, ok := h.KeywordsIndex[strings.ToLower(kw)]; ok {
			for _, id := range ids {
				matchedIDs[id]++
			}
		}
	}

	var results []*MemoryItem
	for id, matchCount := range matchedIDs {
		if item, ok := h.LongTermMemory[id]; ok {
			if matchCount >= 1 {
				results = append(results, item)
			}
		}
	}

	for i := range results {
		results[i].LastAccessed = time.Now()
		results[i].AccessCount++
	}

	return results
}

func (h *Hippocampus) GetPreferences() map[string]string {
	h.mu.RLock()
	defer h.mu.RUnlock()

	prefs := make(map[string]string)
	for k, v := range h.UserPreferences {
		prefs[k] = v
	}
	return prefs
}

func (h *Hippocampus) GetRecentFacts(limit int) []*MemoryItem {
	h.mu.RLock()
	defer h.mu.RUnlock()

	if limit > len(h.RecentFacts) {
		limit = len(h.RecentFacts)
	}
	return h.RecentFacts[:limit]
}

func (h *Hippocampus) GetAll() []*MemoryItem {
	h.mu.RLock()
	defer h.mu.RUnlock()

	items := make([]*MemoryItem, 0, len(h.LongTermMemory))
	for _, item := range h.LongTermMemory {
		items = append(items, item)
	}
	return items
}

func (h *Hippocampus) Forget(id string) bool {
	h.mu.Lock()
	defer h.mu.Unlock()

	item, ok := h.LongTermMemory[id]
	if !ok {
		return false
	}

	for _, kw := range item.Keywords {
		ids := h.KeywordsIndex[kw]
		for i, itemID := range ids {
			if itemID == id {
				h.KeywordsIndex[kw] = append(ids[:i], ids[i+1:]...)
				break
			}
		}
	}

	delete(h.LongTermMemory, id)

	for i, fact := range h.RecentFacts {
		if fact.ID == id {
			h.RecentFacts = append(h.RecentFacts[:i], h.RecentFacts[i+1:]...)
			break
		}
	}

	h.save()
	return true
}

func (h *Hippocampus) FormatContext() string {
	h.mu.RLock()
	defer h.mu.RUnlock()

	var sb strings.Builder

	if len(h.UserPreferences) > 0 {
		sb.WriteString("User preferences:\n")
		for k, v := range h.UserPreferences {
			sb.WriteString(fmt.Sprintf("- %s: %s\n", k, v))
		}
		sb.WriteString("\n")
	}

	if len(h.RecentFacts) > 0 {
		sb.WriteString("Recent facts:\n")
		for _, fact := range h.RecentFacts {
			if fact.AccessCount > 0 {
				sb.WriteString(fmt.Sprintf("- %s\n", fact.Content))
			}
		}
	}

	return sb.String()
}

func (h *Hippocampus) ShouldRemember(content string) bool {
	rememberPatterns := []string{
		"remember", "don't forget", "write down", "note that",
		"i like", "i love", "i hate", "i prefer", "my favorite",
		"my name is", "my birthday", "my phone", "my email", "my address",
		"记住", "别忘了", "记下来", "我喜欢", "我讨厌", "我的名字", "我的生日",
		"覚えて", "忘れないで", "メモして", "好き", "嫌い",
	}

	lowerContent := strings.ToLower(content)
	for _, pattern := range rememberPatterns {
		if strings.Contains(lowerContent, pattern) {
			return true
		}
	}

	return false
}

func (h *Hippocampus) DetectCategory(content string) MemoryCategory {
	lowerContent := strings.ToLower(content)

	if strings.Contains(lowerContent, "like") || strings.Contains(lowerContent, "prefer") ||
		strings.Contains(lowerContent, "hate") || strings.Contains(lowerContent, "favorite") ||
		strings.Contains(lowerContent, "喜欢") || strings.Contains(lowerContent, "讨厌") ||
		strings.Contains(lowerContent, "好き") || strings.Contains(lowerContent, "嫌い") {
		return CategoryPreference
	}

	if strings.Contains(lowerContent, "birthday") || strings.Contains(lowerContent, "meeting") ||
		strings.Contains(lowerContent, "appointment") || strings.Contains(lowerContent, "event") ||
		strings.Contains(lowerContent, "生日") || strings.Contains(lowerContent, "会议") ||
		strings.Contains(lowerContent, "誕生日") || strings.Contains(lowerContent, "会議") {
		return CategoryEvent
	}

	if strings.Contains(lowerContent, "phone") || strings.Contains(lowerContent, "email") ||
		strings.Contains(lowerContent, "address") || strings.Contains(lowerContent, "contact") ||
		strings.Contains(lowerContent, "电话") || strings.Contains(lowerContent, "邮箱") ||
		strings.Contains(lowerContent, "電話") || strings.Contains(lowerContent, "住所") {
		return CategoryContact
	}

	return CategoryFact
}

func generateID() string {
	return fmt.Sprintf("mem_%d", time.Now().UnixNano())
}

func extractKeywords(content string) []string {
	words := strings.Fields(strings.ToLower(content))
	keywords := make([]string, 0)

	stopWords := map[string]bool{
		"the": true, "a": true, "an": true, "is": true, "are": true,
		"was": true, "were": true, "be": true, "been": true,
		"have": true, "has": true, "had": true, "do": true,
		"does": true, "did": true, "will": true, "would": true,
		"could": true, "should": true, "may": true, "might": true,
		"must": true, "shall": true, "can": true, "need": true,
		"i": true, "you": true, "he": true, "she": true, "it": true,
		"we": true, "they": true, "this": true, "that": true,
		"these": true, "those": true, "to": true, "of": true,
		"in": true, "for": true, "on": true, "with": true,
		"at": true, "by": true, "from": true, "as": true,
		"的": true, "是": true, "在": true, "了": true, "和": true,
		"有": true, "我": true, "你": true, "他": true, "她": true,
		"の": true, "は": true, "が": true, "を": true, "に": true,
		"で": true, "と": true, "し": true, "て": true,
	}

	for _, word := range words {
		word = strings.Trim(word, ".,!?;:\"'()[]{}")
		if len(word) > 1 && !stopWords[word] {
			keywords = append(keywords, word)
		}
	}

	return keywords
}
