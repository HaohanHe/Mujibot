package i18n

import (
	"fmt"
	"strings"

	"github.com/HaohanHe/mujibot/internal/system"
)

// Language 语言代码
type Language string

const (
	English  Language = "en-US"
	Chinese  Language = "zh-CN"
	Japanese Language = "ja-JP"
)

// I18n 国际化结构体
type I18n struct {
	Language Language
}

// New 创建国际化实例
func New(lang string) *I18n {
	return &I18n{
		Language: GetLanguageByCode(lang),
	}
}

// T 获取翻译
func (i *I18n) T(key string) string {
	switch key {
	case "currentTime":
		switch i.Language {
		case Chinese:
			return "当前时间"
		case Japanese:
			return "現在の時間"
		default:
			return "Current Time"
		}
	case "timezone":
		switch i.Language {
		case Chinese:
			return "时区"
		case Japanese:
			return "タイムゾーン"
		default:
			return "Timezone"
		}
	case "systemType":
		switch i.Language {
		case Chinese:
			return "系统类型"
		case Japanese:
			return "システムタイプ"
		default:
			return "System Type"
		}
	case "availableTools":
		switch i.Language {
		case Chinese:
			return "可用工具"
		case Japanese:
			return "利用可能なツール"
		default:
			return "Available Tools"
		}
	case "toolsIntro":
		switch i.Language {
		case Chinese:
			return "你可以使用以下工具来协助用户："
		case Japanese:
			return "以下のツールを使用してユーザーを支援できます："
		default:
			return "You can use the following tools to assist users:"
		}
	case "toolUsage":
		switch i.Language {
		case Chinese:
			return "使用工具时，请提供必要的参数，并确保操作安全。"
		case Japanese:
			return "ツールを使用する場合は、必要なパラメータを提供し、操作が安全であることを確認してください。"
		default:
			return "When using tools, please provide necessary parameters and ensure operations are safe."
		}
	case "memoryContext":
		switch i.Language {
		case Chinese:
			return "记忆上下文"
		case Japanese:
			return "記憶コンテキスト"
		default:
			return "Memory Context"
		}
	case "userLanguage":
		switch i.Language {
		case Chinese:
			return "用户语言"
		case Japanese:
			return "ユーザーの言語"
		default:
			return "User Language"
		}
	case "replyInSameLang":
		switch i.Language {
		case Chinese:
			return "请使用与用户相同的语言回复。"
		case Japanese:
			return "ユーザーと同じ言語で返信してください。"
		default:
			return "Please respond in the same language as the user."
		}
	case "memoryRulesTitle":
		switch i.Language {
		case Chinese:
			return "记忆规则"
		case Japanese:
			return "記憶ルール"
		default:
			return "Memory Rules"
		}
	case "memoryRules":
		switch i.Language {
		case Chinese:
			return "当用户表达以下意图时，自动调用 memory_write 工具：\n1. \"记住...\" / \"别忘了...\" / \"记下来...\"\n2. \"我喜欢...\" / \"我讨厌...\" / \"我的...\"\n3. 重要日期、联系方式、地址等\n4. 用户反复提及的信息"
		case Japanese:
			return "ユーザーが次の意図を表現したときは、自動的に memory_write ツールを呼び出してください：\n1. \"覚えて...\" / \"忘れないで...\" / \"メモして...\"\n2. \"私は...が好きです\" / \"私は...が嫌いです\" / \"私の...\"\n3. 重要な日付、連絡先、住所など\n4. ユーザーが繰り返し言及する情報"
		default:
			return "When users express the following intentions, automatically call the memory_write tool:\n1. \"Remember...\" / \"Don't forget...\" / \"Write down...\"\n2. \"I like...\" / \"I hate...\" / \"My...\"\n3. Important dates, contact information, addresses, etc.\n4. Information that users repeatedly mention"
		}
	case "memoryCategories":
		switch i.Language {
		case Chinese:
			return "记忆分类：\n- preference: 用户偏好\n- fact: 事实信息\n- event: 事件/日期\n- contact: 联系人信息"
		case Japanese:
			return "記憶のカテゴリ：\n- preference: ユーザーの嗜好\n- fact: 事実情報\n- event: イベント/日付\n- contact: 連絡先情報"
		default:
			return "Memory categories:\n- preference: user preferences\n- fact: factual information\n- event: events/dates\n- contact: contact information"
		}
	default:
		return key
	}
}

// SystemPrompts 系统提示词映射
var SystemPrompts = map[Language]string{
	English: `You are an AI assistant running on %s. You are efficient, concise, and helpful. You can use various tools to assist users.`,
	Chinese: `你是一个运行在 %s 上的AI助手。你高效、简洁、乐于助人。你可以使用各种工具来协助用户。`,
	Japanese: `あなたは %s で動作するAIアシスタントです。あなたは効率的で、簡潔で、助けになります。さまざまなツールを使用してユーザーを支援することができます。`,
}

// MemoryRules 记忆规则映射
var MemoryRules = map[Language]string{
	English: `## Memory Rules

When users express the following intentions, automatically call the memory_write tool:
1. "Remember..." / "Don't forget..." / "Write down..."
2. "I like..." / "I hate..." / "My..."
3. Important dates, contact information, addresses, etc.
4. Information that users repeatedly mention

Memory categories:
- preference: user preferences
- fact: factual information
- event: events/dates
- contact: contact information`,
	Chinese: `## 记忆规则

当用户表达以下意图时，自动调用 memory_write 工具：
1. "记住..." / "别忘了..." / "记下来..."
2. "我喜欢..." / "我讨厌..." / "我的..."
3. 重要日期、联系方式、地址等
4. 用户反复提及的信息

记忆分类：
- preference: 用户偏好
- fact: 事实信息
- event: 事件/日期
- contact: 联系人信息`,
	Japanese: `## 記憶ルール

ユーザーが次の意図を表現したときは、自動的に memory_write ツールを呼び出してください：
1. "覚えて..." / "忘れないで..." / "メモして..."
2. "私は...が好きです" / "私は...が嫌いです" / "私の..."
3. 重要な日付、連絡先、住所など
4. ユーザーが繰り返し言及する情報

記憶のカテゴリ：
- preference: ユーザーの嗜好
- fact: 事実情報
- event: イベント/日付
- contact: 連絡先情報`,
}

// UserLanguagePrompt 用户语言提示
var UserLanguagePrompt = map[Language]string{
	English: `## User Language

Current user language: {detected_language}

Please respond in the same language as the user.`,
	Chinese: `## 用户语言

当前用户使用语言: {detected_language}

请使用与用户相同的语言回复。`,
	Japanese: `## ユーザーの言語

現在のユーザーの言語: {detected_language}

ユーザーと同じ言語で返信してください。`,
}

// Greetings 问候语映射
var Greetings = map[Language]string{
	English: "Hello",
	Chinese: "你好",
	Japanese: "こんにちは",
}

// LanguageOptions 语言选择选项
var LanguageOptions = map[Language]string{
	English: "English (US)",
	Chinese: "简体中文",
	Japanese: "日本語",
}

// GetSystemPrompt 获取系统提示词
func GetSystemPrompt(lang Language, systemInfo *system.SystemInfo) string {
	prompt, exists := SystemPrompts[lang]
	if !exists {
		prompt = SystemPrompts[English] // 默认使用英文
	}
	return fmt.Sprintf(prompt, systemInfo.String())
}

// GetMemoryRules 获取记忆规则
func GetMemoryRules(lang Language) string {
	rules, exists := MemoryRules[lang]
	if !exists {
		rules = MemoryRules[English] // 默认使用英文
	}
	return rules
}

// GetUserLanguagePrompt 获取用户语言提示
func GetUserLanguagePrompt(lang Language, detectedLang string) string {
	prompt, exists := UserLanguagePrompt[lang]
	if !exists {
		prompt = UserLanguagePrompt[English] // 默认使用英文
	}
	return strings.Replace(prompt, "{detected_language}", detectedLang, 1)
}

// GetGreeting 获取问候语
func GetGreeting(lang Language) string {
	greeting, exists := Greetings[lang]
	if !exists {
		greeting = Greetings[English] // 默认使用英文
	}
	return greeting
}

// GetLanguageOptions 获取语言选择选项
func GetLanguageOptions() map[Language]string {
	return LanguageOptions
}

// DetectLanguage 检测用户语言
func DetectLanguage(text string) Language {
	// 简单的语言检测逻辑
	// 检查中文字符
	for _, r := range text {
		if r >= 0x4e00 && r <= 0x9fff {
			return Chinese
		}
	}

	// 检查日文字符
	for _, r := range text {
		if (r >= 0x3040 && r <= 0x30ff) || (r >= 0xff60 && r <= 0xff9f) {
			return Japanese
		}
	}

	// 默认返回英文
	return English
}

// IsSupportedLanguage 检查语言是否被支持
func IsSupportedLanguage(lang string) bool {
	supportedLangs := []Language{English, Chinese, Japanese}
	for _, supportedLang := range supportedLangs {
		if string(supportedLang) == lang {
			return true
		}
	}
	return false
}

// GetLanguageByCode 根据语言代码获取语言
func GetLanguageByCode(code string) Language {
	supportedLangs := []Language{English, Chinese, Japanese}
	for _, lang := range supportedLangs {
		if string(lang) == code {
			return lang
		}
	}
	return English // 默认返回英文
}

// SupportedLanguages 获取支持的语言列表
func SupportedLanguages() []string {
	return []string{string(English), string(Chinese), string(Japanese)}
}

// LanguageName 获取语言名称
func LanguageName(lang string) string {
	languageMap := map[string]string{
		"en-US": "English (US)",
		"zh-CN": "简体中文",
		"ja-JP": "日本語",
	}
	if name, ok := languageMap[lang]; ok {
		return name
	}
	return lang
}
