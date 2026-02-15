package tools

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/HaohanHe/mujibot/internal/logger"
)

func TestNewManager(t *testing.T) {
	tempDir := t.TempDir()

	log, err := logger.New(logger.Config{Level: "error"})
	if err != nil {
		t.Fatalf("failed to create logger: %v", err)
	}
	defer log.Close()

	cfg := Config{
		WorkDir:          tempDir,
		Timeout:          30,
		ConfirmDangerous: true,
		BlockedCommands:  []string{"reboot", "shutdown"},
	}

	mgr, err := NewManager(cfg, log)
	if err != nil {
		t.Fatalf("failed to create manager: %v", err)
	}

	// 验证工作目录已创建
	if _, err := os.Stat(tempDir); os.IsNotExist(err) {
		t.Error("work directory should be created")
	}

	// 验证内置工具已注册
	tools := mgr.GetAll()
	if len(tools) == 0 {
		t.Error("builtin tools should be registered")
	}
}

func TestReadFileTool(t *testing.T) {
	tempDir := t.TempDir()

	log, _ := logger.New(logger.Config{Level: "error"})
	defer log.Close()

	cfg := Config{
		WorkDir:          tempDir,
		Timeout:          30,
		ConfirmDangerous: true,
	}

	mgr, _ := NewManager(cfg, log)

	// 创建测试文件
	testFile := filepath.Join(tempDir, "test.txt")
	testContent := "Hello, World!"
	os.WriteFile(testFile, []byte(testContent), 0644)

	// 测试读取文件
	result, err := mgr.Execute("read_file", map[string]interface{}{
		"path": "test.txt",
	})
	if err != nil {
		t.Errorf("read_file should succeed: %v", err)
	}
	if result != testContent {
		t.Errorf("read_file content mismatch, got: %s, want: %s", result, testContent)
	}
}

func TestWriteFileTool(t *testing.T) {
	tempDir := t.TempDir()

	log, _ := logger.New(logger.Config{Level: "error"})
	defer log.Close()

	cfg := Config{
		WorkDir:          tempDir,
		Timeout:          30,
		ConfirmDangerous: true,
	}

	mgr, _ := NewManager(cfg, log)

	// 测试写入文件
	testContent := "Test content"
	_, err := mgr.Execute("write_file", map[string]interface{}{
		"path":    "output.txt",
		"content": testContent,
	})
	if err != nil {
		t.Errorf("write_file should succeed: %v", err)
	}

	// 验证文件内容
	content, err := os.ReadFile(filepath.Join(tempDir, "output.txt"))
	if err != nil {
		t.Errorf("file should exist: %v", err)
	}
	if string(content) != testContent {
		t.Errorf("file content mismatch, got: %s, want: %s", string(content), testContent)
	}
}

func TestListDirectoryTool(t *testing.T) {
	tempDir := t.TempDir()

	log, _ := logger.New(logger.Config{Level: "error"})
	defer log.Close()

	cfg := Config{
		WorkDir:          tempDir,
		Timeout:          30,
		ConfirmDangerous: true,
	}

	mgr, _ := NewManager(cfg, log)

	// 创建测试文件和目录
	os.WriteFile(filepath.Join(tempDir, "file1.txt"), []byte("test"), 0644)
	os.Mkdir(filepath.Join(tempDir, "subdir"), 0755)

	// 测试列出目录
	result, err := mgr.Execute("list_directory", map[string]interface{}{
		"path": ".",
	})
	if err != nil {
		t.Errorf("list_directory should succeed: %v", err)
	}

	// 验证输出包含文件和目录
	if result == "" {
		t.Error("list_directory should return content")
	}
}

func TestSanitizePath(t *testing.T) {
	tempDir := t.TempDir()

	log, _ := logger.New(logger.Config{Level: "error"})
	defer log.Close()

	cfg := Config{
		WorkDir:          tempDir,
		Timeout:          30,
		ConfirmDangerous: true,
	}

	mgr, _ := NewManager(cfg, log)

	tests := []struct {
		name    string
		path    string
		wantErr bool
	}{
		{
			name:    "valid relative path",
			path:    "test.txt",
			wantErr: false,
		},
		{
			name:    "valid absolute path in workdir",
			path:    tempDir + string(filepath.Separator) + "test.txt",
			wantErr: false,
		},
		{
			name:    "path traversal attempt",
			path:    ".." + string(filepath.Separator) + ".." + string(filepath.Separator) + "etc" + string(filepath.Separator) + "passwd",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := mgr.sanitizePath(tt.path)
			if (err != nil) != tt.wantErr {
				t.Errorf("sanitizePath() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestIsDangerousCommand(t *testing.T) {
	tempDir := t.TempDir()

	log, _ := logger.New(logger.Config{Level: "error"})
	defer log.Close()

	cfg := Config{
		WorkDir:          tempDir,
		Timeout:          30,
		ConfirmDangerous: true,
	}

	mgr, _ := NewManager(cfg, log)

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
			result := mgr.isDangerousCommand(tt.cmd)
			if result != tt.expected {
				t.Errorf("isDangerousCommand(%q) = %v, want %v", tt.cmd, result, tt.expected)
			}
		})
	}
}
