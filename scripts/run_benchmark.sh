#!/bin/bash
set -e

echo "==================================================================="
echo "Inference Gateway - Concurrency Benchmark Sweep"
echo "==================================================================="
echo ""

# Check if gateway and loadgen are built
if [ ! -f "bin/gateway" ] || [ ! -f "bin/loadgen" ]; then
    echo "Error: Binaries not found. Run 'make build' first."
    exit 1
fi

# Check if llama.cpp server is running
if ! curl -s http://localhost:8080/health > /dev/null 2>&1; then
    echo "Warning: llama.cpp server doesn't appear to be running on port 8080"
    echo "Start it with:"
    echo "  ./build/bin/llama-server -m models/YOUR_MODEL.gguf --port 8080 --parallel 8"
    echo ""
    read -p "Continue anyway? (y/N) " -n 1 -r
    echo
    if [[ ! $REPLY =~ ^[Yy]$ ]]; then
        exit 1
    fi
fi

# Create results directory
mkdir -p results
TIMESTAMP=$(date +%Y%m%d_%H%M%S)
RESULTS_DIR="results/benchmark_$TIMESTAMP"
mkdir -p "$RESULTS_DIR"

echo "Results will be saved to: $RESULTS_DIR"
echo ""

# Concurrency levels to test
LEVELS=(1 2 4 8 16 32)
DURATION="${1:-180s}"  # Default 3 minutes, or first argument
WARMUP="${2:-30s}"     # Default 30 seconds, or second argument

echo "Test duration: $DURATION per level (warmup: $WARMUP)"
echo ""

# Run benchmark for each concurrency level
for CONCURRENCY in "${LEVELS[@]}"; do
    echo "==================================================================="
    echo "Testing concurrency level: $CONCURRENCY"
    echo "==================================================================="
    
    # Update config
    sed -i.bak "s/max_in_flight: .*/max_in_flight: $CONCURRENCY/" config.yaml
    
    # Stop any running gateway
    pkill -9 gateway 2>/dev/null || true
    sleep 2
    
    # Start gateway in background
    echo "Starting gateway with max_in_flight=$CONCURRENCY..."
    ./bin/gateway -config config.yaml > "$RESULTS_DIR/gateway_$CONCURRENCY.log" 2>&1 &
    GATEWAY_PID=$!
    
    # Wait for gateway to be ready
    sleep 5
    
    # Run load test
    echo "Running load test..."
    ./bin/loadgen \
        -workers $CONCURRENCY \
        -duration "$DURATION" \
        -warmup "$WARMUP" \
        -model tinyllama \
        -prompt "Write a short story about a robot learning to code." \
        -max-tokens 100 \
        | tee "$RESULTS_DIR/loadgen_$CONCURRENCY.txt"
    
    # Stop gateway
    kill $GATEWAY_PID 2>/dev/null || true
    sleep 2
    
    echo ""
done

# Restore original config
mv config.yaml.bak config.yaml

echo "==================================================================="
echo "Benchmark complete!"
echo ""
echo "Results saved to: $RESULTS_DIR"
echo ""
echo "To generate summary table, run:"
echo "  ./scripts/summarize_results.sh $RESULTS_DIR"
echo "==================================================================="
