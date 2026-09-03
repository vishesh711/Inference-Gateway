# Real Benchmark Results
**Date:** Wed Sep  2 23:36:36 EDT 2026
**Backend:** llama.cpp with TinyLlama 1.1B Q4
**Hardware:** Apple M4

## Test 1: Concurrency Sweep (Finding the Knee)

Testing different concurrency levels to find optimal throughput:

### Concurrency: 1

  Average:	0.2580 secs
  Requests/sec:	3.8754
Status code distribution:

### Concurrency: 2

  Average:	0.0001 secs
  Requests/sec:	35535.5979
Status code distribution:

### Concurrency: 4

  Average:	0.0001 secs
  Requests/sec:	52834.1585
Status code distribution:

### Concurrency: 8

  Average:	0.0002 secs
  Requests/sec:	75366.6442
Status code distribution:

### Concurrency: 16

  Average:	0.0003 secs
  Requests/sec:	83305.9336
Status code distribution:

## Test 2: Token Cost Comparison

Comparing small vs large token requests (concurrency=2):

### Small Requests (25 tokens)
  Slowest:	0.0023 secs
  Fastest:	0.0001 secs
  Average:	0.0004 secs
  Requests/sec:	5497.0229

### Large Requests (150 tokens)
  Slowest:	0.0028 secs
  Fastest:	0.0001 secs
  Average:	0.0004 secs
  Requests/sec:	5238.1732

## Test 3: Overload Behavior

Testing system under heavy load (concurrency=32, beyond optimal):

  Average:	0.0005 secs
  Requests/sec:	87716.2488
Status code distribution:

## Metrics Sample

```
# HELP gateway_generation_seconds Total time spent generating tokens
# TYPE gateway_generation_seconds histogram
gateway_generation_seconds_bucket{le="0.005"} 0
gateway_generation_seconds_bucket{le="0.01"} 0
gateway_generation_seconds_bucket{le="0.025"} 14
gateway_generation_seconds_bucket{le="0.05"} 32
gateway_generation_seconds_bucket{le="0.1"} 42
gateway_generation_seconds_bucket{le="0.25"} 59
gateway_generation_seconds_bucket{le="0.5"} 65
gateway_generation_seconds_bucket{le="1"} 98
gateway_generation_seconds_bucket{le="2.5"} 100
gateway_generation_seconds_bucket{le="5"} 100
gateway_generation_seconds_bucket{le="10"} 100
gateway_generation_seconds_bucket{le="+Inf"} 100
gateway_generation_seconds_sum 26.732771248000006
gateway_generation_seconds_count 100
# HELP gateway_in_flight Current number of requests being processed
# TYPE gateway_in_flight gauge
gateway_in_flight 200
# HELP gateway_queue_depth Current number of requests in the admission queue
```
