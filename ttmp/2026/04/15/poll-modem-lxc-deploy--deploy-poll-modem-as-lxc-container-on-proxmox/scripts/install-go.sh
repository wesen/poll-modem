#!/bin/bash
# Install Go in the LXC container
# Run this inside the container (pct exec <id> -- /bin/bash)

set -euo pipefail

GO_VERSION="1.23.4"
ARCH="amd64"

echo "Installing Go ${GO_VERSION}..."

# Update system
apt-get update
apt-get install -y wget git gcc libc6-dev

# Download and install Go
cd /tmp
wget -q "https://go.dev/dl/go${GO_VERSION}.linux-${ARCH}.tar.gz"
tar -C /usr/local -xzf "go${GO_VERSION}.linux-${ARCH}.tar.gz"
rm "go${GO_VERSION}.linux-${ARCH}.tar.gz"

# Setup environment
echo 'export PATH=$PATH:/usr/local/go/bin' >> /etc/profile
echo 'export GOPATH=/root/go' >> /etc/profile
echo 'export PATH=$PATH:$GOPATH/bin' >> /etc/profile

# Source for current session
export PATH=$PATH:/usr/local/go/bin
export GOPATH=/root/go
export PATH=$PATH:$GOPATH/bin

# Verify installation
go version

echo "Go ${GO_VERSION} installed successfully"
