#!/bin/bash
set -e

echo "==================================================================="
echo "Inference Gateway - llama.cpp Setup Script"
echo "==================================================================="
echo ""

# Check for required tools
command -v cmake >/dev/null 2>&1 || { echo "Error: cmake is required but not installed. Install with: brew install cmake"; exit 1; }
command -v git >/dev/null 2>&1 || { echo "Error: git is required but not installed."; exit 1; }

# Create workspace directory
WORKSPACE_DIR="${HOME}/llama-workspace"
mkdir -p "$WORKSPACE_DIR"
cd "$WORKSPACE_DIR"

echo "Working directory: $WORKSPACE_DIR"
echo ""

# Clone llama.cpp if not already present
if [ ! -d "llama.cpp" ]; then
    echo "Cloning llama.cpp..."
    git clone https://github.com/ggerganov/llama.cpp
else
    echo "llama.cpp already cloned, updating..."
    cd llama.cpp
    git pull
    cd ..
fi

# Build llama.cpp
echo ""
echo "Building llama.cpp..."
cd llama.cpp
cmake -B build && cmake --build build --config Release -j
echo "Build complete!"

# Create models directory
mkdir -p models
cd models

# Check if a model is already present
if ls *.gguf 1> /dev/null 2>&1; then
    echo ""
    echo "GGUF model(s) found:"
    ls -lh *.gguf
    echo ""
    echo "Setup complete!"
else
    echo ""
    echo "==================================================================="
    echo "No models found. Download a GGUF model to get started."
    echo ""
    echo "Recommended small models for testing:"
    echo "  - TinyLlama-1.1B-Chat-v1.0 (Q4_K_M, ~700MB)"
    echo "  - Qwen2.5-0.5B-Instruct (Q4_K_M, ~300MB)"
    echo ""
    echo "Download from Hugging Face:"
    echo "  https://huggingface.co/models?search=gguf"
    echo ""
    echo "Place the .gguf file in: $WORKSPACE_DIR/llama.cpp/models/"
    echo "==================================================================="
fi

echo ""
echo "==================================================================="
echo "To start the llama.cpp server:"
echo ""
echo "  cd $WORKSPACE_DIR/llama.cpp"
echo "  ./build/bin/llama-server \\"
echo "    -m models/YOUR_MODEL.gguf \\"
echo "    --port 8080 \\"
echo "    --parallel 8 \\"
echo "    -c 4096"
echo ""
echo "Then start the gateway:"
echo "  cd /Users/vishesh/Documents/Github/Inference-Gateway"
echo "  ./bin/gateway"
echo "==================================================================="
