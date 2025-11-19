#!/bin/bash

# Script to run all services for the gRPC Replication Demo
# This script uses tmux to run all services in separate panes

set -e

DEMO_DIR="cmd/examples/grpc-replication-demo"

echo "🚀 Starting UnisonDB gRPC Replication Demo"
echo "=========================================="
echo ""

# Check if tmux is installed
if ! command -v tmux &> /dev/null; then
    echo "❌ tmux is not installed. Please install it first:"
    echo "   Ubuntu/Debian: sudo apt-get install tmux"
    echo "   macOS: brew install tmux"
    exit 1
fi

# Check if unisondb binary exists
if [ ! -f "./unisondb" ]; then
    echo "❌ unisondb binary not found. Building..."
    go build -o unisondb ./cmd/unisondb
    echo "✅ Built unisondb"
fi

# Clean up old data directories
echo "🧹 Cleaning up old data directories..."
rm -rf /tmp/unisondb/grpc-demo
rm -rf /tmp/unisondb-client1
rm -rf /tmp/unisondb-client2
echo "✅ Cleaned up"
echo ""

# Create a new tmux session
SESSION_NAME="unisondb-demo"

# Kill existing session if it exists
tmux kill-session -t $SESSION_NAME 2>/dev/null || true

echo "📺 Creating tmux session: $SESSION_NAME"
echo ""

# Create new session and windows
tmux new-session -d -s $SESSION_NAME -n "primary"

# Window 1: UnisonDB Primary
tmux send-keys -t $SESSION_NAME:0 "./unisondb --config ${DEMO_DIR}/configs/primary.toml replicator" C-m

# Wait for primary to start
sleep 3

# Window 2: gRPC Client 1
tmux new-window -t $SESSION_NAME -n "client1"
tmux send-keys -t $SESSION_NAME:1 "cd ${DEMO_DIR}/grpc-client1 && go run main.go" C-m

# Wait for client1 to connect
sleep 2

# Window 3: gRPC Client 2
tmux new-window -t $SESSION_NAME -n "client2"
tmux send-keys -t $SESSION_NAME:2 "cd ${DEMO_DIR}/grpc-client2 && go run main.go" C-m

# Wait for client2 to connect
sleep 2

# Window 4: Writer API
tmux new-window -t $SESSION_NAME -n "writer-api"
tmux send-keys -t $SESSION_NAME:3 "cd ${DEMO_DIR}/writer-api && go run main.go" C-m

# Wait for writer API to start
sleep 2

# Window 5: Test terminal (for running commands)
tmux new-window -t $SESSION_NAME -n "test"
tmux send-keys -t $SESSION_NAME:4 "cd ${DEMO_DIR}" C-m
tmux send-keys -t $SESSION_NAME:4 "echo ''" C-m
tmux send-keys -t $SESSION_NAME:4 "echo '✅ All services started!'" C-m
tmux send-keys -t $SESSION_NAME:4 "echo ''" C-m
tmux send-keys -t $SESSION_NAME:4 "echo 'Windows:'" C-m
tmux send-keys -t $SESSION_NAME:4 "echo '  0: UnisonDB Primary (HTTP :8001, gRPC :4001)'" C-m
tmux send-keys -t $SESSION_NAME:4 "echo '  1: gRPC Client 1'" C-m
tmux send-keys -t $SESSION_NAME:4 "echo '  2: gRPC Client 2'" C-m
tmux send-keys -t $SESSION_NAME:4 "echo '  3: Writer API (:8080)'" C-m
tmux send-keys -t $SESSION_NAME:4 "echo '  4: Test Terminal (you are here)'" C-m
tmux send-keys -t $SESSION_NAME:4 "echo ''" C-m
tmux send-keys -t $SESSION_NAME:4 "echo 'Switch windows: Ctrl+b then 0-4'" C-m
tmux send-keys -t $SESSION_NAME:4 "echo 'Detach: Ctrl+b then d'" C-m
tmux send-keys -t $SESSION_NAME:4 "echo 'Kill session: tmux kill-session -t ${SESSION_NAME}'" C-m
tmux send-keys -t $SESSION_NAME:4 "echo ''" C-m
tmux send-keys -t $SESSION_NAME:4 "echo '📝 To write test data, run:'" C-m
tmux send-keys -t $SESSION_NAME:4 "echo '   ./test-write.sh'" C-m
tmux send-keys -t $SESSION_NAME:4 "echo ''" C-m

echo "✅ All services started in tmux session: $SESSION_NAME"
echo ""
echo "To attach to the session:"
echo "  tmux attach -t $SESSION_NAME"
echo ""
echo "To kill the session:"
echo "  tmux kill-session -t $SESSION_NAME"
echo ""
echo "Windows:"
echo "  0: UnisonDB Primary"
echo "  1: gRPC Client 1"
echo "  2: gRPC Client 2"
echo "  3: Writer API"
echo "  4: Test Terminal"
echo ""

# Attach to the session
tmux attach -t $SESSION_NAME
