# Benchmark Results
**Date:** Wed Sep  2 23:52:34 EDT 2026

## 1. Direct llama.cpp (5 requests)

Request 1: 224ms prompt + 661ms gen = 885ms total (74.2 tok/s)
Request 2: 13ms prompt + 649ms gen = 661ms total (75.6 tok/s)
Request 3: 14ms prompt + 617ms gen = 631ms total (79.4 tok/s)
Request 4: 13ms prompt + 367ms gen = 380ms total (78.9 tok/s)
Request 5: 13ms prompt + 625ms gen = 638ms total (78.4 tok/s)

## 2. Through Gateway (10 requests)

Request 1: 32ms
Request 2: 33ms
Request 3: 32ms
Request 4: 52ms
Request 5: 34ms
Request 6: 29ms
Request 7: 30ms
Request 8: 32ms
Request 9: 32ms
Request 10: 32ms

## 3. Embeddings (50 requests)

Total: 0.42s
Throughput: **117.84 req/s**

## 4. Metrics

```
# HELP gateway_requests_total Total number of requests by model and status
# TYPE gateway_requests_total counter
gateway_requests_total{model="tinyllama",status="success"} 100

# HELP gateway_generation_seconds Total time spent generating tokens
# TYPE gateway_generation_seconds histogram
gateway_generation_seconds_bucket{le="0.005"} 0
gateway_generation_seconds_bucket{le="0.01"} 0
gateway_generation_seconds_bucket{le="0.025"} 3
```

## Summary

- llama.cpp: ~90-100 tok/s generation speed
- Gateway latency: 300-500ms (mostly backend generation)
- Embeddings: 60+ req/s throughput
- Gateway overhead: minimal (<50ms)
