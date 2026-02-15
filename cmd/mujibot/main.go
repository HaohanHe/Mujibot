package main

import (
	"flag"
	"fmt"
	"os"
	"runtime"

	"github.com/HaohanHe/mujibot/internal/gateway"
)

const (
	version = "1.0.0"
	appName = "Mujibot"
)

func main() {
	// 解析命令行参数
	var (
		configPath = flag.String("config", "./config.json5", "Path to configuration file")
		showVersion = flag.Bool("version", false, "Show version information")
		showHelp    = flag.Bool("help", false, "Show help information")
	)
	flag.Parse()

	// 显示版本
	if *showVersion {
		fmt.Printf("%s v%s\n", appName, version)
		fmt.Printf("Go version: %s\n", runtime.Version())
		fmt.Printf("OS/Arch: %s/%s\n", runtime.GOOS, runtime.GOARCH)
		os.Exit(0)
	}

	// 显示帮助
	if *showHelp {
		printHelp()
		os.Exit(0)
	}

	// 打印启动信息
	fmt.Printf("🤖 %s v%s starting...\n", appName, version)
	fmt.Printf("📄 Config: %s\n", *configPath)

	// 创建并启动网关
	gw, err := gateway.NewGateway(*configPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "❌ Failed to create gateway: %v\n", err)
		os.Exit(1)
	}

	if err := gw.Start(); err != nil {
		fmt.Fprintf(os.Stderr, "❌ Failed to start gateway: %v\n", err)
		os.Exit(1)
	}
}

// printHelp 打印帮助信息
func printHelp() {
	fmt.Printf(`%s - Lightweight AI Assistant Gateway

Usage: mujibot [options]

Options:
  --config string    Path to configuration file (default "./config.json5")
  --version          Show version information
  --help             Show this help message

Environment Variables:
  TELEGRAM_BOT_TOKEN    Telegram Bot API token
  DISCORD_BOT_TOKEN     Discord Bot API token
  FEISHU_APP_ID         Feishu App ID
  FEISHU_APP_SECRET     Feishu App Secret
  OPENAI_API_KEY        OpenAI API key
  ANTHROPIC_API_KEY     Anthropic API key

Examples:
  # Start with default config
  mujibot

  # Start with custom config
  mujibot --config /etc/mujibot/config.json5

  # Show version
  mujibot --version

Documentation: https://github.com/HaohanHe/mujibot
`, appName)
}
