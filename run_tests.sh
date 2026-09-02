#!/bin/bash
set -e

cd /Users/vishesh/Documents/Github/Inference-Gateway

echo "Running comprehensive benchmark tests..."
echo ""

# Test concurrency levels
LEVELS=(1 2 4 8 16 32)
RESULTS_FILE="benchmark_results.txt"

echo "Benchmark Results - Mock Engine" > $RESULTS_FILE
echo "================================" >> $RESULTS_FILE
echo "" >> $RESULTS_FILE

for LEVEL in "${LEVELS[@]}"; do
    echo "Testing concurrency level: $LEVEL"
    
    # Update config
    sed -i.bak "s/max_in_flight: .*/max_in_flight: $LEVEL/" config.yaml
    
    # Stop any existing gateway
    pkill -9 gateway 2>/dev/null || true
    sleep 2
    
    # Start gateway
    ./bin/gateway > gateway_$LEVEL.log 2>&1 &
    GATEWAY_PID=$!
    sleep 3
    
    # Run load test
    ./bin/loadgen -workers $LEVEL -duration 15s -warmup 3s -model mock \
        > loadgen_$LEVEL.log 2>&1
    
    # Extract results
    echo "Concurrency: $LEVEL" >> $RESULTS_FILE
    grep -A 15 "LOAD TEST RESULTS" loadgen_$LEVEL.log | head -20 >> $RESULTS_FILE
    echo "" >> $RESULTS_FILE
    
    # Stop gateway
    kill $GATEWAY_PID 2>/dev/null || true
    sleep 1
    
    echo "  Done."
done

# Restore original config
mv config.yaml.bak config.yaml

echo ""
echo "Results saved to: $RESULTS_FILE"
cat $RESULTS_FILE
