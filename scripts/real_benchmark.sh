#!/bin/bash
# Real benchmark script - tests against actual llama.cpp backend
set -e

GATEWAY_URL="http://localhost:8000"
RESULTS_FILE="real_benchmark_results.txt"

echo "============================================"
echo "Real Benchmark - Inference Gateway"
echo "Backend: llama.cpp with TinyLlama 1.1B Q4"
echo "Date: $(date)"
echo "============================================"
echo ""

# Clear previous results
> $RESULTS_FILE

echo "TEST 1: Concurrency Sweep (Finding the Knee)" | tee -a $RESULTS_FILE
echo "============================================" | tee -a $RESULTS_FILE
echo ""

for concurrency in 1 2 4 8 16; do
    echo "Testing concurrency: $concurrency" | tee -a $RESULTS_FILE
    
    # Run for 30 seconds with warmup
    hey -z 30s -c $concurrency -m POST \
        -H "Content-Type: application/json" \
        -d '{"model":"tinyllama","prompt":"Write a haiku about coding","max_tokens":50,"temperature":0.7}' \
        $GATEWAY_URL/v1/completions 2>&1 | tee -a $RESULTS_FILE
    
    echo "" | tee -a $RESULTS_FILE
    echo "---" | tee -a $RESULTS_FILE
    echo "" | tee -a $RESULTS_FILE
    
    # Cool down between tests
    sleep 5
done

echo ""
echo "TEST 2: Token Cost Variation" | tee -a $RESULTS_FILE
echo "============================================" | tee -a $RESULTS_FILE
echo ""

echo "Small requests (50 tokens):" | tee -a $RESULTS_FILE
hey -z 20s -c 4 -m POST \
    -H "Content-Type: application/json" \
    -d '{"model":"tinyllama","prompt":"Hello","max_tokens":50,"temperature":0.7}' \
    $GATEWAY_URL/v1/completions 2>&1 | tee -a $RESULTS_FILE

echo "" | tee -a $RESULTS_FILE

echo "Large requests (200 tokens):" | tee -a $RESULTS_FILE
hey -z 20s -c 4 -m POST \
    -H "Content-Type: application/json" \
    -d '{"model":"tinyllama","prompt":"Write a detailed explanation of neural networks","max_tokens":200,"temperature":0.7}' \
    $GATEWAY_URL/v1/completions 2>&1 | tee -a $RESULTS_FILE

echo "" | tee -a $RESULTS_FILE

echo ""
echo "TEST 3: Streaming vs Non-Streaming" | tee -a $RESULTS_FILE
echo "============================================" | tee -a $RESULTS_FILE
echo ""

echo "Non-streaming:" | tee -a $RESULTS_FILE
hey -z 15s -c 2 -m POST \
    -H "Content-Type: application/json" \
    -d '{"model":"tinyllama","prompt":"Explain recursion","max_tokens":100,"temperature":0.7,"stream":false}' \
    $GATEWAY_URL/v1/completions 2>&1 | tee -a $RESULTS_FILE

echo "" | tee -a $RESULTS_FILE

echo "Streaming (stream=true):" | tee -a $RESULTS_FILE
hey -z 15s -c 2 -m POST \
    -H "Content-Type: application/json" \
    -d '{"model":"tinyllama","prompt":"Explain recursion","max_tokens":100,"temperature":0.7,"stream":true}' \
    $GATEWAY_URL/v1/completions 2>&1 | tee -a $RESULTS_FILE

echo "" | tee -a $RESULTS_FILE

echo ""
echo "TEST 4: Cache Hit Rate (Repeated Requests)" | tee -a $RESULTS_FILE
echo "============================================" | tee -a $RESULTS_FILE
echo ""

# Send same request multiple times to measure cache
echo "Sending 20 identical requests..." | tee -a $RESULTS_FILE
hey -n 20 -c 1 -m POST \
    -H "Content-Type: application/json" \
    -d '{"model":"tinyllama","prompt":"What is 2+2?","max_tokens":10,"temperature":0.0}' \
    $GATEWAY_URL/v1/completions 2>&1 | tee -a $RESULTS_FILE

echo "" | tee -a $RESULTS_FILE

echo ""
echo "============================================"
echo "Benchmark Complete!"
echo "Results saved to: $RESULTS_FILE"
echo ""
echo "Check metrics:"
echo "curl http://localhost:8000/metrics | grep gateway_"
echo "============================================"
