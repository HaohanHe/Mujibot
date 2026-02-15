package i18n

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sync"
)

type Messages struct {
	Hello            string `json:"hello"`
	SelectLanguage   string `json:"selectLanguage"`
	CurrentTime      string `json:"currentTime"`
	Timezone         string `json:"timezone"`
	SystemType       string `json:"systemType"`
	AvailableTools   string `json:"availableTools"`
	ToolsIntro       string `json:"toolsIntro"`
	MemoryContext    string `json:"memoryContext"`
	ToolUsage        string `json:"toolUsage"`
	UserLanguage     string `json:"userLanguage"`
	ReplyInSameLang  string `json:"replyInSameLang"`
	MemoryRulesTitle string `json:"memoryRulesTitle"`
	MemoryRules      string `json:"memoryRules"`
	MemoryCategories string `json:"memoryCategories"`
}

var defaultMessages = map[string]Messages{
	"en-US": {
		Hello:            "Hello",
		SelectLanguage:   "Please select your language",
		CurrentTime:      "Current time",
		Timezone:         "Timezone",
		SystemType:       "System type",
		AvailableTools:   "Available tools",
		ToolsIntro:       "You can use the following tools to help users:",
		MemoryContext:    "Memory context",
		ToolUsage:        "When using tools, ensure parameters are correct. If a tool call fails, explain the reason to the user.",
		UserLanguage:     "User language",
		ReplyInSameLang:  "Please reply in the same language as the user.",
		MemoryRulesTitle: "Memory rules",
		MemoryRules: `When the user expresses the following intentions, automatically call the memory_write tool:
1. "Remember..." / "Don't forget..." / "Write this down..."
2. "I like..." / "I hate..." / "My..."
3. Important dates, contacts, addresses
4. Information the user repeatedly mentions`,
		MemoryCategories: `Memory categories:
- preference: User preferences
- fact: Factual information
- event: Events/dates
- contact: Contact information`,
	},
	"zh-CN": {
		Hello:            "你好",
		SelectLanguage:   "请选择您的语言",
		CurrentTime:      "当前时间",
		Timezone:         "时区",
		SystemType:       "系统类型",
		AvailableTools:   "可用工具",
		ToolsIntro:       "你可以使用以下工具来帮助用户:",
		MemoryContext:    "记忆上下文",
		ToolUsage:        "使用工具时，请确保参数正确。如果工具调用失败，向用户解释原因。",
		UserLanguage:     "用户语言",
		ReplyInSameLang:  "请使用与用户相同的语言回复。",
		MemoryRulesTitle: "记忆规则",
		MemoryRules: `当用户表达以下意图时，自动调用 memory_write 工具：
1. "记住..." / "别忘了..." / "记下来..."
2. "我喜欢..." / "我讨厌..." / "我的..."
3. 重要日期、联系方式、地址等
4. 用户反复提及的信息`,
		MemoryCategories: `记忆分类：
- preference: 用户偏好
- fact: 事实信息
- event: 事件/日期
- contact: 联系人信息`,
	},
	"ja-JP": {
		Hello:            "こんにちは",
		SelectLanguage:   "言語を選択してください",
		CurrentTime:      "現在時刻",
		Timezone:         "タイムゾーン",
		SystemType:       "システムタイプ",
		AvailableTools:   "利用可能なツール",
		ToolsIntro:       "以下のツールを使用してユーザーを支援できます:",
		MemoryContext:    "メモリコンテキスト",
		ToolUsage:        "ツールを使用する際は、パラメータが正しいことを確認してください。ツールの呼び出しに失敗した場合は、ユーザーに理由を説明してください。",
		UserLanguage:     "ユーザー言語",
		ReplyInSameLang:  "ユーザーと同じ言語で返信してください。",
		MemoryRulesTitle: "メモリルール",
		MemoryRules: `ユーザーが以下の意図を表現した場合、自動的にmemory_writeツールを呼び出します：
1. 「覚えて...」/「忘れないで...」/「書き留めて...」
2. 「私は...が好き」/「私は...が嫌い」/「私の...」
3. 重要な日付、連絡先、住所
4. ユーザーが繰り返し言及する情報`,
		MemoryCategories: `メモリカテゴリ：
- preference: ユーザーの好み
- fact: 事実情報
- event: イベント/日付
- contact: 連絡先情報`,
	},
}

type I18n struct {
	currentLang string
	messages    map[string]Messages
	mu          sync.RWMutex
}

func New(defaultLang string) *I18n {
	return &I18n{
		currentLang: defaultLang,
		messages:    defaultMessages,
	}
}

func (i *I18n) SetLanguage(lang string) {
	i.mu.Lock()
	defer i.mu.Unlock()
	i.currentLang = lang
}

func (i *I18n) GetLanguage() string {
	i.mu.RLock()
	defer i.mu.RUnlock()
	return i.currentLang
}

func (i *I18n) T(key string) string {
	i.mu.RLock()
	defer i.mu.RUnlock()

	msgs, ok := i.messages[i.currentLang]
	if !ok {
		msgs = i.messages["en-US"]
	}

	switch key {
	case "hello":
		return msgs.Hello
	case "selectLanguage":
		return msgs.SelectLanguage
	case "currentTime":
		return msgs.CurrentTime
	case "timezone":
		return msgs.Timezone
	case "systemType":
		return msgs.SystemType
	case "availableTools":
		return msgs.AvailableTools
	case "toolsIntro":
		return msgs.ToolsIntro
	case "memoryContext":
		return msgs.MemoryContext
	case "toolUsage":
		return msgs.ToolUsage
	case "userLanguage":
		return msgs.UserLanguage
	case "replyInSameLang":
		return msgs.ReplyInSameLang
	case "memoryRulesTitle":
		return msgs.MemoryRulesTitle
	case "memoryRules":
		return msgs.MemoryRules
	case "memoryCategories":
		return msgs.MemoryCategories
	default:
		return key
	}
}

func (i *I18n) GetMessages() Messages {
	i.mu.RLock()
	defer i.mu.RUnlock()

	msgs, ok := i.messages[i.currentLang]
	if !ok {
		msgs = i.messages["en-US"]
	}
	return msgs
}

func (i *I18n) LoadCustomTranslations(dir string) error {
	files, err := os.ReadDir(dir)
	if err != nil {
		return err
	}

	for _, file := range files {
		if filepath.Ext(file.Name()) == ".json" {
			lang := file.Name()[:len(file.Name())-5]
			data, err := os.ReadFile(filepath.Join(dir, file.Name()))
			if err != nil {
				continue
			}

			var msgs Messages
			if err := json.Unmarshal(data, &msgs); err != nil {
				continue
			}

			i.mu.Lock()
			i.messages[lang] = msgs
			i.mu.Unlock()
		}
	}

	return nil
}

func SupportedLanguages() []string {
	return []string{"en-US", "zh-CN", "ja-JP"}
}

func LanguageName(code string) string {
	names := map[string]string{
		"en-US": "English (US)",
		"zh-CN": "简体中文",
		"ja-JP": "日本語",
	}
	return names[code]
}
