#!/bin/bash

# Mujibot 安装脚本
# 支持玩客云、树莓派等ARM设备

set -e

APP_NAME="mujibot"
VERSION="1.0.1"
INSTALL_DIR="/opt/mujibot"
BIN_DIR="/usr/local/bin"
CONFIG_FILE="$INSTALL_DIR/config.json5"
LOG_DIR="/var/log/mujibot"

# 颜色输出
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# 打印信息
info() {
    echo -e "${BLUE}[INFO]${NC} $1"
}

# 打印成功
success() {
    echo -e "${GREEN}[SUCCESS]${NC} $1"
}

# 打印警告
warn() {
    echo -e "${YELLOW}[WARN]${NC} $1"
}

# 打印错误
error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

# 检测架构
detect_arch() {
    local arch=$(uname -m)
    case $arch in
        x86_64)
            echo "amd64"
            ;;
        aarch64|arm64)
            echo "arm64"
            ;;
        armv7l|armv7)
            echo "armv7"
            ;;
        *)
            error "Unsupported architecture: $arch"
            exit 1
            ;;
    esac
}

# 检测系统
detect_os() {
    if [ -f /etc/os-release ]; then
        . /etc/os-release
        echo $ID
    else
        echo "unknown"
    fi
}

# 创建用户和组
create_user() {
    if ! id -u $APP_NAME &>/dev/null; then
        info "Creating user: $APP_NAME"
        useradd --system --no-create-home --shell /bin/false $APP_NAME
    else
        info "User $APP_NAME already exists"
    fi
}

# 创建目录
create_directories() {
    info "Creating directories..."
    
    mkdir -p $INSTALL_DIR
    mkdir -p $LOG_DIR
    
    chown -R $APP_NAME:$APP_NAME $INSTALL_DIR
    chown -R $APP_NAME:$APP_NAME $LOG_DIR
    chmod 755 $INSTALL_DIR
    chmod 755 $LOG_DIR
}

# 下载二进制文件
download_binary() {
    local arch=$1
    local download_url="https://github.com/HaohanHe/mujibot/releases/download/v$VERSION/${APP_NAME}-${VERSION}-linux-${arch}.tar.gz"
    local temp_dir=$(mktemp -d)
    
    info "Downloading Mujibot for $arch..."
    info "URL: $download_url"
    
    if command -v curl &> /dev/null; then
        curl -L -o "$temp_dir/mujibot.tar.gz" "$download_url" || {
            error "Failed to download binary"
            rm -rf $temp_dir
            exit 1
        }
    elif command -v wget &> /dev/null; then
        wget -O "$temp_dir/mujibot.tar.gz" "$download_url" || {
            error "Failed to download binary"
            rm -rf $temp_dir
            exit 1
        }
    else
        error "curl or wget is required"
        exit 1
    fi
    
    info "Extracting binary..."
    tar -xzf "$temp_dir/mujibot.tar.gz" -C $temp_dir
    
    # 复制二进制文件
    cp "$temp_dir/${APP_NAME}-${VERSION}-linux-${arch}/$APP_NAME" $BIN_DIR/
    chmod +x $BIN_DIR/$APP_NAME
    
    # 复制配置文件示例
    if [ -f "$temp_dir/${APP_NAME}-${VERSION}-linux-${arch}/config.json5" ]; then
        if [ ! -f "$CONFIG_FILE" ]; then
            cp "$temp_dir/${APP_NAME}-${VERSION}-linux-${arch}/config.json5" $CONFIG_FILE
            success "Config file created at $CONFIG_FILE"
        else
            warn "Config file already exists, skipping"
        fi
    fi
    
    rm -rf $temp_dir
    success "Binary installed to $BIN_DIR/$APP_NAME"
}

# 本地安装（如果二进制文件存在）
local_install() {
    local arch=$1
    local build_dir="./build"
    
    if [ -f "$build_dir/${APP_NAME}-${arch}" ]; then
        info "Installing local build..."
        cp "$build_dir/${APP_NAME}-${arch}" $BIN_DIR/$APP_NAME
        chmod +x $BIN_DIR/$APP_NAME
        success "Binary installed from local build"
    elif [ -f "$build_dir/$APP_NAME" ]; then
        info "Installing local build..."
        cp "$build_dir/$APP_NAME" $BIN_DIR/$APP_NAME
        chmod +x $BIN_DIR/$APP_NAME
        success "Binary installed from local build"
    else
        return 1
    fi
}

# 安装systemd服务
install_service() {
    info "Installing systemd service..."
    
    cat > /etc/systemd/system/${APP_NAME}.service << 'EOF'
[Unit]
Description=Mujibot Lightweight AI Assistant
After=network.target

[Service]
Type=simple
User=mujibot
Group=mujibot
WorkingDirectory=/opt/mujibot
ExecStart=/usr/local/bin/mujibot --config /opt/mujibot/config.json5
Restart=always
RestartSec=5
MemoryMax=100M
StandardOutput=journal
StandardError=journal

[Install]
WantedBy=multi-user.target
EOF
    
    systemctl daemon-reload
    success "Systemd service installed"
}

# 配置环境变量提示
configure_env() {
    info "Configuration required:"
    echo ""
    echo "Please set the following environment variables in $CONFIG_FILE:"
    echo ""
    echo "  # Telegram (optional)"
    echo "  export TELEGRAM_BOT_TOKEN='your_telegram_bot_token'"
    echo ""
    echo "  # Discord (optional)"
    echo "  export DISCORD_BOT_TOKEN='your_discord_bot_token'"
    echo ""
    echo "  # Feishu (optional)"
    echo "  export FEISHU_APP_ID='your_feishu_app_id'"
    echo "  export FEISHU_APP_SECRET='your_feishu_app_secret'"
    echo ""
    echo "  # LLM (required)"
    echo "  export OPENAI_API_KEY='your_openai_api_key'"
    echo ""
    echo "Or edit $CONFIG_FILE directly."
    echo ""
}

# 启动服务
start_service() {
    info "Starting Mujibot service..."
    systemctl enable $APP_NAME
    systemctl start $APP_NAME
    
    sleep 2
    
    if systemctl is-active --quiet $APP_NAME; then
        success "Mujibot service is running!"
        info "View logs: journalctl -u $APP_NAME -f"
        info "Web console: http://localhost:8080"
    else
        error "Failed to start Mujibot service"
        info "Check logs: journalctl -u $APP_NAME -n 50"
        exit 1
    fi
}

# 打印完成信息
print_finish() {
    local ip_address=$(hostname -I | awk '{print $1}')
    local hostname=$(hostname)
    
    echo ""
    echo "╔══════════════════════════════════════════════════════════════╗"
    echo "║                   Mujibot Installation Complete              ║"
    echo "╚══════════════════════════════════════════════════════════════╝"
    echo ""
    echo "  📁 Installation directory: $INSTALL_DIR"
    echo "  ⚙️  Configuration file: $CONFIG_FILE"
    echo "  📜 Log directory: $LOG_DIR"
    echo ""
    echo "  🌐 Web Console Access:"
    echo "     • Local:   http://localhost:8080"
    echo "     • LAN:     http://$ip_address:8080"
    echo "     • Host:    http://$hostname:8080"
    echo ""
    echo "  🔧 Debug Commands:"
    echo "    sudo systemctl start|stop|restart|status $APP_NAME"
    echo "    sudo journalctl -u $APP_NAME -f"
    echo "    $APP_NAME --version"
    echo ""
    echo "  📝 Next Steps:"
    echo "    1. Open http://$ip_address:8080 in your browser"
    echo "    2. Or edit $CONFIG_FILE to configure your API keys"
    echo "    3. Restart: sudo systemctl restart $APP_NAME"
    echo ""
    echo "  💡 Tips:"
    echo "    • Check status: sudo systemctl status $APP_NAME"
    echo "    • View logs:    sudo journalctl -u $APP_NAME -n 50"
    echo ""
}

# 主函数
main() {
    echo "╔══════════════════════════════════════════════════════════════╗"
    echo "║              Mujibot Installer v$VERSION                       ║"
    echo "╚══════════════════════════════════════════════════════════════╝"
    echo ""
    
    # 检查root权限
    if [ "$EUID" -ne 0 ]; then
        error "Please run as root (use sudo)"
        exit 1
    fi
    
    # 检测架构
    ARCH=$(detect_arch)
    info "Detected architecture: $ARCH"
    
    # 检测操作系统
    OS=$(detect_os)
    info "Detected OS: $OS"
    
    # 创建用户
    create_user
    
    # 创建目录
    create_directories
    
    # 安装二进制文件
    if ! local_install $ARCH; then
        download_binary $ARCH
    fi
    
    # 安装服务
    install_service
    
    # 配置提示
    configure_env
    
    # 询问是否启动
    read -p "Start Mujibot service now? (y/N) " -n 1 -r
    echo
    if [[ $REPLY =~ ^[Yy]$ ]]; then
        start_service
    else
        info "You can start the service later with:"
        info "  sudo systemctl enable --now $APP_NAME"
    fi
    
    # 打印完成信息
    print_finish
}

# 运行主函数
main "$@"
