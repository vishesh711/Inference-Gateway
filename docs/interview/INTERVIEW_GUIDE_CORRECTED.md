# Inference Gateway — Interview Guide (Corrected)

**Status:** ⚠️ See BENCHMARK_STATUS.md for current testing status

---

## The One-Minute Pitch

> "I built an LLM serving gateway in Go. The core problem is that clients submit faster than a GPU can serve, and the naive fix—an unbounded queue—turns a capacity problem into a latency disaster where everything times out while the server keeps working.
>
> So admission is bounded and rejects with 429 immediately. But the piece I'd point at is **token-aware scheduling**: requests consume budget proportional to estimated token cost rather than counting as one slot each, because a 50-token and a 2000-token completion tie up the engine for very different lengths of time.
>
> I also implemented request coalescing for embeddings, where the batch timer starts on the first arrival rather than every arrival, since resetting per-arrival means a steady trickle never dispatches.
>
> Everything is instrumented with histograms rather than averages, separating queue wait from generation time so you can tell a backlog from slow inference. I've verified the control paths functionally. I still need to benchmark against a real engine, since my current numbers came from a mock responding in microseconds and aren't meaningful."

---

## Resume Bullets (Use These)

**Token-aware scheduling (lead with this):**
> Implemented token-aware admission using a weighted semaphore, so a request consumes scheduler budget proportional to its estimated token cost rather than counting as one slot, since a 50-token and a 2000-token completion occupy the engine for very different durations

**Admission control:**
> Built an LLM serving gateway in Go implementing admission control with a bounded queue and semaphore-limited concurrency, so overload produces immediate 429 backpressure rather than a timeout cascade where every request expires while the server keeps working

**Request batching:**
> Implemented request coalescing for embeddings with size and time triggers, starting the batch timer on the first arrival rather than on every arrival, since resetting per-arrival means a steady trickle never dispatches

**Context cancellation:**
> Propagated context cancellation end to end so a client disconnect frees its concurrency slot immediately, and released slots through `defer` so a panic cannot permanently leak capacity

**Multi-backend routing:**
> Built multi-backend routing with weighted selection based on `(load/capacity) + (p95_latency × 10)` and circuit breakers that open after 3 consecutive failures, providing automatic fault isolation and recovery

**SSE streaming:**
> Implemented SSE streaming with separate TTFT (time to first token) and TPOT (time per output token) metrics, since TTFT measures perceived latency while TPOT measures generation quality

**Instrumentation:**
> Instrumented with 23 Prometheus metrics using histograms rather than averages, separating queue wait from generation time so a backlog is distinguishable from slow inference

---

## What's Been Validated

### ✅ Functional Tests (All Pass)
- Admission control: Queue full → 429 rejection
- Cache: Hit and miss paths work
- Context cancellation: Client disconnect frees slot
- Graceful shutdown: In-flight work completes
- Metrics: All 23 metrics export correctly
- Streaming: SSE format and TTFT/TPOT tracking
- Multi-backend: Router selects backends, circuit breakers open/close
- Token scheduling: Budget acquire/release works

### ⏳ Performance Benchmarks (Pending)
- Mock benchmarks invalid (see BENCHMARK_STATUS.md)
- Need real llama.cpp benchmarks
- Will measure concurrency knee
- Will measure batching improvement

---

## Key Design Decisions

### 1. Token-Aware Scheduling (LLM-Specific)

**The insight:** Treating all requests as "1 slot" is wrong for LLMs. A 50-token request takes 50ms; a 2000-token request takes 2 seconds. Naive concurrency limits cause starvation.

**Implementation:**
```go
// Weighted semaphore
cost := EstimateTokens(prompt) + maxTokens
sem.Acquire(ctx, cost)  // Acquire 'cost' units, not just 1
defer sem.Release(cost)
```

**Why it matters:** Fair scheduling by actual GPU work, not HTTP request count.

> "This is the correct insight for LLM serving. Almost nobody applying to inference infrastructure roles will have built this."

### 2. Bounded Queue + Semaphore

**Problem:** Unbounded queue → timeout cascade (every request times out)

**Solution:** 
- Bounded admission queue (e.g., 100 slots)
- Semaphore caps concurrency (e.g., 8 in-flight)
- Full queue → immediate 429 rejection

**Why both?**
- Queue absorbs bursts
- Semaphore prevents engine saturation

> "Bounded queue over unbounded prevents the timeout cascade problem. This is textbook admission control."

### 3. Batch Timer Subtlety

**Wrong way:**
```go
for req := range requests {
    timer = time.After(20ms)  // WRONG: resets every request
}
```

A steady trickle at 15ms intervals never dispatches.

**Right way:**
```go
if len(batch) == 1 {
    timer = time.After(20ms)  // Start ONLY on first request
}
```

**Questions:** "What if you reset the timer per-request?"
> "A steady trickle never dispatches. If requests arrive every 15ms and your window is 20ms, you reset before dispatching. The timer must start on the first arrival."

### 4. Panic-Safe Slot Release

```go
defer scheduler.Release()  // Runs even on panic
```

Without `defer`, a panic leaks a slot permanently, reducing capacity.

### 5. Context Cancellation Propagation

```go
if err := engine.CreateCompletion(ctx, req); err != nil {
    // ctx.Done() fires if client disconnected
}
```

Client disconnect → goroutine exits → slot freed immediately.

### 6. Weighted Multi-Backend Routing

**Score function:**
```go
score = (load / capacity) + (p95_latency * 10)
```

**Why 10×?** Balances load ratio (0-1) with latency (seconds). Backend at 50% load with 100ms p95 gets score 1.5.

**Adaptive:** If a backend slows down, its score increases → traffic shifts away automatically.

### 7. Circuit Breaker (3-Strike)

```go
failures >= 3 → circuit opens (stop routing)
successes >= 3 → circuit closes (resume routing)
```

**Why 3?** One failure = transient blip. Three = clear signal.

### 8. Histograms Not Averages

```go
histogram_quantile(0.95, rate(gateway_latency_seconds_bucket[5m]))
```

**Why:** p95 latency is what users feel. Averages hide tail latency.

---

## Questions You'll Be Asked

**"What's the ideal concurrency limit?"**
> "Depends on the model and hardware. You sweep max_in_flight from 1 to 32, measure throughput and p95 latency at each level, and look for the knee where latency climbs without throughput gain. That's your optimal point."

**"Why token-based instead of request-based?"**
> "LLM inference time is dominated by token count. A 50-token request and a 2000-token request differ 40× in GPU time. Request-based treats them equally, which causes starvation and wastes capacity. Token-based schedules by actual work."

**"What if the token estimation is wrong?"**
> "If I consistently under-estimate, the system over-subscribes and latency spikes. If I over-estimate, I waste capacity. I'd sample 1% with a real tokenizer to calibrate. The ~4 chars/token heuristic is good enough for aggregate behavior."

**"Why weighted least-loaded instead of round-robin?"**
> "Round-robin assumes all backends are equal. If one backend slows down, round-robin keeps sending it 33% of traffic. Weighted routing adapts—the slow backend's score increases and traffic shifts away. It's self-healing."

**"What happens if all backends fail?"**
> "Router returns 503. The admission queue still works, so clients get fast rejections. That's correct—if capacity is zero, reject immediately rather than queueing indefinitely."

**"How do you handle the embeddings multi-model bug?"**
> "Currently the batcher doesn't partition by model—if requests for two models arrive in the same window, they'd be incorrectly batched. The fix is grouping by model before dispatch. I documented this limitation."

**"Does streaming cache?"**
> "Not currently—caching requires buffering the full response, which defeats streaming's low TTFT. You could cache after stream completes, but that's complex. Streaming requests bypass the cache."

**"What would you do differently?"**
> "Run real benchmarks against llama.cpp to get meaningful numbers. Add prefix-aware routing for KV cache locality. Implement gradual ramp-up after circuit closes to prevent thundering herd. Use `container/list` for O(1) LRU updates. Sample 1% with real tokenizer to calibrate estimation."

---

## What This Project Demonstrates

### Systems Engineering
- Admission control (bounded queue prevents cascades)
- Concurrency limiting (semaphore over worker pool)
- Graceful degradation (429 backpressure)
- Context propagation (cancellation, timeouts)
- Panic safety (`defer` for cleanup)

### Distributed Systems
- Multi-backend routing
- Health checks (automatic fault detection)
- Circuit breakers (fault isolation)
- Weighted load balancing (adaptive)
- Fault tolerance patterns

### LLM-Specific
- Token-aware scheduling (the key insight)
- Request coalescing (embeddings batching)
- SSE streaming (TTFT vs TPOT)
- Token estimation heuristics

### Observability
- 23 Prometheus metrics
- Histograms for percentiles
- Separated concerns (queue vs generation)
- Per-backend health tracking

---

## Current Status

**Code:** ✅ Complete, builds successfully  
**Functional Tests:** ✅ All verified  
**Performance Tests:** ⏳ Pending real benchmarks  
**Documentation:** ✅ Comprehensive, honest about limitations  

See **BENCHMARK_STATUS.md** for detailed status and real benchmark plan.

---

## Files to Reference

- **BENCHMARK_STATUS.md** — Current testing status
- **FIXES_NEEDED.md** — Documentation updates in progress
- **PHASE1_INTERVIEW_ADDITIONS.md** — Detailed Phase 1 features
- **README.md** — Technical overview
- **config.yaml** — Configuration examples

---

**This is the single source of truth for interview prep. All other docs being updated to match.**
