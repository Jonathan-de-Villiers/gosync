#!/bin/bash

# GoSync Installation Script
# This script installs GoSync from GitHub releases

set -euo pipefail

# Colors
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m'

# Configuration
REPO="Jonathan-de-Villiers/gosync"
INSTALL_DIR="/usr/local/bin"
VERSION="${1:-latest}"

# Functions
log_info() {
    echo -e "${BLUE}[INFO]${NC} $1"
}

log_success() {
    echo -e "${GREEN}[SUCCESS]${NC} $1"
}

log_warning() {
    echo -e "${YELLOW}[WARNING]${NC} $1"
}

log_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

# Detect OS and architecture
detect_os_arch() {
    local os=$(uname -s | tr '[:upper:]' '[:lower:]')
    local arch=$(uname -m)
    
    case $os in
        linux|darwin) ;;
        *) log_error "Unsupported OS: $os"; exit 1 ;;
    esac
    
    case $arch in
        x86_64) arch="amd64" ;;
        aarch64|arm64) arch="arm64" ;;
        *) log_error "Unsupported architecture: $arch"; exit 1 ;;
    esac
    
    echo "${os}-${arch}"
}

# Get latest version
get_latest_version() {
    if command -v curl >/dev/null 2>&1; then
        curl -s "https://api.github.com/repos/${REPO}/releases/latest" | grep -o '"tag_name": *"[^"]*"' | cut -d'"' -f4
    elif command -v wget >/dev/null 2>&1; then
        wget -qO- "https://api.github.com/repos/${REPO}/releases/latest" | grep -o '"tag_name": *"[^"]*"' | cut -d'"' -f4
    else
        log_error "curl or wget is required"
        exit 1
    fi
}

# Download and install GoSync
install_gosync() {
    local version="$1"
    local os_arch="$2"
    
    if [[ "$version" == "latest" ]]; then
        version=$(get_latest_version)
        if [[ -z "$version" ]]; then
            log_error "Failed to get latest version"
            exit 1
        fi
    fi
    
    local filename="gosync-${version}-${os_arch}"
    if [[ "$os_arch" == windows-* ]]; then
        filename="${filename}.zip"
        extract_cmd="unzip"
    else
        filename="${filename}.tar.gz"
        extract_cmd="tar -xzf"
    fi
    
    local download_url="https://github.com/${REPO}/releases/download/${version}/${filename}"
    local temp_dir=$(mktemp -d)
    
    log_info "Downloading GoSync ${version} for ${os_arch}..."
    
    # Download
    if command -v curl >/dev/null 2>&1; then
        curl -L -o "${temp_dir}/${filename}" "$download_url"
    elif command -v wget >/dev/null 2>&1; then
        wget -O "${temp_dir}/${filename}" "$download_url"
    else
        log_error "curl or wget is required"
        exit 1
    fi
    
    # Extract
    log_info "Extracting..."
    cd "$temp_dir"
    $extract_cmd "$filename"
    
    # Install
    local binary_name="gosync"
    if [[ "$os_arch" == windows-* ]]; then
        binary_name="gosync.exe"
    fi
    
    if [[ -w "$INSTALL_DIR" ]]; then
        mv "$binary_name" "$INSTALL_DIR/gosync"
    else
        log_info "Using sudo to install to $INSTALL_DIR"
        sudo mv "$binary_name" "$INSTALL_DIR/gosync"
    fi
    
    # Cleanup
    cd /
    rm -rf "$temp_dir"
    
    log_success "GoSync ${version} installed successfully!"
}

# Check if GoSync is already installed
check_existing() {
    if command -v gosync >/dev/null 2>&1; then
        local existing_version=$(gosync --version 2>/dev/null | head -n1 || echo "unknown")
        log_warning "GoSync is already installed: $existing_version"
        read -p "Do you want to continue and overwrite? (y/N): " -n 1 -r
        echo
        if [[ ! $REPLY =~ ^[Yy]$ ]]; then
            log_info "Installation cancelled"
            exit 0
        fi
    fi
}

# Main installation
main() {
    log_info "GoSync Installation Script"
    log_info "Repository: ${REPO}"
    log_info "Version: ${VERSION}"
    
    # Check dependencies
    if ! command -v curl >/dev/null 2>&1 && ! command -v wget >/dev/null 2>&1; then
        log_error "curl or wget is required"
        exit 1
    fi
    
    # Check existing installation
    check_existing
    
    # Detect OS and architecture
    local os_arch=$(detect_os_arch)
    log_info "Detected platform: ${os_arch}"
    
    # Install
    install_gosync "$VERSION" "$os_arch"
    
    # Verify installation
    if command -v gosync >/dev/null 2>&1; then
        log_success "Installation verified! Run 'gosync --help' to get started."
    else
        log_error "Installation verification failed"
        exit 1
    fi
}

# Run main function
main "$@"
