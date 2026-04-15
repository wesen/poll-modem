#!/bin/bash
# Master deployment script for poll-modem on Proxmox LXC
# Run this on the Proxmox host (root@pve)

set -euo pipefail

CT_ID=${1:-100}
CT_NAME=${2:-poll-modem}
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

echo "=============================================="
echo "Deploying poll-modem to LXC container ${CT_ID}"
echo "=============================================="

# Step 1: Create container
echo ""
echo "[1/5] Creating LXC container..."
bash "${SCRIPT_DIR}/create-lxc-container.sh" "$CT_ID" "$CT_NAME"

# Step 2: Install Go
echo ""
echo "[2/5] Installing Go..."
pct exec "$CT_ID" -- bash -c "$(cat ${SCRIPT_DIR}/install-go.sh)"

# Step 3: Install poll-modem
echo ""
echo "[3/5] Installing poll-modem..."
pct exec "$CT_ID" -- bash -c "$(cat ${SCRIPT_DIR}/install-poll-modem.sh)"

# Step 4: Install tmux
echo ""
echo "[4/5] Installing tmux..."
pct exec "$CT_ID" -- apt-get install -y tmux

# Step 5: Setup complete
echo ""
echo "[5/5] Deployment complete!"
echo ""
echo "Container IP: $(pct exec $CT_ID -- ip addr show eth0 | grep 'inet ' | awk '{print $2}' | cut -d/ -f1)"
echo ""
echo "To use poll-modem:"
echo "  1. Enter container: pct exec $CT_ID -- /bin/bash"
echo "  2. Set credentials: export MODEM_USER=admin MODEM_PASS=yourpassword"
echo "  3. Run: bash /opt/poll-modem/scripts/run-in-tmux.sh"
echo "  4. Attach: tmux attach -t poll-modem"
echo ""
echo "Or run directly:"
echo "  pct exec $CT_ID -- /usr/local/bin/poll-modem --url http://192.168.0.1 --username admin --password pass"
