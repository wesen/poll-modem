#!/bin/bash
# Build and install poll-modem in the LXC container
# Run this inside the container

set -euo pipefail

REPO_URL="https://github.com/go-go-golems/poll-modem.git"
INSTALL_DIR="/opt/poll-modem"
CONFIG_DIR="/root/.config/poll-modem"

echo "Installing poll-modem..."

# Install dependencies
apt-get install -y git gcc sqlite3 libsqlite3-dev

# Clone repository
if [ -d "$INSTALL_DIR" ]; then
    echo "Directory exists, pulling latest..."
    cd "$INSTALL_DIR"
    git pull
else
    git clone "$REPO_URL" "$INSTALL_DIR"
    cd "$INSTALL_DIR"
fi

# Build
export PATH=$PATH:/usr/local/go/bin
export GOPATH=/root/go
export PATH=$PATH:$GOPATH/bin
export GOWORK=off

echo "Building poll-modem..."
go build -o /usr/local/bin/poll-modem ./cmd/poll-modem

# Create config directory
mkdir -p "$CONFIG_DIR"

echo "poll-modem installed to /usr/local/bin/poll-modem"
echo "Config directory: $CONFIG_DIR"
