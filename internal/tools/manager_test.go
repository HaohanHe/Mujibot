package tools

import (
	"testing"
)

func TestManager_Execute(t *testing.T) {
	// 测试用例
	// ...
}

func TestIsDangerousCommand(t *testing.T) {
	tests := []struct {
		cmd      string
		expected bool
	}{
		{"rm -rf /", true},
		{"rm -rf /home/user", true},
		{"dd if=/dev/zero of=/dev/sda", true},
		{"mkfs.ext4 /dev/sda1", true},
		{"chmod 777 /etc/passwd", true},
		{"ls -la", false},
		{"cat file.txt", false},
		{"echo hello", false},
	}

	for _, tt := range tests {
		t.Run(tt.cmd, func(t *testing.T) {
			result := isDangerousCommand(tt.cmd)
			if result != tt.expected {
				t.Errorf("isDangerousCommand(%q) = %v, want %v", tt.cmd, result, tt.expected)
			}
		})
	}
}

func TestHasCommandInjection(t *testing.T) {
	tests := []struct {
		cmd      string
		expected bool
	}{
		{"ls -la", false},
		{"ls -la; rm -rf /", true},
		{"ls -la && rm -rf /", true},
		{"ls -la || rm -rf /", true},
		{"ls -la | grep test", true},
		{"echo \"test\"", false},
		{"echo 'test'", false},
		{"echo $(date)", true},
		{"echo `date`", true},
		{"echo ${HOME}", true},
	}

	for _, tt := range tests {
		t.Run(tt.cmd, func(t *testing.T) {
			result := hasCommandInjection(tt.cmd)
			if result != tt.expected {
				t.Errorf("hasCommandInjection(%q) = %v, want %v", tt.cmd, result, tt.expected)
			}
		})
	}
}

func TestIsPrivateIP(t *testing.T) {
	tests := []struct {
		ip       string
		expected bool
	}{
		{"192.168.1.1", true},
		{"10.0.0.1", true},
		{"172.16.0.1", true},
		{"127.0.0.1", true},
		{"8.8.8.8", false},
		{"1.1.1.1", false},
		{"localhost", false},
		{"example.com", false},
	}

	for _, tt := range tests {
		t.Run(tt.ip, func(t *testing.T) {
			result := isPrivateIP(tt.ip)
			if result != tt.expected {
				t.Errorf("isPrivateIP(%q) = %v, want %v", tt.ip, result, tt.expected)
			}
		})
	}
}
