#!/bin/bash
# Create LXC container for poll-modem on Proxmox
# Run this on the Proxmox host (root@pve)

set -euo pipefail

CT_ID=${1:-100}
CT_NAME=${2:-poll-modem}
CT_HOSTNAME="${CT_NAME}"
CT_MEMORY=512  # MB
CT_CORES=1
CT_DISK=4      # GB
CT_NET="ip=dhcp,bridge=vmbr0"
CT_TEMPLATE="local:vztmpl/debian-12-standard_12.12-1_amd64.tar.zst"

echo "Creating LXC container ${CT_ID} (${CT_NAME})..."

# Check if template exists
if ! pveam list local | grep -q "debian-12-standard"; then
    echo "Template not found. Downloading..."
    pveam download local debian-12-standard_12.12-1_amd64.tar.zst
fi

# Create container
pct create ${CT_ID} ${CT_TEMPLATE} \
    --hostname ${CT_HOSTNAME} \
    --memory ${CT_MEMORY} \
    --cores ${CT_CORES} \
    --rootfs local-lvm:${CT_DISK} \
    --net0 ${CT_NET} \
    --unprivileged 1 \
    --features nesting=1 \
    --ostype debian \
    --start 0

echo "Container ${CT_ID} created successfully"
echo "Starting container..."
pct start ${CT_ID}

# Wait for container to be ready
sleep 5

echo "Container ${CT_ID} started"
echo "IP Address: $(pct exec ${CT_ID} -- ip addr show eth0 | grep 'inet ' | awk '{print $2}' | cut -d/ -f1)"
