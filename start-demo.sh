#!/bin/bash
# Quick start script for local development

set -e

echo "🚀 Starting Inference Gateway Demo..."
echo ""

# Colors
GREEN='\033[0;32m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Check if llama.cpp is running
if ! curl -s http://localhost:8080/health > /dev/null 2>&1; then
    echo -e "${BLUE}⚠️  llama.cpp not detected on port 8080${NC}"
    echo "Please start llama.cpp in another terminal:"
    echo ""
    echo "  cd ~/Documents/Github/llama.cpp"
    echo "  ./build/bin/llama-server -m models/tinyllama-1.1b-chat-v1.0.Q4_K_M.gguf --port 8080"
    echo ""
    echo "Then run this script again."
    exit 1
fi

echo -e "${GREEN}✓ llama.cpp is running on port 8080${NC}"

# Check if gateway binary exists
if [ ! -f "./bin/gateway" ]; then
    echo "Building gateway..."
    go build -o bin/gateway cmd/gateway/main.go
    echo -e "${GREEN}✓ Gateway built${NC}"
fi

# Start gateway in background
echo "Starting gateway on port 8000..."
./bin/gateway > gateway.log 2>&1 &
GATEWAY_PID=$!
echo $GATEWAY_PID > gateway.pid
sleep 2

# Check if gateway started successfully
if ! curl -s http://localhost:8000/health > /dev/null 2>&1; then
    echo "❌ Failed to start gateway"
    cat gateway.log
    exit 1
fi

echo -e "${GREEN}✓ Gateway is running on port 8000${NC}"

# Start frontend
cd frontend

if [ ! -d "node_modules" ]; then
    echo "Installing frontend dependencies..."
    npm install
fi

echo "Starting frontend on port 3000..."
echo ""
echo -e "${GREEN}================================${NC}"
echo -e "${GREEN}✅ Demo is ready!${NC}"
echo -e "${GREEN}================================${NC}"
echo ""
echo "Open your browser: ${BLUE}http://localhost:3000${NC}"
echo ""
echo "To stop the demo:"
echo "  1. Press Ctrl+C in this terminal"
echo "  2. Run: kill \$(cat ../gateway.pid)"
echo ""

npm run dev
