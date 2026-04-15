#!/bin/bash
# Run poll-modem in tmux session for persistent TUI
# Run this inside the container

set -euo pipefail

SESSION_NAME="poll-modem"
MODEM_URL="${MODEM_URL:-http://192.168.0.1}"
MODEM_USER="${MODEM_USER:-}"
MODEM_PASS="${MODEM_PASS:-}"
INTERVAL="${INTERVAL:-30s}"

# Build command
CMD="/usr/local/bin/poll-modem --url ${MODEM_URL} --interval ${INTERVAL}"

if [ -n "$MODEM_USER" ]; then
    CMD="${CMD} --username ${MODEM_USER}"
fi

if [ -n "$MODEM_PASS" ]; then
    CMD="${CMD} --password ${MODEM_PASS}"
fi

# Kill existing session if exists
tmux kill-session -t "$SESSION_NAME" 2>/dev/null || true

# Create new session
tmux new-session -d -s "$SESSION_NAME "$CMD"

echo "poll-modem running in tmux session: $SESSION_NAME"
echo "Attach with: tmux attach -t $SESSION_NAME"
echo "Detach with: Ctrl+B, then D"
