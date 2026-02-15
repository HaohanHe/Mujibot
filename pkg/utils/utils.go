package utils

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"regexp"
	"strings"
	"unicode"
)

// GenerateID 生成随机ID
func GenerateID() string {
	b := make([]byte, 16)
	rand.Read(b)
	return hex.EncodeToString(b)
}

// Truncate 截断字符串
func Truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen-3] + "..."
}

// SanitizeString 清理字符串（去除特殊字符）
func SanitizeString(s string) string {
	// 只允许字母数字和常见标点
	re := regexp.MustCompile(`[^a-zA-Z0-9\s\-_\.@]`)
	return re.ReplaceAllString(s, "")
}

// ContainsString 检查字符串切片是否包含元素
func ContainsString(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}

// RemoveString 从切片中移除元素
func RemoveString(slice []string, item string) []string {
	result := make([]string, 0, len(slice))
	for _, s := range slice {
		if s != item {
			result = append(result, s)
		}
	}
	return result
}

// UniqueStrings 去重字符串切片
func UniqueStrings(slice []string) []string {
	seen := make(map[string]bool)
	result := make([]string, 0, len(slice))
	for _, s := range slice {
		if !seen[s] {
			seen[s] = true
			result = append(result, s)
		}
	}
	return result
}

// FormatBytes 格式化字节大小
func FormatBytes(bytes uint64) string {
	const unit = 1024
	if bytes < unit {
		return fmt.Sprintf("%d B", bytes)
	}
	div, exp := uint64(unit), 0
	for n := bytes / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %cB", float64(bytes)/float64(div), "KMGTPE"[exp])
}

// IsValidEmail 验证邮箱格式
func IsValidEmail(email string) bool {
	pattern := `^[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}$`
	re := regexp.MustCompile(pattern)
	return re.MatchString(email)
}

// MaskString 掩码字符串（如API密钥）
func MaskString(s string, visible int) string {
	if len(s) <= visible*2 {
		return strings.Repeat("*", len(s))
	}
	return s[:visible] + strings.Repeat("*", len(s)-visible*2) + s[len(s)-visible:]
}

// CleanWhitespace 清理多余空白字符
func CleanWhitespace(s string) string {
	// 将多个空白字符替换为单个空格
	re := regexp.MustCompile(`\s+`)
	s = re.ReplaceAllString(s, " ")
	// 去除首尾空白
	return strings.TrimSpace(s)
}

// WordCount 计算单词数
func WordCount(s string) int {
	words := strings.Fields(s)
	return len(words)
}

// ReverseString 反转字符串
func ReverseString(s string) string {
	runes := []rune(s)
	for i, j := 0, len(runes)-1; i < j; i, j = i+1, j-1 {
		runes[i], runes[j] = runes[j], runes[i]
	}
	return string(runes)
}

// IsPrintable 检查字符串是否可打印
func IsPrintable(s string) bool {
	for _, r := range s {
		if !unicode.IsPrint(r) && !unicode.IsSpace(r) {
			return false
		}
	}
	return true
}

// SafeFilename 生成安全的文件名
func SafeFilename(filename string) string {
	// 替换危险字符
	re := regexp.MustCompile(`[<>:"/\\|?*\x00-\x1f]`)
	filename = re.ReplaceAllString(filename, "_")
	
	// 限制长度
	if len(filename) > 255 {
		filename = filename[:255]
	}
	
	return filename
}

// ParseBool 解析布尔值（支持多种格式）
func ParseBool(s string) (bool, error) {
	s = strings.ToLower(strings.TrimSpace(s))
	switch s {
	case "true", "yes", "y", "1", "on":
		return true, nil
	case "false", "no", "n", "0", "off":
		return false, nil
	default:
		return false, fmt.Errorf("cannot parse %q as boolean", s)
	}
}
