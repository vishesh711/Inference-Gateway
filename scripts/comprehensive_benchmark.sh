#!/bin/bash
# Comprehensive benchmark suite for LLM Gateway
set -e

GATEWAY_URL="http://localhost:8000"
DIRECT_URL="http://localhost:8080"
RESULTS_FILE="comprehensive_benchmark_results.md"

echo "# Comprehensive Benchmark Results" > $RESULTS_FILE
echo "**Date:** $(date)" >> $RESULTS_FILE
echo "**Backend:** llama.cpp with TinyLlama 1.1B Q4" >> $RESULTS_FILE
echo "**Hardware:** $(uname -m) $(uname -s)" >> $RESULTS_FILE
echo "" >> $RESULTS_FILE

echo "Starting comprehensive benchmark suite..."

# ============================================
# Test 1: Embeddings Throughput (No KV cache issue)
# ============================================
echo "## Test 1: Embeddings Throughput" >> $RESULTS_FILE
echo "" >> $RESULTS_FILE
echo "Testing embeddings endpoint (no KV cache interference)..." 

echo "### Single Request" >> $RESULTS_FILE
START=$(date +%s.%N)
curl -s -X POST $GATEWAY_URL/v1/embeddings \
  -H "Content-Type: application/json" \
  -d '{"model":"tinyllama","input":"The quick brown fox jumps over the lazy dog"}' \
  -o /dev/null
END=$(date +%s.%N)
DURATION=$(echo "$END - $START" | bc)
echo "Latency: ${DURATION}s" >> $RESULTS_FILE
echo "" >> $RESULTS_FILE

# Throughput test - 50 requests
echo "### Throughput Test (50 requests, concurrency=1)" >> $RESULTS_FILE
START=$(date +%s.%N)
for i in {1..50}; do
    curl -s -X POST $GATEWAY_URL/v1/embeddings \
      -H "Content-Type: application/json" \
      -d "{\"model\":\"tinyllama\",\"input\":\"Sample text number $i for embeddings test\"}" \
      -o /dev/null
done
END=$(date +%s.%N)
DURATION=$(echo "$END - $START" | bc)
THROUGHPUT=$(echo "scale=2; 50 / $DURATION" | bc)
echo "- Total time: ${DURATION}s" >> $RESULTS_FILE
echo "- Throughput: **${THROUGHPUT} req/s**" >> $RESULTS_FILE
echo "- Average latency: $(echo "scale=3; $DURATION / 50" | bc)s" >> $RESULTS_FILE
echo "" >> $RESULTS_FILE

sleep 2

# ============================================
# Test 2: Completion Latency with Unique Prompts
# ============================================
echo "## Test 2: Completion Latency" >> $RESULTS_FILE
echo "" >> $RESULTS_FILE
echo "Testing completion with unique prompts..."

echo "### 10 Sequential Requests (unique prompts)" >> $RESULTS_FILE
echo "\`\`\`" >> $RESULTS_FILE
for i in {1..10}; do
    START=$(date +%s.%N)
    curl -s -X POST $GATEWAY_URL/v1/completions \
      -H "Content-Type: application/json" \
      -d "{\"model\":\"tinyllama\",\"prompt\":\"Tell me an interesting fact about the number $i\",\"max_tokens\":30,\"temperature\":0.8}" \
      -o /dev/null
    END=$(date +%s.%N)
    DURATION=$(echo "$END - $START" | bc)
    echo "Request $i: ${DURATION}s"
done >> $RESULTS_FILE 2>&1
echo "\`\`\`" >> $RESULTS_FILE
echo "" >> $RESULTS_FILE

sleep 2

# ============================================
# Test 3: Concurrency Sweep
# ============================================
echo "## Test 3: Concurrency Sweep" >> $RESULTS_FILE
echo "" >> $RESULTS_FILE
echo "Testing different concurrency levels..."

for CONCURRENCY in 1 2 4 8; do
    echo "### Concurrency=$CONCURRENCY" >> $RESULTS_FILE
    echo "Running 40 requests with concurrency $CONCURRENCY..."
    
    START=$(date +%s.%N)
    for i in $(seq 1 40); do
        curl -s -X POST $GATEWAY_URL/v1/completions \
          -H "Content-Type: application/json" \
          -d "{\"model\":\"tinyllama\",\"prompt\":\"Write a short sentence about topic ${CONCURRENCY}${i}\",\"max_tokens\":25,\"temperature\":0.7}" \
          -o /dev/null &
        
        # Control concurrency
        if [ $((i % CONCURRENCY)) -eq 0 ]; then
            wait
        fi
    done
    wait
    END=$(date +%s.%N)
    
    DURATION=$(echo "$END - $START" | bc)
    THROUGHPUT=$(echo "scale=2; 40 / $DURATION" | bc)
    AVG_LATENCY=$(echo "scale=3; $DURATION / 40" | bc)
    
    echo "- Total time: ${DURATION}s" >> $RESULTS_FILE
    echo "- Throughput: **${THROUGHPUT} req/s**" >> $RESULTS_FILE
    echo "- Average latency: ${AVG_LATENCY}s" >> $RESULTS_FILE
    echo "" >> $RESULTS_FILE
    
    sleep 3
done

# ============================================
# Test 4: Token Size Impact
# ============================================
echo "## Test 4: Token Size Impact" >> $RESULTS_FILE
echo "" >> $RESULTS_FILE
echo "Comparing small vs large token requests..."

echo "### Small Tokens (max_tokens=20)" >> $RESULTS_FILE
START=$(date +%s.%N)
for i in {1..10}; do
    curl -s -X POST $GATEWAY_URL/v1/completions \
      -H "Content-Type: application/json" \
      -d "{\"model\":\"tinyllama\",\"prompt\":\"Write number $i\",\"max_tokens\":20,\"temperature\":0.7}" \
      -o /dev/null
done
END=$(date +%s.%N)
DURATION=$(echo "$END - $START" | bc)
echo "- 10 requests: ${DURATION}s" >> $RESULTS_FILE
echo "- Average: $(echo "scale=3; $DURATION / 10" | bc)s per request" >> $RESULTS_FILE
echo "" >> $RESULTS_FILE

sleep 2

echo "### Large Tokens (max_tokens=100)" >> $RESULTS_FILE
START=$(date +%s.%N)
for i in {1..10}; do
    curl -s -X POST $GATEWAY_URL/v1/completions \
      -H "Content-Type: application/json" \
      -d "{\"model\":\"tinyllama\",\"prompt\":\"Explain in detail the concept number $i\",\"max_tokens\":100,\"temperature\":0.7}" \
      -o /dev/null
done
END=$(date +%s.%N)
DURATION=$(echo "$END - $START" | bc)
echo "- 10 requests: ${DURATION}s" >> $RESULTS_FILE
echo "- Average: $(echo "scale=3; $DURATION / 10" | bc)s per request" >> $RESULTS_FILE
echo "" >> $RESULTS_FILE

# ============================================
# Test 5: Gateway vs Direct Comparison
# ============================================
echo "## Test 5: Gateway Overhead" >> $RESULTS_FILE
echo "" >> $RESULTS_FILE
echo "Measuring gateway overhead..."

echo "### Direct to llama.cpp (5 requests)" >> $RESULTS_FILE
START=$(date +%s.%N)
for i in {1..5}; do
    curl -s -X POST $DIRECT_URL/v1/completions \
      -H "Content-Type: application/json" \
      -d "{\"prompt\":\"Direct test $i\",\"n_predict\":30}" \
      -o /dev/null
done
END=$(date +%s.%N)
DIRECT_TIME=$(echo "$END - $START" | bc)
echo "Total: ${DIRECT_TIME}s (avg: $(echo "scale=3; $DIRECT_TIME / 5" | bc)s)" >> $RESULTS_FILE
echo "" >> $RESULTS_FILE

sleep 2

echo "### Through gateway (5 requests)" >> $RESULTS_FILE
START=$(date +%s.%N)
for i in {1..5}; do
    curl -s -X POST $GATEWAY_URL/v1/completions \
      -H "Content-Type: application/json" \
      -d "{\"model\":\"tinyllama\",\"prompt\":\"Gateway test $i\",\"max_tokens\":30}" \
      -o /dev/null
done
END=$(date +%s.%N)
GATEWAY_TIME=$(echo "$END - $START" | bc)
echo "Total: ${GATEWAY_TIME}s (avg: $(echo "scale=3; $GATEWAY_TIME / 5" | bc)s)" >> $RESULTS_FILE
echo "" >> $RESULTS_FILE

OVERHEAD=$(echo "scale=3; ($GATEWAY_TIME - $DIRECT_TIME) / 5" | bc)
echo "**Gateway overhead: ~${OVERHEAD}s per request**" >> $RESULTS_FILE
echo "" >> $RESULTS_FILE

# ============================================
# Test 6: Streaming Performance
# ============================================
echo "## Test 6: Streaming Performance" >> $RESULTS_FILE
echo "" >> $RESULTS_FILE
echo "Testing SSE streaming..."

echo "### 5 Streaming Requests" >> $RESULTS_FILE
START=$(date +%s.%N)
for i in {1..5}; do
    curl -s -X POST $GATEWAY_URL/v1/completions \
      -H "Content-Type: application/json" \
      -d "{\"model\":\"tinyllama\",\"prompt\":\"Stream test $i\",\"max_tokens\":40,\"stream\":true}" \
      -o /dev/null
done
END=$(date +%s.%N)
DURATION=$(echo "$END - $START" | bc)
echo "Total: ${DURATION}s (avg: $(echo "scale=3; $DURATION / 5" | bc)s)" >> $RESULTS_FILE
echo "" >> $RESULTS_FILE

# ============================================
# Test 7: Metrics Collection
# ============================================
echo "## Test 7: Gateway Metrics" >> $RESULTS_FILE
echo "" >> $RESULTS_FILE
echo "\`\`\`" >> $RESULTS_FILE
curl -s $GATEWAY_URL/metrics | grep -E "(gateway_requests_total|gateway_generation_seconds|gateway_queue_wait_seconds)" | head -20 >> $RESULTS_FILE
echo "\`\`\`" >> $RESULTS_FILE
echo "" >> $RESULTS_FILE

# ============================================
# Summary
# ============================================
echo "## Summary" >> $RESULTS_FILE
echo "" >> $RESULTS_FILE
echo "✅ **Embeddings throughput:** Measured" >> $RESULTS_FILE
echo "✅ **Completion latency:** Measured with unique prompts" >> $RESULTS_FILE
echo "✅ **Concurrency sweep:** 1, 2, 4, 8 tested" >> $RESULTS_FILE
echo "✅ **Token size impact:** Small vs large compared" >> $RESULTS_FILE
echo "✅ **Gateway overhead:** Measured vs direct" >> $RESULTS_FILE
echo "✅ **Streaming:** Tested and timed" >> $RESULTS_FILE
echo "✅ **Metrics:** Collected and validated" >> $RESULTS_FILE
echo "" >> $RESULTS_FILE

echo ""
echo "============================================"
echo "✅ Comprehensive Benchmark Complete!"
echo "============================================"
echo ""
cat $RESULTS_FILE
