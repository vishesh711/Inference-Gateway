# Quick Reference Card

## The Numbers (Memorize These)

| Metric | Value | Context |
|--------|-------|---------|
| Token generation | **75-80 tok/s** | Backend (llama.cpp) |
| Gateway overhead | **<50ms** | Additional latency |
| Embeddings throughput | **117+ req/s** | Sustained load |
| Queue wait (p95) | **<5ms** | Under normal load |
| Success rate | **100%** | No failures in testing |
| Metrics instrumented | **23** | Prometheus histograms |

## The One-Minute Pitch (60 seconds)

> "I built an LLM serving gateway in Go. The core problem is that clients submit faster than a GPU can serve, and the naive fix—an unbounded queue—turns a capacity problem into a latency disaster where everything times out.
>
> So admission is bounded and rejects with 429 immediately. But the piece I'd point at is **token-aware scheduling**: requests consume budget proportional to estimated token cost, because a 50-token and a 2000-token completion tie up the engine for very different lengths of time.
>
> I also implemented request coalescing for embeddings, where the batch timer starts on the first arrival rather than every arrival—resetting per-arrival means a steady trickle never dispatches.
>
> Everything is instrumented with histograms, separating queue wait from generation time. I've tested against llama.cpp with TinyLlama 1.1B—the backend generates at 75-80 tokens per second, my gateway adds less than 50ms overhead, and I measured 117 requests per second for embeddings."

## Resume Bullets (Copy These)

**1. Token-aware scheduling (LEAD WITH THIS):**
> Implemented token-aware admission using a weighted semaphore, so a request consumes scheduler budget proportional to its estimated token cost rather than counting as one slot, since a 50-token and a 2000-token completion occupy the engine for very different durations

**2. Measured performance:**
> Tested against llama.cpp with TinyLlama 1.1B Q4, measured 75-80 tokens/second generation speed with <50ms gateway overhead, and sustained 117+ requests/second for embeddings endpoint

**3. Admission control:**
> Built an LLM serving gateway in Go implementing admission control with a bounded queue and semaphore-limited concurrency, so overload produces immediate 429 backpressure rather than a timeout cascade

**4. Request batching:**
> Implemented request coalescing for embeddings with size and time triggers, starting the batch timer on the first arrival rather than on every arrival, since resetting per-arrival means a steady trickle never dispatches

**5. Multi-backend routing:**
> Built multi-backend routing with weighted selection based on (load/capacity) + (p95_latency × 10) and circuit breakers that open after 3 consecutive failures

**6. Instrumentation:**
> Instrumented with 23 Prometheus metrics using histograms rather than averages, separating queue wait from generation time so a backlog is distinguishable from slow inference

## Top 5 Interview Questions

### 1. "What performance did you measure?"
> "I benchmarked against llama.cpp with TinyLlama 1.1B. The backend generates at 75-80 tokens per second, and I measured my gateway adding less than 50ms overhead. For embeddings, I sustained over 117 requests per second.
>
> An interesting challenge: llama.cpp implements KV caching, so repeated prompts return in under 20ms. That's a production optimization—all modern LLM servers do this—but it makes measuring peak completion throughput difficult. For completions, throughput is generation-bound rather than gateway-bound. The gateway isn't the bottleneck."

### 2. "Why token-aware instead of request-based scheduling?"
> "LLM inference time is dominated by token count. A 50-token request and a 2000-token request differ 40× in GPU time. Request-based treats them equally, which causes starvation. Token-based schedules by actual work."

### 3. "What's the ideal concurrency limit?"
> "Depends on the model and hardware. You sweep max_in_flight from 1 to 32, measure throughput and p95 latency at each level, and look for the knee where latency climbs without throughput gain. That's your optimal point."

### 4. "Why bounded queue?"
> "Unbounded queues cause timeout cascades. During overload, requests pile up until they all timeout, but the server keeps processing responses no one will receive. A bounded queue rejects with 429 immediately when full—explicit backpressure within milliseconds, not after 30 seconds."

### 5. "What would you do differently?"
> "Add prefix-aware routing to maintain KV cache locality—if similar prompts go to the same backend, cache hit rate improves. Implement gradual ramp-up after circuit breaker closes. Use `container/list` for O(1) LRU cache updates instead of O(n) slice scan. Sample 1% with real tokenizer to calibrate estimation."

## Tech Stack (One Line Each)

- **Language:** Go 1.21+ (true concurrency, no GIL, 2KB goroutine stacks)
- **HTTP:** net/http stdlib (production-grade, no framework needed)
- **Concurrency:** Weighted semaphore from golang.org/x/sync/semaphore
- **Metrics:** Prometheus client_golang with histogram support
- **Testing:** llama.cpp with TinyLlama 1.1B Q4_K_M (638MB model)

## Key Design Decisions

1. **Token-aware scheduling** - LLM-specific insight (novel)
2. **Bounded queue + semaphore** - Prevents timeout cascades (textbook)
3. **Batch timer starts on first arrival** - Subtle, most people get wrong
4. **defer for slot release** - Panic-safe (production pattern)
5. **Context propagation** - Client disconnect frees slot immediately
6. **Weighted routing** - Adaptive load balancing (not round-robin)
7. **Circuit breakers (3-strike)** - Fault isolation and recovery
8. **Histograms not averages** - p95 latency is what users feel

## What NOT to Say

❌ "44,000 req/s" (mock benchmark, invalid)  
❌ "97% success rate" (from mock)  
❌ "Zero bugs vs Copilot's 9" (removed)  
❌ "Production-ready" (personal project)  
❌ "67% cache hit rate" (functional test, not real rate)

## What to Emphasize

✅ Token-aware scheduling (LLM-specific, novel)  
✅ Real numbers (75-80 tok/s, 117 req/s, <50ms)  
✅ Design reasoning (timeout cascades, head-of-line blocking)  
✅ KV cache discovery (shows systems understanding)  
✅ Honest about challenges (measurement validity)

## The KV Cache Answer

> "llama.cpp implements KV caching for prompts. First prompt takes 224ms to process, subsequent similar prompts take 13ms—a 94% reduction. That's a production optimization all modern LLM servers use, but it makes synthetic load testing difficult because repeated prompts hit the cache and return instantly. Understanding this is important for measuring real-world performance correctly."

## Files to Reference

**For interview prep (15 min):** BENCHMARK_SUMMARY.md  
**For deep dive (30 min):** FINAL_BENCHMARK_RESULTS.md  
**For comprehensive study (1-2 hrs):** docs/interview/INTERVIEW_GUIDE_CORRECTED.md

## Architecture (One Diagram)

```
Client → Handler → Cache → Admission Queue
                               ↓
                       Token Budget (20K)
                               ↓
                    Router (weighted routing)
                               ↓
               ┌───────────────┴───────────────┐
               ↓               ↓               ↓
          Backend 1       Backend 2       Backend 3
          (healthy)       (healthy)    (circuit open)
               ↑               ↑               ↑
          Health Check (10s interval)
```

## Final Check Before Interview

- [ ] Can recite one-minute pitch from memory
- [ ] Know the numbers: 75-80, 117, <50ms, <5ms, 23 metrics
- [ ] Can explain token-aware scheduling in 30 seconds
- [ ] Understand KV cache challenge (strength, not weakness)
- [ ] Have "what would you do differently" ready
- [ ] Can explain: bounded queue, timer logic, weighted routing
- [ ] Know which claims to avoid (mock benchmarks, Copilot comparison)

---

**You're ready. Lead with token-aware scheduling. Support with real numbers. Good luck!**

