package system

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"runtime"
	"strings"
	"time"
)

type SystemInfo struct {
	OS           string `json:"os"`
	Arch         string `json:"arch"`
	KernelVersion string `json:"kernelVersion"`
	Hostname     string `json:"hostname"`
	Distro       string `json:"distro"`
	DistroVersion string `json:"distroVersion"`
	MemoryTotal  uint64 `json:"memoryTotal"`
	CPUModel     string `json:"cpuModel"`
	CPUCores     int    `json:"cpuCores"`
}

func GetInfo() *SystemInfo {
	info := &SystemInfo{
		OS:       runtime.GOOS,
		Arch:     runtime.GOARCH,
		CPUCores: runtime.NumCPU(),
	}

	info.Hostname, _ = os.Hostname()

	switch runtime.GOOS {
	case "linux":
		info.getLinuxInfo()
	case "darwin":
		info.getDarwinInfo()
	case "windows":
		info.getWindowsInfo()
	}

	return info
}

func (i *SystemInfo) getLinuxInfo() {
	if data, err := os.ReadFile("/proc/version"); err == nil {
		i.KernelVersion = strings.TrimSpace(string(data))
		if parts := strings.Split(i.KernelVersion, " "); len(parts) > 2 {
			i.KernelVersion = parts[2]
		}
	}

	if data, err := os.ReadFile("/etc/os-release"); err == nil {
		lines := strings.Split(string(data), "\n")
		for _, line := range lines {
			if strings.HasPrefix(line, "PRETTY_NAME=") {
				i.Distro = strings.Trim(strings.TrimPrefix(line, "PRETTY_NAME="), "\"")
			}
			if strings.HasPrefix(line, "VERSION_ID=") {
				i.DistroVersion = strings.Trim(strings.TrimPrefix(line, "VERSION_ID="), "\"")
			}
		}
	}

	if data, err := os.ReadFile("/proc/meminfo"); err == nil {
		lines := strings.Split(string(data), "\n")
		for _, line := range lines {
			if strings.HasPrefix(line, "MemTotal:") {
				fields := strings.Fields(line)
				if len(fields) >= 2 {
					fmt.Sscanf(fields[1], "%d", &i.MemoryTotal)
					i.MemoryTotal = i.MemoryTotal / 1024
				}
				break
			}
		}
	}

	if data, err := os.ReadFile("/proc/cpuinfo"); err == nil {
		lines := strings.Split(string(data), "\n")
		for _, line := range lines {
			if strings.HasPrefix(line, "model name") || strings.HasPrefix(line, "Model") {
				parts := strings.SplitN(line, ":", 2)
				if len(parts) == 2 {
					i.CPUModel = strings.TrimSpace(parts[1])
				}
				break
			}
		}
	}
}

func (i *SystemInfo) getDarwinInfo() {
	if out, err := exec.Command("uname", "-r").Output(); err == nil {
		i.KernelVersion = strings.TrimSpace(string(out))
	}

	if out, err := exec.Command("sw_vers", "-productName").Output(); err == nil {
		i.Distro = strings.TrimSpace(string(out))
	}
	if out, err := exec.Command("sw_vers", "-productVersion").Output(); err == nil {
		i.DistroVersion = strings.TrimSpace(string(out))
	}
}

func (i *SystemInfo) getWindowsInfo() {
	if out, err := exec.Command("cmd", "/c", "ver").Output(); err == nil {
		i.KernelVersion = strings.TrimSpace(string(out))
	}
	i.Distro = "Windows"
}

func (i *SystemInfo) Format() string {
	var buf bytes.Buffer

	buf.WriteString(fmt.Sprintf("- 操作系统: %s", i.OS))
	if i.Distro != "" {
		buf.WriteString(fmt.Sprintf(" (%s", i.Distro))
		if i.DistroVersion != "" {
			buf.WriteString(fmt.Sprintf(" %s", i.DistroVersion))
		}
		buf.WriteString(")")
	}
	buf.WriteString("\n")

	buf.WriteString(fmt.Sprintf("- 系统架构: %s", i.Arch))
	switch i.Arch {
	case "arm":
		buf.WriteString(" (ARM 32-bit)")
	case "arm64":
		buf.WriteString(" (ARM 64-bit)")
	case "amd64":
		buf.WriteString(" (x86_64)")
	case "386":
		buf.WriteString(" (x86 32-bit)")
	}
	if i.CPUModel != "" {
		buf.WriteString(fmt.Sprintf(" - %s", i.CPUModel))
	}
	buf.WriteString("\n")

	if i.KernelVersion != "" {
		buf.WriteString(fmt.Sprintf("- 内核版本: %s\n", i.KernelVersion))
	}

	buf.WriteString(fmt.Sprintf("- CPU核心: %d\n", i.CPUCores))

	if i.MemoryTotal > 0 {
		buf.WriteString(fmt.Sprintf("- 内存容量: %d MB\n", i.MemoryTotal))
	}

	buf.WriteString(fmt.Sprintf("- 主机名: %s\n", i.Hostname))

	return buf.String()
}

func (i *SystemInfo) ShortInfo() string {
	arch := i.Arch
	switch i.Arch {
	case "arm":
		arch = "ARMv7"
	case "arm64":
		arch = "ARM64"
	case "amd64":
		arch = "x64"
	}

	if i.Distro != "" {
		return fmt.Sprintf("%s/%s", i.Distro, arch)
	}
	return fmt.Sprintf("%s/%s", i.OS, arch)
}

func GetCurrentTime() string {
	return time.Now().Format("2006-01-02 15:04:05 MST")
}

func GetTimezone() string {
	name, _ := time.Now().Zone()
	return name
}
