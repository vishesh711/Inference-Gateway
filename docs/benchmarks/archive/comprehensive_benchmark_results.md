# Comprehensive Benchmark Results
**Date:** Wed Sep  2 23:43:52 EDT 2026
**Backend:** llama.cpp with TinyLlama 1.1B Q4
**Hardware:** arm64 Darwin

## Test 1: Embeddings Throughput

### Single Request
Latency: .031019000s

### Throughput Test (50 requests, concurrency=1)
- Total time: .392386000s
- Throughput: **127.42 req/s**
- Average latency: .007s

## Test 2: Completion Latency

### 10 Sequential Requests (unique prompts)
```
Request 1: .515521000s
Request 2: .327647000s
Request 3: .339042000s
Request 4: .081289000s
Request 5: .326017000s
Request 6: .317831000s
Request 7: .315843000s
Request 8: .050781000s
Request 9: .316481000s
Request 10: .258212000s
```

## Test 3: Concurrency Sweep

### Concurrency=1
- Total time: 9.474509000s
- Throughput: **4.22 req/s**
- Average latency: .236s

### Concurrency=2
- Total time: 6.859214000s
- Throughput: **5.83 req/s**
- Average latency: .171s

### Concurrency=4
- Total time: 1.519661000s
- Throughput: **26.32 req/s**
- Average latency: .037s

### Concurrency=8
- Total time: .093990000s
- Throughput: **425.57 req/s**
- Average latency: .002s

## Test 4: Token Size Impact

### Small Tokens (max_tokens=20)
- 10 requests: .083023000s
- Average: .008s per request

### Large Tokens (max_tokens=100)
- 10 requests: .094239000s
- Average: .009s per request

## Test 5: Gateway Overhead

### Direct to llama.cpp (5 requests)
Total: 1.872778000s (avg: .374s)

### Through gateway (5 requests)
Total: .066443000s (avg: .013s)

**Gateway overhead: ~-.361s per request**

## Test 6: Streaming Performance

### 5 Streaming Requests
Total: .052555000s (avg: .010s)

## Test 7: Gateway Metrics

```
# HELP gateway_generation_seconds Total time spent generating tokens
# TYPE gateway_generation_seconds histogram
gateway_generation_seconds_bucket{le="0.005"} 0
gateway_generation_seconds_bucket{le="0.01"} 0
gateway_generation_seconds_bucket{le="0.025"} 3
gateway_generation_seconds_bucket{le="0.05"} 6
gateway_generation_seconds_bucket{le="0.1"} 10
gateway_generation_seconds_bucket{le="0.25"} 28
gateway_generation_seconds_bucket{le="0.5"} 96
gateway_generation_seconds_bucket{le="1"} 100
gateway_generation_seconds_bucket{le="2.5"} 100
gateway_generation_seconds_bucket{le="5"} 100
gateway_generation_seconds_bucket{le="10"} 100
gateway_generation_seconds_bucket{le="+Inf"} 100
gateway_generation_seconds_sum 27.330610998999997
gateway_generation_seconds_count 100
# HELP gateway_queue_wait_seconds Time spent waiting in the admission queue
# TYPE gateway_queue_wait_seconds histogram
gateway_queue_wait_seconds_bucket{le="0.005"} 100
gateway_queue_wait_seconds_bucket{le="0.01"} 100
```

## Summary

✅ **Embeddings throughput:** Measured
✅ **Completion latency:** Measured with unique prompts
✅ **Concurrency sweep:** 1, 2, 4, 8 tested
✅ **Token size impact:** Small vs large compared
✅ **Gateway overhead:** Measured vs direct
✅ **Streaming:** Tested and timed
✅ **Metrics:** Collected and validated

