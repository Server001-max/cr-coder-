#!/bin/bash
# cr-coder installer script
# Usage: curl -fsSL https://raw.githubusercontent.com/Server001-max/cr-coder/main/scripts/install.sh | sh

set -e

REPO="Server001-max/cr-coder"
BINARY_NAME="cr-coder"
INSTALL_DIR="${INSTALL_DIR:-$HOME/.local/bin}"

# Colors
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

log_info() { echo -e "${BLUE}[INFO]${NC} $*"; }
log_success() { echo -e "${GREEN}[SUCCESS]${NC} $*"; }
log_warn() { echo -e "${YELLOW}[WARN]${NC} $*"; }
log_error() { echo -e "${RED}[ERROR]${NC} $*"; }

# Detect OS and Architecture
detect_platform() {
    OS="$(uname -s | tr '[:upper:]' '[:lower:]')"
    ARCH="$(uname -m)"

    case "$OS" in
        linux) OS="linux" ;;
        darwin) OS="darwin" ;;
        mingw*|msys*|cygwin*) OS="windows" ;;
        *) log_error "Unsupported OS: $OS"; exit 1 ;;
    esac

    case "$ARCH" in
        x86_64|amd64) ARCH="amd64" ;;
        arm64|aarch64) ARCH="arm64" ;;
        *) log_error "Unsupported architecture: $ARCH"; exit 1 ;;
    esac

    # Windows uses .exe
    if [ "$OS" = "windows" ]; then
        BINARY_NAME="${BINARY_NAME}.exe"
    fi

    PLATFORM="${OS}-${ARCH}"
    log_info "Detected platform: $PLATFORM"
}

# Get latest release version
get_latest_version() {
    log_info "Fetching latest release..."
    VERSION=$(curl -fsSL "https://api.github.com/repos/${REPO}/releases/latest" | grep '"tag_name":' | sed -E 's/.*"([^"]+)".*/\1/')
    if [ -z "$VERSION" ]; then
        log_error "Failed to get latest version"
        exit 1
    fi
    log_info "Latest version: $VERSION"
}

# Download and install
install_binary() {
    local url="https://github.com/${REPO}/releases/download/${VERSION}/${BINARY_NAME}_${VERSION}_${PLATFORM}"
    local archive_ext="tar.gz"
    local temp_dir=$(mktemp -d)

    if [ "$OS" = "windows" ]; then
        archive_ext="zip"
        url="${url}.zip"
    else
        url="${url}.tar.gz"
    fi

    log_info "Downloading from: $url"
    cd "$temp_dir"

    if command -v curl >/dev/null 2>&1; then
        curl -fsSL "$url" -o "archive.${archive_ext}"
    elif command -v wget >/dev/null 2>&1; then
        wget -q "$url" -O "archive.${archive_ext}"
    else
        log_error "Neither curl nor wget found"
        exit 1
    fi

    log_info "Extracting..."
    if [ "$archive_ext" = "zip" ]; then
        unzip -q "archive.${archive_ext}"
    else
        tar -xzf "archive.${archive_ext}"
    fi

    # Find binary
    local binary_path=$(find . -name "$BINARY_NAME" -type f | head -1)
    if [ -z "$binary_path" ]; then
        log_error "Binary not found in archive"
        exit 1
    fi

    # Create install directory
    mkdir -p "$INSTALL_DIR"

    # Install binary
    log_info "Installing to $INSTALL_DIR/$BINARY_NAME"
    cp "$binary_path" "$INSTALL_DIR/$BINARY_NAME"
    chmod +x "$INSTALL_DIR/$BINARY_NAME"

    # Cleanup
    cd /
    rm -rf "$temp_dir"

    log_success "cr-coder installed successfully!"
}

# Verify installation
verify_installation() {
    if command -v "$BINARY_NAME" >/dev/null 2>&1; then
        log_success "cr-coder is available in PATH"
        "$BINARY_NAME" version
    elif [ -x "$INSTALL_DIR/$BINARY_NAME" ]; then
        log_warn "cr-coder installed but $INSTALL_DIR is not in PATH"
        log_info "Add this to your shell profile (~/.bashrc, ~/.zshrc, etc.):"
        echo "    export PATH=\"\$PATH:$INSTALL_DIR\""
        log_info "Then run: source ~/.bashrc (or restart your terminal)"
    else
        log_error "Installation verification failed"
        exit 1
    fi
}

# Main
main() {
    echo "🚀 CR CODER Installer"
    echo "====================="
    echo ""

    detect_platform
    get_latest_version
    install_binary
    verify_installation

    echo ""
    log_info "Next steps:"
    echo "  1. Run 'cr-coder init' to download the default AI model"
    echo "  2. Run 'cr-coder chat \"hello\"' to start chatting"
    echo "  3. Run 'cr-coder agent \"task\"' to run the coding agent"
    echo ""
    log_info "Note: You need Ollama installed for local AI models."
    log_info "Install from: https://ollama.ai"
}

main "$@"