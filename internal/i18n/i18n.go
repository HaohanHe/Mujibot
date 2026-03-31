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
