#!/bin/bash
# Quick test script for the Inference Gateway

set -e

GATEWAY_URL="http://localhost:8000"

echo "=================================================================="
echo "Inference Gateway - Quick Test"
echo "=================================================================="
echo ""

# Check if gateway is running
if ! curl -s "$GATEWAY_URL/health" > /dev/null 2>&1; then
    echo "❌ Gateway is not running on $GATEWAY_URL"
    echo ""
    echo "Start it with:"
    echo "  ./bin/gateway"
    echo ""
    exit 1
fi

echo "✅ Gateway is running"
echo ""

# Test 1: Health check
echo "Test 1: Health Check"
echo "------------------------------------------------------------"
curl -s "$GATEWAY_URL/health"
echo ""
echo ""

# Test 2: Completions endpoint
echo "Test 2: Completion Request"
echo "------------------------------------------------------------"
curl -s -X POST "$GATEWAY_URL/v1/completions" \
  -H "Content-Type: application/json" \
  -d '{
    "model": "test",
    "prompt": "The capital of France is",
    "max_tokens": 20,
    "temperature": 0.7
  }' | python3 -m json.tool
echo ""

# Test 3: Chat completions endpoint
echo "Test 3: Chat Completion Request"
echo "------------------------------------------------------------"
curl -s -X POST "$GATEWAY_URL/v1/chat/completions" \
  -H "Content-Type: application/json" \
  -d '{
    "model": "test",
    "messages": [
      {"role": "user", "content": "Say hello!"}
    ],
    "max_tokens": 50
  }' | python3 -m json.tool
echo ""

# Test 4: Check metrics
echo "Test 4: Metrics"
echo "------------------------------------------------------------"
curl -s "$GATEWAY_URL/metrics" | grep -E "gateway_(requests_total|in_flight|queue_depth)" | head -10
echo ""
echo "Full metrics available at: $GATEWAY_URL/metrics"
echo ""

echo "=================================================================="
echo "✅ All tests passed!"
echo ""
echo "Next steps:"
echo "  - Run load test:  ./bin/loadgen -workers 4 -duration 30s -warmup 5s"
echo "  - View metrics:   curl $GATEWAY_URL/metrics"
echo "  - Run benchmark:  ./scripts/run_benchmark.sh"
echo "=================================================================="
