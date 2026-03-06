#!/bin/sh
set -e

REPO="devforward/krawl"
INSTALL_DIR="/usr/local/bin"

OS=$(uname -s | tr '[:upper:]' '[:lower:]')
ARCH=$(uname -m)
case "$ARCH" in
  x86_64) ARCH="amd64" ;;
  aarch64|arm64) ARCH="arm64" ;;
  *) echo "Unsupported architecture: $ARCH" && exit 1 ;;
esac

NAME="krawl-${OS}-${ARCH}"
DOWNLOAD_URL="https://github.com/${REPO}/releases/latest/download/${NAME}"

echo "Downloading ${NAME}..."
curl -fSL -o krawl "$DOWNLOAD_URL"
chmod +x krawl

if [ -w "$INSTALL_DIR" ]; then
  mv krawl "$INSTALL_DIR/krawl"
else
  echo "Installing to ${INSTALL_DIR} (requires sudo)..."
  sudo mv krawl "$INSTALL_DIR/krawl"
fi

echo "Installed krawl to ${INSTALL_DIR}/krawl"
krawl --help
