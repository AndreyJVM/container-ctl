#!/bin/sh
# container-ctl installer script
# Automatically detects OS and CPU architecture, downloads the release from GitHub,
# and installs the standalone container-ctl binary.
#
# Usage:
#   wget -qO- https://raw.githubusercontent.com/AndreyJVM/container-ctl/main/install.sh | sudo bash
#   curl -fsSL https://raw.githubusercontent.com/AndreyJVM/container-ctl/main/install.sh | sudo bash
#
# Environment variables:
#   VERSION: Specify version to install (e.g. "0.22.7" or "v0.22.7", defaults to latest)
#   BIN_DIR: Directory to install container-ctl into (defaults to /usr/local/bin or ~/.local/bin)

set -e

REPO="AndreyJVM/container-ctl"

# Detect Operating System
OS="$(uname -s | tr '[:upper:]' '[:lower:]')"
case "$OS" in
  linux*)   OS="linux" ;;
  darwin*)  OS="darwin" ;;
  freebsd*) OS="freebsd" ;;
  *)
    echo "Error: Unsupported operating system: $OS" >&2
    exit 1
    ;;
esac

# Detect Architecture
ARCH="$(uname -m)"
case "$ARCH" in
  x86_64|amd64)   ARCH="amd64" ;;
  aarch64|arm64)  ARCH="arm64" ;;
  *)
    echo "Error: Unsupported architecture: $ARCH" >&2
    exit 1
    ;;
esac

# Verify FreeBSD arm64 is not attempted (GoReleaser ignores freebsd/arm64)
if [ "$OS" = "freebsd" ] && [ "$ARCH" = "arm64" ]; then
  echo "Error: Pre-built FreeBSD binaries are only available for amd64." >&2
  exit 1
fi

# Helper functions for downloading
download_file() {
  url="$1"
  dest="$2"
  if command -v wget >/dev/null 2>&1; then
    wget -q --show-progress -O "$dest" "$url"
  elif command -v curl >/dev/null 2>&1; then
    curl -fL --progress-bar -o "$dest" "$url"
  else
    echo "Error: Neither wget nor curl found. Please install wget or curl." >&2
    exit 1
  fi
}

# Determine target version
if [ -z "$VERSION" ]; then
  echo "Determining latest release of container-ctl..."
  TAG=""
  
  # Try via GitHub redirect headers (avoids GitHub API rate limits)
  if command -v wget >/dev/null 2>&1; then
    TAG="$(wget --spider -S "https://github.com/${REPO}/releases/latest" 2>&1 | grep -i "Location:.*tag" | head -n 1 | awk -F'/' '{print $NF}' | tr -d '\r ')"
  elif command -v curl >/dev/null 2>&1; then
    TAG="$(curl -fsSL -o /dev/null -w '%{url_effective}' "https://github.com/${REPO}/releases/latest" | awk -F'/' '{print $NF}' | tr -d '\r ')"
  fi

  # Fallback to GitHub Releases API if needed
  if [ -z "$TAG" ] || [ "$TAG" = "latest" ]; then
    if command -v wget >/dev/null 2>&1; then
      TAG="$(wget -qO- "https://api.github.com/repos/${REPO}/releases/latest" 2>/dev/null | grep '"tag_name":' | head -n 1 | sed -E 's/.*"([^"]+)".*/\1/' | tr -d '\r ')"
    elif command -v curl >/dev/null 2>&1; then
      TAG="$(curl -fsSL "https://api.github.com/repos/${REPO}/releases/latest" 2>/dev/null | grep '"tag_name":' | head -n 1 | sed -E 's/.*"([^"]+)".*/\1/' | tr -d '\r ')"
    fi
  fi

  if [ -z "$TAG" ] || [ "$TAG" = "latest" ]; then
    echo "Error: Failed to determine latest release tag. Please specify VERSION=vX.Y.Z manually." >&2
    exit 1
  fi
else
  case "$VERSION" in
    v*) TAG="$VERSION" ;;
    *)  TAG="v$VERSION" ;;
  esac
fi

CLEAN_VERSION="${TAG#v}"
echo "Selected release: ${TAG} (${OS}/${ARCH})"

# Determine target installation directory
if [ -n "$BIN_DIR" ]; then
  INSTALL_DIR="$BIN_DIR"
elif [ "$(id -u)" -eq 0 ]; then
  INSTALL_DIR="/usr/local/bin"
elif [ -d "$HOME/.local/bin" ] && echo ":$PATH:" | grep -q ":$HOME/.local/bin:"; then
  INSTALL_DIR="$HOME/.local/bin"
else
  INSTALL_DIR="/usr/local/bin"
fi

TMP_DIR="$(mktemp -d 2>/dev/null || mktemp -d -t 'container-ctl')"
trap 'rm -rf "$TMP_DIR"' EXIT INT TERM

ARCHIVE_NAME="container-ctl_${CLEAN_VERSION}_${OS}_${ARCH}.tar.gz"
DOWNLOAD_URL="https://github.com/${REPO}/releases/download/${TAG}/${ARCHIVE_NAME}"

echo "Downloading ${DOWNLOAD_URL}..."
download_file "$DOWNLOAD_URL" "${TMP_DIR}/${ARCHIVE_NAME}"

echo "Extracting binary..."
tar -xzf "${TMP_DIR}/${ARCHIVE_NAME}" -C "$TMP_DIR" container-ctl

# Ensure installation directory exists
if [ ! -d "$INSTALL_DIR" ]; then
  if [ -w "$(dirname "$INSTALL_DIR")" ]; then
    mkdir -p "$INSTALL_DIR"
  elif command -v sudo >/dev/null 2>&1; then
    sudo mkdir -p "$INSTALL_DIR"
  fi
fi

# Move binary to target location
echo "Installing to ${INSTALL_DIR}/container-ctl..."
if [ -w "$INSTALL_DIR" ]; then
  mv -f "${TMP_DIR}/container-ctl" "${INSTALL_DIR}/container-ctl"
  chmod +x "${INSTALL_DIR}/container-ctl"
else
  if command -v sudo >/dev/null 2>&1; then
    sudo mv -f "${TMP_DIR}/container-ctl" "${INSTALL_DIR}/container-ctl"
    sudo chmod +x "${INSTALL_DIR}/container-ctl"
  else
    echo "Error: Directory ${INSTALL_DIR} is not writable and sudo is not available." >&2
    echo "Try running with BIN_DIR=\$HOME/.local/bin or as root/sudo." >&2
    exit 1
  fi
fi

echo ""
echo "============================================================"
echo " container-ctl ${TAG} installed successfully!"
echo " Location: ${INSTALL_DIR}/container-ctl"
echo "============================================================"
echo ""
echo "Quick start:"
echo "  container-ctl"
echo ""
