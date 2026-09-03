#!/bin/bash
# Focused real benchmark - meaningful tests only
set -e

GATEWAY_URL="http://localhost:8000"
RESULTS_FILE="focused_benchmark_results.md"

echo "# Real Benchmark Results" > $RESULTS_FILE
echo "**Date:** $(date)" >> $RESULTS_FILE
echo "**Backend:** llama.cpp with TinyLlama 1.1B Q4" >> $RESULTS_FILE
echo "**Hardware:** $(sysctl -n machdep.cpu.brand_string)" >> $RESULTS_FILE
echo "" >> $RESULTS_FILE

echo "Running focused benchmarks (cache disabled)..."
echo ""

echo "## Test 1: Concurrency Sweep (Finding the Knee)" >> $RESULTS_FILE
echo "" >> $RESULTS_FILE
echo "Testing different concurrency levels to find optimal throughput:" >> $RESULTS_FILE
echo "" >> $RESULTS_FILE

for concurrency in 1 2 4 8 16; do
    echo "Testing concurrency: $concurrency"
    echo "### Concurrency: $concurrency" >> $RESULTS_FILE
    echo "" >> $RESULTS_FILE
    
    # 20 second test
    hey -z 20s -c $concurrency -m POST \
        -H "Content-Type: application/json" \
        -d '{"model":"tinyllama","prompt":"Write a haiku about programming","max_tokens":50,"temperature":0.7}' \
        $GATEWAY_URL/v1/completions 2>&1 | grep -E "Requests/sec|Average|p50|p95|Status" | tee -a $RESULTS_FILE
    
    echo "" >> $RESULTS_FILE
    sleep 3
done

echo ""
echo "## Test 2: Token Cost Comparison" >> $RESULTS_FILE
echo "" >> $RESULTS_FILE
echo "Comparing small vs large token requests (concurrency=2):" >> $RESULTS_FILE
echo "" >> $RESULTS_FILE

echo "Testing small requests (max_tokens=25)..."
echo "### Small Requests (25 tokens)" >> $RESULTS_FILE
hey -n 20 -c 2 -m POST \
    -H "Content-Type: application/json" \
    -d '{"model":"tinyllama","prompt":"Hello","max_tokens":25,"temperature":0.7}' \
    $GATEWAY_URL/v1/completions 2>&1 | grep -E "Requests/sec|Average|Fastest|Slowest" | tee -a $RESULTS_FILE

echo "" >> $RESULTS_FILE
sleep 2

echo "Testing large requests (max_tokens=150)..."
echo "### Large Requests (150 tokens)" >> $RESULTS_FILE
hey -n 20 -c 2 -m POST \
    -H "Content-Type: application/json" \
    -d '{"model":"tinyllama","prompt":"Explain neural networks in detail","max_tokens":150,"temperature":0.7}' \
    $GATEWAY_URL/v1/completions 2>&1 | grep -E "Requests/sec|Average|Fastest|Slowest" | tee -a $RESULTS_FILE

echo "" >> $RESULTS_FILE

echo ""
echo "## Test 3: Overload Behavior" >> $RESULTS_FILE
echo "" >> $RESULTS_FILE
echo "Testing system under heavy load (concurrency=32, beyond optimal):" >> $RESULTS_FILE
echo "" >> $RESULTS_FILE

echo "Testing overload (concurrency=32)..."
hey -z 15s -c 32 -m POST \
    -H "Content-Type: application/json" \
    -d '{"model":"tinyllama","prompt":"Test","max_tokens":30,"temperature":0.7}' \
    $GATEWAY_URL/v1/completions 2>&1 | grep -E "Requests/sec|Average|Status" | tee -a $RESULTS_FILE

echo "" >> $RESULTS_FILE

echo ""
echo "## Metrics Sample" >> $RESULTS_FILE
echo "" >> $RESULTS_FILE
echo "\`\`\`" >> $RESULTS_FILE
curl -s http://localhost:8000/metrics | grep -E "gateway_requests_total|gateway_queue|gateway_in_flight|gateway_generation_seconds|gateway_token_budget" | head -20 >> $RESULTS_FILE
echo "\`\`\`" >> $RESULTS_FILE

echo ""
echo "============================================"
echo "Benchmark Complete!"
echo "Results saved to: $RESULTS_FILE"
echo "============================================"
cat $RESULTS_FILE
