package main

import (
	"bufio"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/HaohanHe/mujibot/internal/config"
	"github.com/HaohanHe/mujibot/internal/gateway"
	"github.com/HaohanHe/mujibot/internal/i18n"
	"github.com/HaohanHe/mujibot/internal/logger"
)

const (
	version = "1.0.0"
	appName = "Mujibot"
)

func main() {
	var (
		configPath  = flag.String("config", "./config.json5", "Path to configuration file")
		showVersion = flag.Bool("version", false, "Show version information")
		showHelp    = flag.Bool("help", false, "Show help information")
		skipSetup   = flag.Bool("skip-setup", false, "Skip initial setup wizard")
	)
	flag.Parse()

	if *showVersion {
		fmt.Printf("%s v%s\n", appName, version)
		os.Exit(0)
	}

	if *showHelp {
		printHelp()
		os.Exit(0)
	}

	fmt.Printf("%s v%s\n", appName, version)
	fmt.Println(strings.Repeat("=", 40))

	if !*skipSetup {
		if err := checkAndRunSetup(*configPath); err != nil {
			fmt.Fprintf(os.Stderr, "Setup failed: %v\n", err)
			os.Exit(1)
		}
	}

	fmt.Printf("Config: %s\n", *configPath)

	gw, err := gateway.NewGateway(*configPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to create gateway: %v\n", err)
		os.Exit(1)
	}

	if err := gw.Start(); err != nil {
		fmt.Fprintf(os.Stderr, "Failed to start gateway: %v\n", err)
		os.Exit(1)
	}
}

func checkAndRunSetup(configPath string) error {
	configExists := false
	if _, err := os.Stat(configPath); err == nil {
		configExists = true
	}

	if !configExists {
		return runSetupWizard(configPath)
	}

	log, err := logger.New(logger.Config{Level: "info"})
	if err != nil {
		return fmt.Errorf("failed to create logger: %w", err)
	}
	cfgMgr, err := config.NewManager(configPath, log)
	if err != nil {
		return runSetupWizard(configPath)
	}
	defer cfgMgr.Close()

	cfg := cfgMgr.Get()
	if cfg.Language.Current == "" || cfg.Language.Current == cfg.Language.Default {
		return runSetupWizard(configPath)
	}

	return nil
}

func runSetupWizard(configPath string) error {
	reader := bufio.NewReader(os.Stdin)

	printWelcome()

	fmt.Println("\nPlease select your language / 请选择您的语言 / 言語を選択してください:")
	fmt.Println()

	languages := i18n.SupportedLanguages()
	for i, lang := range languages {
		fmt.Printf("  %d. %s (%s)\n", i+1, i18n.LanguageName(lang), lang)
	}
	fmt.Println()

	var choice int
	for {
		fmt.Print("Enter [1-3]: ")
		input, _ := reader.ReadString('\n')
		input = strings.TrimSpace(input)

		if _, err := fmt.Sscanf(input, "%d", &choice); err == nil && choice >= 1 && choice <= len(languages) {
			break
		}
		fmt.Println("Invalid choice, please try again.")
	}

	selectedLang := languages[choice-1]

	fmt.Printf("\nSelected: %s\n\n", i18n.LanguageName(selectedLang))

	if err := createInitialConfig(configPath, selectedLang); err != nil {
		return fmt.Errorf("failed to create config: %w", err)
	}

	fmt.Println("Configuration created successfully!")
	return nil
}

func printWelcome() {
	fmt.Println()
	fmt.Println(strings.Repeat("=", 50))
	fmt.Println()
	fmt.Println("    Hello / 你好 / こんにちは / Hallo")
	fmt.Println("    Bonjour / Hola / Ciao / Olá")
	fmt.Println("    Привет / こんにちは / Merhaba / Hej")
	fmt.Println("    Salut / Namaste / Shalom / Aloha")
	fmt.Println()
	fmt.Println("    Welcome to Mujibot!")
	fmt.Println("    欢迎使用 Mujibot!")
	fmt.Println("    Mujibotへようこそ!")
	fmt.Println("    Willkommen bei Mujibot!")
	fmt.Println("    Bienvenue sur Mujibot!")
	fmt.Println()
	fmt.Println(strings.Repeat("=", 50))
}

func createInitialConfig(configPath, language string) error {
	dir := filepath.Dir(configPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}

	systemPrompt := getSystemPrompt(language)

	configContent := fmt.Sprintf(`{
  "server": {
    "port": 8080,
    "healthCheck": true
  },
  "channels": {
    "telegram": {
      "enabled": false,
      "token": "${TELEGRAM_BOT_TOKEN}",
      "allowedUsers": []
    },
    "discord": {
      "enabled": false,
      "token": "${DISCORD_BOT_TOKEN}",
      "allowedGuilds": []
    },
    "feishu": {
      "enabled": false,
      "appId": "${FEISHU_APP_ID}",
      "appSecret": "${FEISHU_APP_SECRET}",
      "encryptKey": "${FEISHU_ENCRYPT_KEY}",
      "allowedUsers": []
    }
  },
  "llm": {
    "provider": "openai",
    "model": "gpt-4o-mini",
    "apiKey": "${OPENAI_API_KEY}",
    "baseURL": "",
    "timeout": 60,
    "maxRetries": 3
  },
  "language": {
    "default": "%s",
    "current": "%s",
    "supported": ["en-US", "zh-CN", "ja-JP"]
  },
  "agents": {
    "default": {
      "name": "Mujibot",
      "systemPrompt": "%s",
      "tools": ["read_file", "write_file", "execute_command", "list_directory"]
    }
  },
  "tools": {
    "workDir": "/tmp/mujibot",
    "timeout": 30,
    "confirmDangerous": true,
    "allowedCommands": [],
    "blockedCommands": ["reboot", "shutdown", "init", "poweroff", "halt"],
    "enabledTools": {
      "read_file": true,
      "write_file": true,
      "list_directory": true,
      "execute_command": true,
      "web_search": true,
      "http_request": true,
      "weather": true,
      "ip_info": true,
      "exchange_rate": true,
      "memory_read": true,
      "memory_write": true
    },
    "customAPIs": []
  },
  "session": {
    "maxMessages": 20,
    "idleTimeout": 3600,
    "maxSessions": 100
  },
  "logging": {
    "level": "info",
    "file": "",
    "maxSize": 5,
    "format": "json"
  },
  "memory": {
    "enabled": true,
    "memoryDir": "./memory",
    "maxFileSize": 102400
  }
}`, language, language, systemPrompt)

	return os.WriteFile(configPath, []byte(configContent), 0644)
}

func getSystemPrompt(lang string) string {
	prompts := map[string]string{
		"en-US": "You are an AI assistant running on a low-power device. You are efficient, concise, and helpful.",
		"zh-CN": "你是一个运行在低功耗设备上的AI助手。你高效、简洁、乐于助人。",
		"ja-JP": "あなたは低電力デバイスで動作するAIアシスタントです。効率的で簡潔、そして親切です。",
	}
	if p, ok := prompts[lang]; ok {
		return p
	}
	return prompts["en-US"]
}

func printHelp() {
	fmt.Printf(`%s - Lightweight AI Assistant Gateway

Usage: mujibot [options]

Options:
  --config string    Path to configuration file (default "./config.json5")
  --version          Show version information
  --help             Show this help message
  --skip-setup       Skip initial setup wizard

Environment Variables:
  TELEGRAM_BOT_TOKEN    Telegram Bot API token
  DISCORD_BOT_TOKEN     Discord Bot API token
  FEISHU_APP_ID         Feishu App ID
  FEISHU_APP_SECRET     Feishu App Secret
  OPENAI_API_KEY        OpenAI API key
  ANTHROPIC_API_KEY     Anthropic API key

Examples:
  mujibot                          # Start with setup wizard
  mujibot --skip-setup             # Skip setup wizard
  mujibot --config /etc/mujibot/config.json5

Documentation: https://github.com/HaohanHe/mujibot
`, appName)
}
