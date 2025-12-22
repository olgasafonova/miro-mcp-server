#!/bin/sh
# Miro MCP Server Installer
# Usage: curl -fsSL https://raw.githubusercontent.com/olgasafonova/miro-mcp-server/main/install.sh | sh

set -e

REPO="olgasafonova/miro-mcp-server"
BINARY_NAME="miro-mcp-server"
INSTALL_DIR="${INSTALL_DIR:-/usr/local/bin}"

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

info() {
    printf "${GREEN}[INFO]${NC} %s\n" "$1"
}

warn() {
    printf "${YELLOW}[WARN]${NC} %s\n" "$1"
}

error() {
    printf "${RED}[ERROR]${NC} %s\n" "$1"
    exit 1
}

# Detect OS
detect_os() {
    OS="$(uname -s)"
    case "$OS" in
        Darwin) echo "darwin" ;;
        Linux) echo "linux" ;;
        MINGW*|MSYS*|CYGWIN*) echo "windows" ;;
        *) error "Unsupported operating system: $OS" ;;
    esac
}

# Detect architecture
detect_arch() {
    ARCH="$(uname -m)"
    case "$ARCH" in
        x86_64|amd64) echo "amd64" ;;
        arm64|aarch64) echo "arm64" ;;
        *) error "Unsupported architecture: $ARCH" ;;
    esac
}

# Get latest release version
get_latest_version() {
    curl -s "https://api.github.com/repos/${REPO}/releases/latest" | grep '"tag_name":' | sed -E 's/.*"([^"]+)".*/\1/'
}

# Main installation
main() {
    info "Miro MCP Server Installer"

    OS=$(detect_os)
    ARCH=$(detect_arch)
    VERSION="${VERSION:-$(get_latest_version)}"

    if [ -z "$VERSION" ]; then
        error "Could not determine latest version. Please set VERSION environment variable."
    fi

    info "Installing version: $VERSION"
    info "Platform: ${OS}-${ARCH}"

    # Build download URL
    SUFFIX="${OS}-${ARCH}"
    if [ "$OS" = "windows" ]; then
        SUFFIX="${OS}-${ARCH}.exe"
        BINARY_NAME="${BINARY_NAME}.exe"
    fi

    DOWNLOAD_URL="https://github.com/${REPO}/releases/download/${VERSION}/${BINARY_NAME}-${SUFFIX}"

    # Create temp directory
    TMP_DIR=$(mktemp -d)
    trap "rm -rf $TMP_DIR" EXIT

    # Download binary
    info "Downloading from: $DOWNLOAD_URL"
    if ! curl -fsSL "$DOWNLOAD_URL" -o "$TMP_DIR/$BINARY_NAME"; then
        error "Failed to download binary. Check that version $VERSION exists."
    fi

    # Make executable
    chmod +x "$TMP_DIR/$BINARY_NAME"

    # Install
    if [ -w "$INSTALL_DIR" ]; then
        mv "$TMP_DIR/$BINARY_NAME" "$INSTALL_DIR/$BINARY_NAME"
    else
        info "Installing to $INSTALL_DIR requires sudo..."
        sudo mv "$TMP_DIR/$BINARY_NAME" "$INSTALL_DIR/$BINARY_NAME"
    fi

    info "Installed $BINARY_NAME to $INSTALL_DIR"

    # Verify installation
    if command -v "$BINARY_NAME" >/dev/null 2>&1; then
        info "Installation successful!"
    else
        warn "$INSTALL_DIR may not be in your PATH. Add it with:"
        warn "  export PATH=\"\$PATH:$INSTALL_DIR\""
    fi

    echo ""
    info "Next steps:"
    echo "  1. Get a Miro access token from: https://miro.com/app/settings/user-profile/apps"
    echo "  2. Set environment variable: export MIRO_ACCESS_TOKEN=\"your_token\""
    echo "  3. Configure Claude Code: claude mcp add miro-mcp-server -- miro-mcp-server"
    echo ""
    info "Documentation: https://github.com/${REPO}#readme"
}

main "$@"
