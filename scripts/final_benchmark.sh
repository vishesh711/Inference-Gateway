#!/bin/bash
# Final real benchmark with varying prompts to avoid KV cache
set -e

GATEWAY_URL="http://localhost:8000"
RESULTS_FILE="final_benchmark_results.md"

echo "# Real Benchmark Results - Varying Prompts" > $RESULTS_FILE
echo "**Date:** $(date)" >> $RESULTS_FILE
echo "**Backend:** llama.cpp with TinyLlama 1.1B Q4" >> $RESULTS_FILE
echo "**Note:** Using varying prompts to bypass llama.cpp KV cache" >> $RESULTS_FILE
echo "" >> $RESULTS_FILE

echo "Running benchmarks with unique prompts..."

# Test 1: Single request latency
echo "## Test 1: Single Request Latency (10 unique prompts)" >> $RESULTS_FILE
echo "" >> $RESULTS_FILE
echo "\`\`\`" >> $RESULTS_FILE
for i in {1..10}; do
    echo "Request $i:"
    /usr/bin/time -p curl -s -X POST http://localhost:8000/v1/completions \
      -H "Content-Type: application/json" \
      -d "{\"model\":\"tinyllama\",\"prompt\":\"Tell me about the number $i in mathematics\",\"max_tokens\":40,\"temperature\":0.7}" \
      -o /dev/null 2>&1 | grep real | awk '{print "  Time: " $2 "s"}'
done >> $RESULTS_FILE 2>&1
echo "\`\`\`" >> $RESULTS_FILE
echo "" >> $RESULTS_FILE

sleep 3

# Test 2: Concurrency=2 with unique prompts
echo "## Test 2: Concurrency=2 (20 unique requests)" >> $RESULTS_FILE
echo "" >> $RESULTS_FILE
START=$(date +%s)
for i in {1..20}; do
    curl -s -X POST http://localhost:8000/v1/completions \
      -H "Content-Type: application/json" \
      -d "{\"model\":\"tinyllama\",\"prompt\":\"Write about topic $i\",\"max_tokens\":40,\"temperature\":0.7}" \
      -o /dev/null &
    
    # Launch 2 at a time
    if [ $((i % 2)) -eq 0 ]; then
        wait
    fi
done
wait
END=$(date +%s)
DURATION=$((END - START))
THROUGHPUT=$(echo "scale=2; 20 / $DURATION" | bc)
echo "Total time: ${DURATION}s" >> $RESULTS_FILE
echo "Throughput: ${THROUGHPUT} req/s" >> $RESULTS_FILE
echo "" >> $RESULTS_FILE

sleep 3

# Test 3: Token size comparison
echo "## Test 3: Token Size Impact" >> $RESULTS_FILE
echo "" >> $RESULTS_FILE

echo "Small (max_tokens=20):" >> $RESULTS_FILE
START=$(date +%s)
for i in {1..5}; do
    curl -s -X POST http://localhost:8000/v1/completions \
      -H "Content-Type: application/json" \
      -d "{\"model\":\"tinyllama\",\"prompt\":\"Number $i\",\"max_tokens\":20,\"temperature\":0.7}" \
      -o /dev/null
done
END=$(date +%s)
echo "  5 requests: $((END - START))s" >> $RESULTS_FILE

sleep 2

echo "Large (max_tokens=100):" >> $RESULTS_FILE
START=$(date +%s)
for i in {1..5}; do
    curl -s -X POST http://localhost:8000/v1/completions \
      -H "Content-Type: application/json" \
      -d "{\"model\":\"tinyllama\",\"prompt\":\"Explain concept $i in detail\",\"max_tokens\":100,\"temperature\":0.7}" \
      -o /dev/null
done
END=$(date +%s)
echo "  5 requests: $((END - START))s" >> $RESULTS_FILE
echo "" >> $RESULTS_FILE

# Test 4: Check metrics
echo "## Metrics" >> $RESULTS_FILE
echo "" >> $RESULTS_FILE
echo "\`\`\`" >> $RESULTS_FILE
curl -s http://localhost:8000/metrics | grep -A2 "gateway_generation_seconds" | head -15 >> $RESULTS_FILE
echo "" >> $RESULTS_FILE
curl -s http://localhost:8000/metrics | grep "gateway_requests_total" | head -5 >> $RESULTS_FILE
echo "\`\`\`" >> $RESULTS_FILE

echo ""
echo "============================================"
echo "Benchmark Complete!"
cat $RESULTS_FILE
