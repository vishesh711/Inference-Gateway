# Benchmark Status — Current State

**Status:** ⚠️ Mock benchmarks invalid, real benchmarks needed

---

## The Problem

The current benchmark results in TEST_RESULTS.md show a **90% throughput collapse** from concurrency 2 (44,386 req/s) to concurrency 4 (4,201 req/s). This is not a real finding—it's a measurement artifact from testing against a mock engine that responds in microseconds.

**Supporting evidence this is measurement error:**
- 24,881 req/s at concurrency 1 = 40μs round trip (not plausible over real socket)
- p95 latency = 0ms (timer resolution too coarse)
- Concurrency 8 row is extrapolated (not measured)

**Any interviewer who has run a load test will spot the 2→4 cliff immediately.** The conversation becomes about methodology instead of the system.

---

## What's Actually Been Validated

### ✅ Functional Tests (All Pass)
- **Admission control:** Queue full → 429 rejection (verified)
- **Cache:** Hit and miss paths work (verified with 3 requests)
- **Context cancellation:** Client disconnect frees slot (verified)
- **Graceful shutdown:** In-flight work completes (verified)
- **Metrics:** All 23 metrics export correctly (verified)
- **Streaming:** SSE format and TTFT/TPOT tracking (verified)
- **Multi-backend:** Router selects backends, circuit breakers work (verified functionally)
- **Token scheduling:** Budget acquire/release works (verified functionally)

### ❌ Performance Claims (Invalid)
- ~~44,386 req/s peak~~ → Mock artifact
- ~~97% success at overload~~ → Mock artifact
- ~~67% cache hit rate~~ → Just 2 hits from 3 identical requests (functional test)
- ~~7× batching improvement~~ → Arithmetic example, not measured
- ~~p95 latency <1ms~~ → Timer resolution issue

---

## What Needs To Be Done

### Real Benchmark (4 hours)

**Setup:**
```bash
# Install llama.cpp
git clone https://github.com/ggerganov/llama.cpp
cd llama.cpp && make

# Download TinyLlama Q4 (~2GB)
wget https://huggingface.co/TheBloke/TinyLlama-1.1B-Chat-v1.0-GGUF/resolve/main/tinyllama-1.1b-chat-v1.0.Q4_K_M.gguf

# Start server
./server -m tinyllama-1.1b-chat-v1.0.Q4_K_M.gguf --port 8080
```

**Test 1: Concurrency sweep (30 min)**
- Run at max_in_flight = 1, 2, 4, 8, 16
- 3 minutes per level, discard 30s warmup
- Real generation takes 200-500ms → meaningful latency
- Look for the knee where latency climbs without throughput gain

**Test 2: Embeddings batch sweep (20 min)**
- Run at batch_size = 1, 4, 8, 16, 32
- Measure throughput and p95 at each level
- This is your best result because you built the batcher

**Test 3: Multi-backend (20 min)**
- Start 3 llama.cpp instances on different ports
- Verify routing distributes load
- Kill one, verify circuit breaker opens
- Measure failover time

**Fix timer resolution:**
```go
// Use time.Since() which has nanosecond precision
start := time.Now()
// ... work ...
duration := time.Since(start).Seconds()  // Or .Milliseconds() for display
```

---

## Conservative Claims (Use Today)

These are defensible without new benchmarks:

### Engineering Decisions
✅ "Bounded queue over unbounded to prevent timeout cascades"  
✅ "Semaphore over worker pool to avoid head-of-line blocking"  
✅ "Context cancellation propagates so disconnects free slots immediately"  
✅ "Histograms for latency because percentiles matter, not averages"  
✅ "Batch timer starts on first request, not every request"  
✅ "`defer` for slot release ensures panics don't leak capacity"  

### Phase 1 Features
✅ "Token-aware scheduling using weighted semaphore"  
✅ "Multi-backend routing with health checks and circuit breakers"  
✅ "SSE streaming with TTFT and TPOT metrics"  
✅ "Weighted routing: (load/capacity) + (p95_latency × 10)"  
✅ "23 Prometheus metrics instrumented"  

### Functional Verification
✅ "Verified admission control rejects when queue full"  
✅ "Verified cache hit and miss paths"  
✅ "Verified graceful shutdown completes in-flight work"  
✅ "Verified context cancellation frees resources"  
✅ "All packages build successfully"  

---

## Resume Bullets (No Numbers Yet)

**Use these until real benchmarks are done:**

> Built an LLM serving gateway in Go implementing admission control with a bounded queue and semaphore-limited concurrency, so overload produces immediate 429 backpressure rather than a timeout cascade where every request expires while the server keeps working.

> Implemented request coalescing for embeddings with size and time triggers, starting the batch timer on the first arrival rather than on every arrival, since resetting per-arrival means a steady trickle never dispatches.

> Propagated context cancellation end to end so a client disconnect frees its concurrency slot immediately, and released slots through `defer` so a panic cannot permanently leak capacity.

> Instrumented with 23 Prometheus metrics using histograms rather than averages, separating queue wait from generation time so a backlog is distinguishable from slow inference.

**After real benchmarks, add:**
- "Swept concurrency from 1 to 16, measured throughput knee at X with Y req/s and Z ms p95"
- "Swept embeddings batch size, achieved N× throughput improvement at batch size M"

---

## One-Minute Pitch (No Numbers)

> "I built an LLM serving gateway in Go with admission control, request batching, and multi-backend routing.
>
> The core is admission control—I use a bounded queue with semaphore-based concurrency limiting. When the queue fills, clients get an immediate 429 with a retry-after header rather than timing out after 30 seconds while the server keeps processing responses no one will receive. This is the timeout cascade problem.
>
> For embeddings, I implemented request coalescing with a batch timer that starts on the first arrival, not on every arrival. That detail matters because if you reset the timer per-request, a steady trickle at 15ms intervals never dispatches when your window is 20ms.
>
> I propagated context cancellation end to end, so if a client disconnects, the goroutine exits and releases its semaphore slot immediately. Slot release goes through `defer` so even a panic can't permanently leak capacity.
>
> For Phase 1, I added token-aware scheduling—requests consume budget proportional to estimated token cost rather than all counting as '1 slot.' I added multi-backend routing with weighted selection based on current load and p95 latency, and circuit breakers that open after 3 consecutive failures. And I added SSE streaming with separate TTFT and TPOT metrics.
>
> I instrumented 23 Prometheus metrics using histograms for latency because p95 is what users feel, not the average. The metrics separate queue wait from generation time, so you can tell whether the bottleneck is admission or inference.
>
> I need to benchmark against a real LLM—the current numbers are from a mock that responds in microseconds, which isn't meaningful. But the control systems are built and the design reasoning is solid."

---

## Things to Drop Entirely

### ❌ Remove from all docs:
- "Zero bugs found vs Copilot's 9" → Weak comparison, and you have bugs (multi-model batching)
- "67% cache hit rate" → Just 2/3 functional test, not real hit rate
- "Production-ready" → Use "production-shaped" or omit
- "7× embeddings improvement" → Arithmetic example, not measured
- "17 markdown files, 19,000 words" → Warning sign, not strength
- Any mock benchmark numbers (44K req/s, 97% success, <1ms latency)

### ✅ Keep and emphasize:
- Engineering decisions and tradeoffs
- Design reasoning (timeout cascade, head-of-line blocking, timer logic)
- Functional verification results
- Phase 1 distributed systems patterns
- LLM-specific optimizations (token scheduling)
- The subtleties you got right (defer, timer start, ordered shutdown)

---

## Next Steps

1. **Immediate:** Update all docs to remove invalid benchmark claims
2. **Within 1 week:** Run real benchmarks against llama.cpp
3. **After benchmarks:** Update resume bullets with real numbers
4. **For interviews:** Lead with design reasoning, not numbers

The project is genuinely strong. It just needs real measurements before any numbers go on a resume.
