package config

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/HaohanHe/mujibot/internal/logger"
)

func TestNewManager(t *testing.T) {
	tempDir := t.TempDir()
	configPath := filepath.Join(tempDir, "test_config.json5")

	// 设置假的 API Key 环境变量以通过验证
	os.Setenv("OPENAI_API_KEY", "test-key-for-testing")
	defer os.Unsetenv("OPENAI_API_KEY")

	log, err := logger.New(logger.Config{Level: "error", Format: "json"})
	if err != nil {
		t.Fatalf("failed to create logger: %v", err)
	}
	defer log.Close()

	mgr, err := NewManager(configPath, log)
	if err != nil {
		t.Fatalf("failed to create manager: %v", err)
	}
	defer mgr.Close()

	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		t.Error("config file should be created")
	}

	cfg := mgr.Get()
	if cfg == nil {
		t.Error("config should not be nil")
	}

	if cfg.Server.Port == 0 {
		t.Error("server port should have default value")
	}
}

func TestStripJSON5Comments(t *testing.T) {
	tests := []struct {
		name        string
		input       string
		contains    string
		notContains string
	}{
		{
			name:        "single line comment",
			input:       `{"key": "value" // comment}`,
			contains:    `"key": "value"`,
			notContains: "// comment",
		},
		{
			name:        "multi line comment",
			input:       `{"key": /* comment */ "value"}`,
			contains:    `"value"`,
			notContains: "/* comment */",
		},
		{
			name:        "trailing comma",
			input:       `{"key": "value",}`,
			contains:    `"key": "value"`,
			notContains: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := stripJSON5Comments(tt.input)
			if tt.contains != "" && !contains(result, tt.contains) {
				t.Errorf("stripJSON5Comments() should contain %q, got %v", tt.contains, result)
			}
			if tt.notContains != "" && contains(result, tt.notContains) {
				t.Errorf("stripJSON5Comments() should not contain %q, got %v", tt.notContains, result)
			}
		})
	}
}

func contains(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

func TestReplaceEnvVars(t *testing.T) {
	os.Setenv("TEST_KEY", "test_value")
	defer os.Unsetenv("TEST_KEY")

	log, _ := logger.New(logger.Config{Level: "error"})
	defer log.Close()

	tempDir := t.TempDir()
	configPath := filepath.Join(tempDir, "test_config.json5")

	configContent := `{
		"llm": {
			"provider": "openai",
			"apiKey": "${TEST_KEY}"
		}
	}`
	os.WriteFile(configPath, []byte(configContent), 0644)

	mgr, err := NewManager(configPath, log)
	if err != nil {
		t.Fatalf("failed to create manager: %v", err)
	}
	defer mgr.Close()

	cfg := mgr.Get()
	if cfg.LLM.APIKey != "test_value" {
		t.Errorf("apiKey should be replaced with env var, got: %s", cfg.LLM.APIKey)
	}
}
