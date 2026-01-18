#!/bin/bash
set -e

# muxie installer script
# Usage: curl -fsSL https://raw.githubusercontent.com/MSmaili/muxie/main/install.sh | bash

VERSION="${MUXIE_VERSION:-latest}"
INSTALL_DIR="${MUXIE_INSTALL_DIR:-/usr/local/bin}"
REPO="MSmaili/muxie"

# Colors
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

info() {
    echo -e "${GREEN}==>${NC} $1"
}

warn() {
    echo -e "${YELLOW}Warning:${NC} $1"
}

error() {
    echo -e "${RED}Error:${NC} $1" >&2
    exit 1
}

# Detect OS and architecture
detect_platform() {
    OS=$(uname -s | tr '[:upper:]' '[:lower:]')
    ARCH=$(uname -m)

    case "$OS" in
        linux) OS="linux" ;;
        darwin) OS="darwin" ;;
        *) error "Unsupported OS: $OS" ;;
    esac

    case "$ARCH" in
        x86_64) ARCH="amd64" ;;
        aarch64|arm64) ARCH="arm64" ;;
        *) error "Unsupported architecture: $ARCH" ;;
    esac

    info "Detected platform: $OS/$ARCH"
}

# Check if Go is installed
has_go() {
    command -v go >/dev/null 2>&1
}

# Install from source using Go
install_from_source() {
    info "Installing from source..."

    if ! has_go; then
        error "Go is not installed. Please install Go or use a pre-built binary."
    fi

    TEMP_DIR=$(mktemp -d)
    trap "rm -rf $TEMP_DIR" EXIT

    info "Cloning repository..."
    git clone --depth 1 https://github.com/$REPO.git "$TEMP_DIR" >/dev/null 2>&1 || \
        error "Failed to clone repository"

    cd "$TEMP_DIR"

    info "Building muxie..."
    go build -o muxie . || error "Build failed"

    info "Installing to $INSTALL_DIR..."
    sudo mv muxie "$INSTALL_DIR/muxie" || error "Failed to install (try with sudo)"
    sudo chmod +x "$INSTALL_DIR/muxie"
}

# Install from GitHub releases
install_from_release() {
    info "Downloading muxie $VERSION for $OS/$ARCH..."

    if [ "$VERSION" = "latest" ]; then
        DOWNLOAD_URL="https://github.com/$REPO/releases/latest/download/muxie-$OS-$ARCH"
    else
        DOWNLOAD_URL="https://github.com/$REPO/releases/download/$VERSION/muxie-$OS-$ARCH"
    fi

    TEMP_FILE=$(mktemp)
    trap "rm -f $TEMP_FILE" EXIT

    if curl -fsSL "$DOWNLOAD_URL" -o "$TEMP_FILE" 2>/dev/null; then
        info "Installing to $INSTALL_DIR..."
        sudo mv "$TEMP_FILE" "$INSTALL_DIR/muxie" || error "Failed to install (try with sudo)"
        sudo chmod +x "$INSTALL_DIR/muxie"
    else
        warn "No pre-built binary found, installing from source..."
        install_from_source
    fi
}

# Check if tmux is installed
check_tmux() {
    if ! command -v tmux >/dev/null 2>&1; then
        warn "tmux is not installed. muxie requires tmux to function."
        echo "Install tmux:"
        echo "  macOS:  brew install tmux"
        echo "  Ubuntu: sudo apt install tmux"
    fi
}

# Main installation
main() {
    detect_platform
    check_tmux

    # Check if forced to install from source
    if [ -n "$MUXIE_FROM_SOURCE" ]; then
        install_from_source
    else
        install_from_release
    fi

    # Verify installation
    if command -v muxie >/dev/null 2>&1; then
        info "Successfully installed muxie!"
        echo ""
        muxie --version
        echo ""
        echo "Get started:"
        echo "  muxie start <workspace>    # Start a workspace"
        echo "  muxie save                 # Save current session"
        echo "  muxie list sessions        # List tmux sessions"
        echo ""
        echo "For more info: muxie --help"
    else
        error "Installation failed"
    fi
}

main
