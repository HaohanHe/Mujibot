#!/bin/bash

# Mujibot 卸载脚本

set -e

APP_NAME="mujibot"
INSTALL_DIR="/opt/mujibot"
BIN_DIR="/usr/local/bin"
LOG_DIR="/var/log/mujibot"

# 颜色输出
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m'

info() {
    echo -e "${BLUE}[INFO]${NC} $1"
}

success() {
    echo -e "${GREEN}[SUCCESS]${NC} $1"
}

warn() {
    echo -e "${YELLOW}[WARN]${NC} $1"
}

error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

# 检查root权限
check_root() {
    if [ "$EUID" -ne 0 ]; then
        error "Please run as root (use sudo)"
        exit 1
    fi
}

# 停止并禁用服务
stop_service() {
    info "Stopping Mujibot service..."
    if systemctl is-active --quiet $APP_NAME 2>/dev/null; then
        systemctl stop $APP_NAME
        success "Service stopped"
    else
        warn "Service not running"
    fi

    if systemctl is-enabled --quiet $APP_NAME 2>/dev/null; then
        systemctl disable $APP_NAME
        success "Service disabled"
    fi
}

# 删除systemd服务文件
remove_service() {
    info "Removing systemd service..."
    if [ -f "/etc/systemd/system/${APP_NAME}.service" ]; then
        rm -f "/etc/systemd/system/${APP_NAME}.service"
        systemctl daemon-reload
        success "Systemd service removed"
    else
        warn "Service file not found"
    fi
}

# 删除二进制文件
remove_binary() {
    info "Removing binary..."
    if [ -f "$BIN_DIR/$APP_NAME" ]; then
        rm -f "$BIN_DIR/$APP_NAME"
        success "Binary removed"
    else
        warn "Binary not found"
    fi
}

# 删除安装目录
remove_install_dir() {
    info "Removing installation directory..."
    if [ -d "$INSTALL_DIR" ]; then
        # 询问是否保留配置文件
        read -p "Keep configuration files? (y/N) " -n 1 -r
        echo
        if [[ $REPLY =~ ^[Yy]$ ]]; then
            # 备份配置文件
            BACKUP_DIR="$HOME/${APP_NAME}-backup-$(date +%Y%m%d%H%M%S)"
            mkdir -p "$BACKUP_DIR"
            cp -r "$INSTALL_DIR"/* "$BACKUP_DIR/" 2>/dev/null || true
            success "Config backed up to $BACKUP_DIR"
        fi
        rm -rf "$INSTALL_DIR"
        success "Installation directory removed"
    else
        warn "Installation directory not found"
    fi
}

# 删除日志目录
remove_log_dir() {
    info "Removing log directory..."
    if [ -d "$LOG_DIR" ]; then
        read -p "Remove log files? (y/N) " -n 1 -r
        echo
        if [[ $REPLY =~ ^[Yy]$ ]]; then
            rm -rf "$LOG_DIR"
            success "Log directory removed"
        else
            warn "Log directory kept at $LOG_DIR"
        fi
    else
        warn "Log directory not found"
    fi
}

# 删除用户
remove_user() {
    info "Removing user..."
    if id -u $APP_NAME &>/dev/null; then
        userdel $APP_NAME 2>/dev/null || true
        success "User removed"
    else
        warn "User not found"
    fi
}

# 清理残留
cleanup() {
    info "Cleaning up..."
    # 清理可能的残留文件
    rm -rf /tmp/mujibot*
    success "Cleanup complete"
}

# 主函数
main() {
    echo "╔══════════════════════════════════════════════════════════════╗"
    echo "║              Mujibot Uninstaller v1.0.1                     ║"
    echo "╚══════════════════════════════════════════════════════════════╝"
    echo ""

    check_root

    # 确认卸载
    read -p "Are you sure you want to uninstall Mujibot? (y/N) " -n 1 -r
    echo
    if [[ ! $REPLY =~ ^[Yy]$ ]]; then
        info "Uninstall cancelled"
        exit 0
    fi

    stop_service
    remove_service
    remove_binary
    remove_install_dir
    remove_log_dir
    remove_user
    cleanup

    echo ""
    echo "╔══════════════════════════════════════════════════════════════╗"
    echo "║              Mujibot Uninstall Complete                      ║"
    echo "╚══════════════════════════════════════════════════════════════╝"
    echo ""
    echo "Mujibot has been completely removed from your system."
    echo ""
}

main "$@"
