package system

import (
	"bufio"
	"fmt"
	"os"
	"runtime"
	"strings"
	"time"
)

// SystemInfo 系统信息结构体
type SystemInfo struct {
	Type     string // 系统类型: Linux/Windows/Darwin
	Arch     string // 架构: amd64/arm64/arm
	Kernel   string // 内核版本
	Hostname string // 主机名
}

// GetSystemInfo 获取系统信息
func GetSystemInfo() *SystemInfo {
	sysType := runtime.GOOS
	sysArch := runtime.GOARCH
	kernel := getKernelVersion()
	hostname, _ := os.Hostname()

	return &SystemInfo{
		Type:     sysType,
		Arch:     sysArch,
		Kernel:   kernel,
		Hostname: hostname,
	}
}

// GetCurrentTime 获取当前时间
func GetCurrentTime() string {
	return time.Now().Format("2006-01-02 15:04:05")
}

// GetTimezone 获取时区
func GetTimezone() string {
	zone, _ := time.Now().Zone()
	return zone
}

// GetInfo 获取系统信息
func GetInfo() *SystemInfo {
	return GetSystemInfo()
}

// Format 格式化系统信息
func (s *SystemInfo) Format() string {
	return s.FormatForPrompt()
}

// getKernelVersion 获取内核版本
func getKernelVersion() string {
	// 在 Linux 系统上读取 /proc/version
	if runtime.GOOS == "linux" {
		file, err := os.Open("/proc/version")
		if err == nil {
			defer file.Close()
			scanner := bufio.NewScanner(file)
			if scanner.Scan() {
				line := scanner.Text()
				// 提取内核版本信息
				parts := strings.Fields(line)
				if len(parts) >= 3 {
					return parts[2]
				}
				return line
			}
		}
	}
	// 其他系统返回默认值
	return "unknown"
}

// String 返回系统信息的字符串表示
func (s *SystemInfo) String() string {
	return fmt.Sprintf("%s (%s) %s @ %s", s.Type, s.Arch, s.Kernel, s.Hostname)
}

// FormatForPrompt 为系统提示词格式化系统信息
func (s *SystemInfo) FormatForPrompt() string {
	return fmt.Sprintf("- 系统类型: %s\n- 系统架构: %s\n- 内核版本: %s\n- 主机名: %s",
		s.Type, s.Arch, s.Kernel, s.Hostname)
}
