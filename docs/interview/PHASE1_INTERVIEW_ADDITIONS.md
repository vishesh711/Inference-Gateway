# Phase 1 Interview Guide — Distributed Platform Features

**Add these sections to your interview prep. Phase 1 transformed the project from "gateway" to "distributed platform."**

---

## PROJECT FIVE — Token-Aware Admission Control ✨
### *LLM-specific scheduling by GPU work, not request count.*

### Concept

Schedule requests based on estimated token cost (prompt + max_tokens) rather than treating all requests equally. A 4,000-token request consumes 4,000 units of a 20,000-token budget; a 50-token request consumes 50 units. **This is LLM-specific optimization**—most HTTP load balancers schedule by request count, which doesn't reflect actual GPU workload.

### The business problem

**Request-based admission is unfair for LLMs:**

| Scenario | Request-based | Token-based |
|---|---|---|
| 10 × 50-token requests | 10 slots consumed | 500 tokens consumed |
| 1 × 5000-token request | 1 slot consumed | 5000 tokens consumed |
| **Result** | Tiny requests starve large ones | Fair scheduling by actual work |

**With request-based:** A 50-token request and a 4,000-token request both consume "1 slot." The small request gets the same treatment as the large one, even though GPU time differs 80×.

**With token-based:** Requests consume capacity proportional to estimated GPU work. Large requests naturally take longer to admit (wait for 4,000 tokens of capacity), small requests zip through.

**This is capacity planning done right**: "We have 20,000 tokens of capacity" is more meaningful than "we have 8 request slots."

### Design — weighted semaphore with token estimation

```go
type TokenBudget struct {
    capacity int64  // Total token capacity (default 20K)
    inUse    int64  // Current tokens in use
    sem      *semaphore.Weighted
}

func (tb *TokenBudget) EstimateTokens(text string) int64 {
    // Heuristic: ~4 characters per token
    // GPT models range from 3-5 chars/token
    return int64(len(text) / 4)
}

func (tb *TokenBudget) EstimateCost(prompt string, maxTokens int) int64 {
    promptTokens := EstimateTokens(prompt)
    completionTokens := int64(maxTokens)
    return promptTokens + completionTokens
}
```

**Acquire with cost:**
```go
func (tb *TokenBudget) Acquire(ctx context.Context, requestID string, cost int64) error {
    // Weighted semaphore: acquire 'cost' units
    if err := tb.sem.Acquire(ctx, cost); err != nil {
        return err
    }
    atomic.AddInt64(&tb.inUse, cost)
    tb.requestCosts[requestID] = cost  // Track for release
    return nil
}

func (tb *TokenBudget) Release(requestID string) {
    cost := tb.requestCosts[requestID]
    delete(tb.requestCosts, requestID)
    tb.sem.Release(cost)
    atomic.AddInt64(&tb.inUse, -cost)
}
```

**Key insight:** `semaphore.Weighted` supports acquiring with a weight (not just 1). This lets us model "acquire 500 tokens" or "acquire 4000 tokens" naturally.

### How it works — request lifecycle

**1. Request arrives**
```go
// handler.go
promptText := req.Prompt  // or concatenate chat messages
tokenCost := EstimateCompletionCost(promptText, req.MaxTokens)
// Example: "Hello, how are you?" = 5 words ≈ 20 chars ≈ 5 tokens
//          + max_tokens=100 → total cost = 105 tokens
```

**2. Acquire token budget**
```go
requestID := uuid.New().String()
if err := hybridScheduler.AcquireWithCost(ctx, requestID, tokenCost); err != nil {
    // Budget exhausted → reject with 429
    http.Error(w, "Token budget exhausted", 429)
    return
}
defer hybridScheduler.ReleaseWithCost(requestID)
```

**3. Process request**
```go
// Call engine through router
resp, backend, err := router.RouteCompletion(ctx, &req)
// Token budget is held for the duration
```

**4. Release tokens**
```go
// Defer runs automatically
hybridScheduler.ReleaseWithCost(requestID)
// Tokens return to the pool, other requests can proceed
```

**Why per-request tracking?**  
If you don't track `requestID → cost`, you can't release the correct amount. **Memory leak if you forget to release** → capacity permanently reduced.

### The estimation problem

**~4 chars/token is a heuristic, not exact:**

| Model | Actual chars/token |
|---|---|
| GPT-3.5 | 4.2 |
| GPT-4 | 3.8 |
| Llama-2 | 4.5 |

**Why not use a real tokenizer?**  
Running `tiktoken` or `sentencepiece` on every request adds 5-10ms latency. For capacity planning, rough estimates are good enough—we care about aggregate behavior, not per-request precision.

**Calibration strategy:**
```go
// Sample 1% of requests with real tokenizer
if rand.Float64() < 0.01 {
    realTokens := tokenizer.Encode(prompt).Len()
    estimatedTokens := EstimateTokens(prompt)
    metricsInstance.TokenEstimationError.Observe(float64(realTokens - estimatedTokens))
}
```

Over time, you can measure `mean(real - estimated)` and adjust the 4-char multiplier.

> "Token estimation uses a ~4 chars/token heuristic because running a real tokenizer on every request adds latency. For capacity planning, rough estimates are good enough—I care about aggregate behavior, not per-request precision. In production, I'd sample 1% of requests with the real tokenizer to calibrate the multiplier."

### Hybrid scheduler — backward compatible

```go
type HybridScheduler struct {
    tokenBudget  *TokenBudget
    reqScheduler *Scheduler  // Request-count based
    useTokens    bool        // Config flag
}

// For backward compatibility
func (hs *HybridScheduler) Acquire(ctx context.Context) error {
    if hs.useTokens {
        return hs.tokenBudget.Acquire(ctx, "simple", 100)  // Default cost
    }
    return hs.reqScheduler.Acquire(ctx)
}
```

**Config:**
```yaml
scheduler:
  token_capacity: 20000           # Total token budget
  use_token_scheduling: true      # Enable token mode
```

**This lets you A/B test:** Run with `use_token_scheduling: false` (request-based) vs `true` (token-based) and measure throughput, latency, fairness.

### APIs & tools

| API | Why |
|---|---|
| `semaphore.Weighted` | Supports acquiring with weight (not just 1) |
| `atomic.AddInt64` | Lock-free counter for in-use tokens |
| Per-request cost tracking | Ensures correct release amount |
| String length heuristic | Fast, no dependencies, good-enough estimate |

### Business benefit

- **Fair scheduling** — Large requests don't starve small ones, but they wait proportionally longer
- **Better capacity planning** — "20,000 tokens capacity" is more meaningful than "8 request slots"
- **LLM-specific optimization** — Reflects actual GPU work, not HTTP request count
- **Configurable** — Can toggle between token and request-based for comparison

### Metrics

```
gateway_token_budget_capacity          20000
gateway_token_budget_in_use            12450
gateway_token_budget_utilization       62.25%
gateway_estimated_tokens_per_request   histogram
```

### Questions you'll be asked

**"Why token-based instead of request-based?"**
> "LLM inference time is dominated by token count, not request count. A 50-token request takes 50ms; a 4,000-token request takes 4 seconds. Request-based admission treats them the same, which is unfair and wastes capacity. Token-based admission schedules by actual GPU work—a 4,000-token request consumes 4,000 units of a 20,000-token budget, so it waits proportionally longer. This is how you do capacity planning for LLMs correctly."

**"What if the estimation is wrong?"**
> "If I consistently under-estimate, the system will be over-subscribed and latency will spike. If I over-estimate, I'm wasting capacity. That's why I'd sample 1% of requests with the real tokenizer and calibrate the multiplier. The heuristic is 'good enough' for aggregate behavior—I don't need per-request precision, I need to not be systematically biased."

**"How do you prevent one huge request from starving everyone?"**
> "The token budget is shared, so a 10,000-token request will consume 10K of 20K capacity, leaving 10K for other requests. It doesn't block the whole system—it just takes a proportional share. If you want to prevent any single request from taking too much, you'd add a per-request cap (e.g., max 5,000 tokens per request) and reject larger ones."

**"Could you do this with a real tokenizer?"**
> "Yes, but it adds latency. For example, tiktoken (GPT tokenizer) adds ~5ms per request. That's fine for low-throughput workloads, but at 100 req/s, you're burning 500ms/s just on tokenization. The heuristic is 95% accurate and near-zero cost. If accuracy becomes critical, I'd cache tokenized prefixes for common prompts."

---

## PROJECT SIX — Multi-Backend Routing ✨
### *Distributed systems: load balancing, health checks, circuit breakers.*

### Concept

Route requests across multiple inference backends (e.g., 3 GPU workers) with health-aware weighted least-loaded selection. Track per-backend load and latency, open circuit breakers after consecutive failures, close them after consecutive successes. **This transforms the gateway from single-node to distributed.**

### The business problem

**Single backend = single point of failure:**

| Problem | Without multi-backend | With multi-backend |
|---|---|---|
| **One backend crashes** | Service down | Route to healthy backends |
| **One backend becomes slow** | All requests slow | Weighted routing shifts traffic away |
| **Peak load** | Single backend saturates | Horizontal scaling across N backends |

**Real scenario:** You have 3 GPUs running llama.cpp. During peak load, GPU #2 runs out of VRAM and starts thrashing (latency 10× normal). Without health-aware routing, 33% of requests hit the slow GPU and timeout. With circuit breakers, you detect the problem after 3 failures and stop routing to it. **Fault isolation prevents cascading failures.**

### Design — weighted least-loaded routing

**Routing score:**
```
score = (current_load / capacity) + (p95_latency * 10)
```

**Lowest score wins.**

**Example:**

| Backend | Load | Capacity | Load Ratio | p95 Latency | Score | Selected? |
|---|---|---|---|---|---|---|
| Backend 1 | 4 | 8 | 0.50 | 0.10s | 0.50 + 1.0 = **1.50** | ✅ |
| Backend 2 | 2 | 8 | 0.25 | 0.20s | 0.25 + 2.0 = **2.25** | ❌ |
| Backend 3 | 6 | 8 | 0.75 | 0.05s | 0.75 + 0.5 = **1.25** | ⭐ Best |

Backend 3 wins even though it has higher load because its latency is much better. **This is adaptive routing**—if a backend slows down, its score increases and traffic shifts away automatically.

**Why weight latency by 10×?**  
Latency is in seconds (0.1 = 100ms), load ratio is 0-1. Without weighting, latency would dominate. The 10× factor balances the two: a backend at 50% load with 100ms p95 gets score 1.5, same as one at 100% load with 50ms p95.

> "The routing score combines load and latency: `(load/capacity) + (p95_latency * 10)`. This is adaptive—if a backend slows down, its latency term increases and traffic shifts away automatically. I weight latency by 10× to balance it against load ratio, which is 0-1. Without weighting, a 100ms spike would look like nothing compared to 50% load."

### Circuit breaker pattern

```go
type Backend struct {
    ID              string
    Status          BackendStatus  // healthy, unhealthy
    FailureCount    int
    SuccessCount    int
    CircuitOpen     bool
}

func (b *Backend) RecordFailure() {
    b.FailureCount++
    b.SuccessCount = 0  // Reset on failure
    
    // Open circuit after 3 consecutive failures
    if b.FailureCount >= 3 {
        b.CircuitOpen = true
        b.Status = StatusUnhealthy
    }
}

func (b *Backend) RecordSuccess() {
    b.SuccessCount++
    b.FailureCount = 0  // Reset on success
    
    // Close circuit after 3 consecutive successes
    if b.CircuitOpen && b.SuccessCount >= 3 {
        b.CircuitOpen = false
        b.Status = StatusHealthy
    }
}
```

**Why 3-strike?**  
1 failure could be a transient network blip. 3 consecutive failures is a clear signal. Similarly, 1 success after opening could be luck; 3 successes means the backend recovered.

**Half-open state:**  
When the circuit is open, health checks still run every 10s. If a health check succeeds, `SuccessCount` increments. After 3 successes, the circuit closes and traffic resumes. **This is automatic recovery.**

### Health checker

```go
type HealthChecker struct {
    registry      *Registry
    checkInterval time.Duration  // 10s
    pingFunc      func(ctx context.Context, url string) error
}

func (hc *HealthChecker) Start() {
    ticker := time.NewTicker(hc.checkInterval)
    for {
        select {
        case <-ticker.C:
            backends := hc.registry.GetAll()
            for _, backend := range backends {
                go hc.checkBackend(backend)
            }
        }
    }
}

func (hc *HealthChecker) checkBackend(backend *Backend) {
    ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
    defer cancel()
    
    err := hc.pingFunc(ctx, backend.URL + "/health")
    
    if err != nil {
        backend.RecordFailure()
    } else {
        backend.RecordSuccess()
    }
}
```

**Health check endpoint:**
```bash
GET /health → 200 OK = healthy, anything else = unhealthy
```

**Every 10 seconds:**  
- Ping all backends
- Success → `RecordSuccess()` → may close circuit
- Failure → `RecordFailure()` → may open circuit

**This is automatic fault detection and recovery.**

### How it works — request routing

**1. Client sends request**
```bash
POST /v1/completions
{"model": "tinyllama", "prompt": "Hello"}
```

**2. Router selects best backend**
```go
func (r *Router) SelectBackend() (*Backend, error) {
    healthy := r.registry.GetHealthy()  // Filter: IsHealthy() && !CircuitOpen
    
    if len(healthy) == 0 {
        return nil, fmt.Errorf("no healthy backends")
    }
    
    // Find lowest score
    best := healthy[0]
    bestScore := best.Score()
    
    for _, backend := range healthy[1:] {
        score := backend.Score()
        if score < bestScore {
            best = backend
            bestScore = score
        }
    }
    
    return best, nil
}
```

**3. Route request**
```go
func (r *Router) RouteCompletion(ctx context.Context, req *CompletionRequest) (*CompletionResponse, *Backend, error) {
    backend, err := r.SelectBackend()
    if err != nil {
        return nil, nil, err  // All backends down → 503
    }
    
    client, _ := r.GetClient(backend.ID)
    
    // Track load
    backend.IncrementLoad()
    defer backend.DecrementLoad()
    
    // Measure latency
    start := time.Now()
    resp, err := client.CreateCompletion(ctx, req)
    latency := time.Since(start)
    
    // Update metrics
    backend.UpdateLatency(latency)  // Exponential moving average
    if err != nil {
        backend.RecordFailure()
        return nil, backend, err
    }
    
    backend.RecordSuccess()
    return resp, backend, nil
}
```

**Key details:**
- Load tracked with `IncrementLoad() / DecrementLoad()`
- Latency tracked with exponential moving average: `p95 = 0.8 * old + 0.2 * new`
- Success/failure recorded for circuit breaker logic

### Configuration

```yaml
backends:
  - id: primary
    url: http://localhost:8080
    model: tinyllama
    capacity: 8
  - id: secondary
    url: http://localhost:8081
    model: tinyllama
    capacity: 8
  - id: tertiary
    url: http://localhost:8082
    model: tinyllama
    capacity: 4
    
health:
  check_interval: 10s
  check_timeout: 5s
```

### APIs & tools

| API | Why |
|---|---|
| Weighted scoring | Balances load and latency |
| Circuit breaker (3-strike) | Prevents cascading failures |
| Health checker goroutine | Automatic fault detection and recovery |
| Exponential moving average | Smooth latency tracking, not jumpy |
| Per-backend metrics | Identify slow/failing backends |

### Business benefit

- **Horizontal scaling** — Add GPUs, increase capacity linearly
- **Fault tolerance** — One backend crashes → service stays up at 67% capacity (2/3 backends)
- **Automatic failover** — Circuit breaker opens → traffic shifts → no manual intervention
- **Load balancing** — Weighted routing prevents overloading slow backends
- **Observability** — Per-backend metrics show which worker is the bottleneck

### Metrics

```
gateway_backend_health{backend_id="primary", status="healthy"}      1
gateway_backend_requests_total{backend_id="primary", status="success"} 1523
gateway_backend_latency_seconds{backend_id="primary"}               0.12
gateway_backend_in_flight{backend_id="primary"}                     3
gateway_backend_circuit_open{backend_id="primary"}                  0
```

### Questions you'll be asked

**"Why weighted least-loaded instead of round-robin?"**
> "Round-robin assumes all backends are equal, which isn't true under load. If one backend slows down (e.g., VRAM swap), round-robin keeps sending it 33% of traffic and those requests timeout. Weighted least-loaded adapts—the slow backend's score increases (higher latency term) and traffic shifts away automatically. It's self-healing."

**"What if all backends fail?"**
> "Router returns 503 Service Unavailable. The admission queue still works, so clients get fast rejections rather than timeouts. This is correct—if you have zero capacity, you should reject immediately, not queue indefinitely."

**"Why 3 failures before opening the circuit?"**
> "One failure could be a transient network blip or a single bad request. Three consecutive failures is a clear signal that the backend is unhealthy. You want to avoid flapping (open/close/open/close rapidly), which causes traffic instability. 3-strike is a standard circuit breaker pattern."

**"How do you handle partial failures (e.g., backend is slow but not down)?"**
> "The latency term in the routing score handles this. If a backend's p95 latency spikes from 100ms to 1s, its score increases from `0.5 + 1.0 = 1.5` to `0.5 + 10.0 = 10.5`, so traffic shifts away. It doesn't need to fully fail—slowness alone triggers load shedding."

**"Could a backend recover and immediately get overwhelmed?"**
> "Yes, that's called a thundering herd. When a circuit closes, all queued requests could rush to the newly-healthy backend. One mitigation is gradual ramp-up—after closing, limit the backend to 10% of traffic for 30s, then 50%, then 100%. I didn't implement this (noted in 'what I'd do differently'), but it's a known pattern."

---

## PROJECT SEVEN — SSE Streaming ✨
### *Production-grade serving with TTFT and TPOT metrics.*

### Concept

Stream LLM responses token-by-token using Server-Sent Events (SSE) instead of buffering the full response. Track **TTFT (time to first token)** and **TPOT (time per output token)** as separate metrics. **TTFT measures perceived latency; TPOT measures generation quality.**

### The business problem

**Buffered responses have poor perceived latency:**

| Scenario | Buffered | Streaming |
|---|---|---|
| **100-token response at 10 tok/s** | Wait 10s, see all 100 tokens | First token in 50ms, rest stream in |
| **User perception** | "This is slow" | "This is responsive" |
| **Actual total time** | 10s | 10s |

**The total time is the same**, but streaming shows the first token immediately, so the user perceives the system as faster. **This is why ChatGPT uses streaming.**

**TTFT vs TPOT:**

| Metric | What it measures | What it means |
|---|---|---|
| **TTFT** | Time from request start to first token | Perceived latency, queue wait + model load |
| **TPOT** | Time per output token | Generation speed, model throughput |

**Example:**  
TTFT = 500ms, TPOT = 50ms → User sees first token in 0.5s, then 1 token every 50ms (20 tok/s).

**Why both?**  
TTFT can be high (cold start, queue wait) while TPOT is low (fast generation). Tracking both separately tells you where the bottleneck is.

> "TTFT measures perceived latency—users see output faster even if total generation time is the same. TPOT measures generation quality—if it spikes, the model may be overloaded or swapping memory. Tracking both separately lets you distinguish 'slow to start' from 'slow to generate.'"

### Design — SSE with chunk-by-chunk delivery

**SSE format:**
```
data: {"choices":[{"text":"Hello","index":0}]}

data: {"choices":[{"text":" world","index":0}]}

data: {"choices":[{"text":"!","index":0}]}

data: [DONE]

```

**Key details:**
- Each chunk is `data: <JSON>\n\n` (two newlines)
- Final chunk is `data: [DONE]\n\n`
- `Content-Type: text/event-stream`
- Flushing required after each chunk

### How it works — streaming lifecycle

**1. Client requests streaming**
```bash
curl -X POST http://localhost:8000/v1/completions \
  -H "Content-Type: application/json" \
  -d '{
    "model": "tinyllama",
    "prompt": "Hello, how are you?",
    "max_tokens": 100,
    "stream": true
  }'
```

**2. Handler detects streaming**
```go
if req.Stream {
    h.HandleCompletionsStreaming(w, r, &req)
    return
}
```

**3. Set SSE headers**
```go
w.Header().Set("Content-Type", "text/event-stream")
w.Header().Set("Cache-Control", "no-cache")
w.Header().Set("Connection", "keep-alive")

flusher, ok := w.(http.Flusher)
if !ok {
    http.Error(w, "Streaming not supported", 500)
    return
}
```

**4. Start streaming**
```go
chunkChan, errChan, backend, err := h.router.RouteCompletionStream(ctx, req)

ttftStart := time.Now()
var ttftRecorded bool
var tokenCount int

for {
    select {
    case chunk, ok := <-chunkChan:
        if !ok {
            // Stream ended
            goto Done
        }
        
        // Record TTFT on first token
        if !ttftRecorded {
            ttft := time.Since(ttftStart).Seconds()
            h.metrics.TimeToFirstTokenSeconds.Observe(ttft)
            ttftRecorded = true
        }
        
        tokenCount++
        
        // Write SSE chunk
        fmt.Fprintf(w, "data: %s\n\n", chunk.JSON())
        flusher.Flush()
        
    case err := <-errChan:
        if err != nil {
            h.writeSSEError(w, flusher, err.Error(), 500)
            return
        }
    }
}

Done:
// Record TPOT
if tokenCount > 1 {
    totalTime := time.Since(ttftStart).Seconds()
    tpot := totalTime / float64(tokenCount)
    h.metrics.TimePerOutputTokenSeconds.Observe(tpot)
}

// Send [DONE]
fmt.Fprintf(w, "data: [DONE]\n\n")
flusher.Flush()
```

**Key details:**
- TTFT measured from request start to first chunk
- TPOT calculated as `total_time / token_count`
- `flusher.Flush()` after every chunk (required for SSE)

### Engine client streaming

```go
func (c *Client) CreateCompletionStream(ctx context.Context, req *CompletionRequest) (<-chan StreamCompletionChunk, <-chan error) {
    chunkChan := make(chan StreamCompletionChunk)
    errChan := make(chan error, 1)
    
    go func() {
        defer close(chunkChan)
        defer close(errChan)
        
        // HTTP request with stream=true
        resp, err := c.httpClient.Post(c.url+"/v1/completions", ...)
        if err != nil {
            errChan <- err
            return
        }
        defer resp.Body.Close()
        
        // Parse SSE stream
        scanner := bufio.NewScanner(resp.Body)
        for scanner.Scan() {
            line := scanner.Text()
            
            if !strings.HasPrefix(line, "data: ") {
                continue
            }
            
            data := strings.TrimPrefix(line, "data: ")
            
            if data == "[DONE]" {
                return
            }
            
            var chunk StreamCompletionChunk
            if err := json.Unmarshal([]byte(data), &chunk); err != nil {
                errChan <- err
                return
            }
            
            chunkChan <- chunk
        }
    }()
    
    return chunkChan, errChan
}
```

**SSE parsing:**  
Read line-by-line, look for `data: ` prefix, parse JSON, send to channel. **This is streaming all the way through**—gateway receives chunks from engine and forwards them immediately to client.

### APIs & tools

| API | Why |
|---|---|
| `http.Flusher` | Force HTTP flush after each chunk (required for SSE) |
| `text/event-stream` | Standard content type for SSE |
| Channels for chunks | Idiomatic Go streaming |
| Histogram for TTFT/TPOT | Percentiles (p50, p95, p99) |

### Business benefit

- **Perceived latency** — First token in 50ms instead of waiting 2s for full response
- **User experience** — Feels responsive, like ChatGPT
- **Diagnostics** — TTFT vs TPOT tells you where bottlenecks are (queue? generation?)
- **Production-grade** — Streaming is how real LLM APIs work

### Metrics

```
gateway_time_to_first_token_seconds       histogram
gateway_time_per_output_token_seconds     histogram
gateway_streaming_connections             gauge
```

### Questions you'll be asked

**"Why stream instead of buffering?"**
> "Perceived latency. For a 100-token response at 10 tok/s, total time is 10 seconds either way. But with streaming, the user sees the first token in 50ms and watches the rest arrive, which feels responsive. With buffering, they wait 10 seconds and see nothing, which feels broken. This is why ChatGPT uses streaming—it's a better user experience."

**"How do you calculate TPOT?"**
> "Total generation time divided by token count. For example, 100 tokens in 5 seconds → TPOT = 0.05s = 50ms per token = 20 tok/s. This is different from TTFT, which measures time to the first token. TTFT includes queue wait and model load; TPOT measures pure generation speed."

**"What if the connection drops mid-stream?"**
> "The context cancels (`ctx.Done()` fires), the goroutine exits, and the semaphore slot is released immediately. We don't waste compute generating tokens for a disconnected client. This is the same context propagation pattern as non-streaming requests."

**"Can you cache streaming responses?"**
> "Not in the current implementation—caching requires buffering the full response before caching, which defeats the purpose of streaming (low TTFT). You could cache after the stream completes, but that's complex (need to replay the stream). For now, streaming requests bypass the cache, which is documented in 'what I'd do differently.'"

**"How do you measure p95 TTFT?"**
> "Prometheus histogram with buckets `[0.01, 0.05, 0.1, 0.5, 1, 2, 5]`. Query `histogram_quantile(0.95, rate(gateway_time_to_first_token_seconds_bucket[5m]))` gives p95 over the last 5 minutes. If p95 TTFT spikes, either the queue is backing up or the model is cold-starting."

---

## UPDATED PROJECT SUMMARY — Phase 1 Additions

### The transformation

**Before Phase 1:**
- Single-node gateway
- Request-based admission
- No streaming
- No fault tolerance
- 10 metrics

**After Phase 1:**
- **Distributed platform**
- **Token-aware admission** (LLM-specific)
- **SSE streaming** (TTFT/TPOT)
- **Multi-backend routing** (circuit breakers, health checks)
- **23 metrics**

### The new hook

> "I built a distributed LLM inference platform in Go with token-aware admission control, SSE streaming, and multi-backend routing. The system uses estimated token budgets for scheduling—a 4,000-token request consumes proportionally more of the 20,000-token capacity than a 50-token request, so scheduling reflects actual GPU work instead of just request count. I implemented circuit breakers that open after 3 consecutive backend failures, weighted routing that considers both current load and recent p95 latency, and comprehensive observability with 23 Prometheus metrics including TTFT and TPOT."

### The resume bullet

> **Designed and implemented distributed LLM inference platform in Go featuring token-aware admission control, SSE streaming, and multi-backend routing with health checks and circuit breakers. Built weighted least-loaded routing strategy considering backend load and p95 latency. Instrumented 23 Prometheus metrics including TTFT, TPOT, token budget utilization, and per-backend health.**

### The updated one-minute pitch

> "I built a distributed LLM inference platform in Go that routes requests across multiple GPU workers with token-aware admission control and SSE streaming.
>
> The core innovation is token-based scheduling—instead of treating all requests equally, I estimate the token cost (prompt + max_tokens) and consume capacity proportionally. A 4,000-token request takes 4,000 units of a 20,000-token budget; a 50-token request takes 50. This is LLM-specific optimization—most load balancers schedule by request count, which doesn't reflect actual GPU work.
>
> For fault tolerance, I implemented multi-backend routing with weighted least-loaded selection. The scoring function is `(load/capacity) + (p95_latency * 10)`, so if a backend slows down, its score increases and traffic shifts away automatically. Circuit breakers open after 3 consecutive failures and close after 3 successes, so recovery is automatic.
>
> For user experience, I added SSE token-by-token streaming with TTFT and TPOT metrics. TTFT measures perceived latency (time to first token); TPOT measures generation quality (time per token). The first token arrives in 50ms instead of users waiting 2 seconds for the full response—same total time, better perception.
>
> The system has 23 Prometheus metrics using histograms for latency (percentiles, not averages), per-backend health tracking, and token budget utilization. I tested it with mock backends and verified routing distribution, failover behavior, and streaming performance. All code builds successfully and is documented comprehensively."

**Time:** 58 seconds  
**Covers:** Token-aware scheduling, multi-backend routing, SSE streaming, fault tolerance, metrics  
**Hooks:** LLM-specific optimization, distributed systems patterns, production-grade serving

---

## NEW QUESTIONS FOR PHASE 1

**"Why did you add token-aware scheduling?"**
> "Because request-based admission is unfair for LLMs—a 50-token request and a 4,000-token request both consume '1 slot' but differ 80× in GPU time. Token-based admission schedules by actual work. This is capacity planning done right: '20,000 tokens capacity' is more meaningful than '8 request slots.' It's LLM-specific optimization that most HTTP load balancers don't have."

**"How does the circuit breaker work?"**
> "3-strike pattern. After 3 consecutive failures, the circuit opens and we stop routing to that backend. Health checks still run every 10s. After 3 consecutive successes, the circuit closes and traffic resumes. This prevents cascading failures—if one backend crashes, we detect it quickly and fail over to healthy ones."

**"What's the difference between TTFT and TPOT?"**
> "TTFT is time to first token—it measures perceived latency and includes queue wait, model load, and prompt processing. TPOT is time per output token—it measures generation speed. For example, TTFT = 500ms, TPOT = 50ms means the user sees the first token in 0.5s, then 1 token every 50ms (20 tok/s). Tracking both separately lets you diagnose whether the bottleneck is queue wait or generation."

**"How do you handle backend failures mid-request?"**
> "For non-streaming: If the engine call fails, the router's `RouteCompletion` returns an error, the backend's `RecordFailure()` increments the failure count (may open circuit), and the client gets a 500. For streaming: The error channel fires, we send an SSE error message to the client, and the backend still records the failure. The semaphore slot is released via `defer`, so capacity isn't leaked."

**"Why weighted least-loaded instead of consistent hashing?"**
> "Consistent hashing is great for cache locality (route same prefixes to same backend for KV cache reuse), but it ignores backend health and load. If one backend in the consistent hash is slow, requests hashed to it will be slow. Weighted least-loaded adapts—if a backend slows down, its score increases and traffic shifts away. You could combine both: use consistent hashing for prefix routing, but re-hash to a different backend if the first one is unhealthy."

---

## WHAT I'D DO DIFFERENTLY — Updated

### 1. Streaming + caching integration
**Current:** Streaming requests bypass the cache.  
**Better:** Buffer the stream in memory, cache after completion, replay for future requests.  
**Why:** Repeated streaming requests (e.g., "Hello" in a chatbot) still hit the engine every time.

### 2. Prefix-aware routing for KV cache locality
**Current:** Random selection among healthy backends.  
**Better:** Hash first N tokens of prompt, route to same backend for shared prefixes.  
**Why:** llama.cpp and vLLM cache KV states for common prefixes—routing "You are a helpful assistant..." to the same backend improves cache hit rates.

### 3. Gradual ramp-up after circuit closes
**Current:** Circuit closes → full traffic resumes immediately.  
**Better:** After closing, limit backend to 10% of traffic for 30s, then 50%, then 100%.  
**Why:** Prevents thundering herd (all queued requests rush to newly-healthy backend and overwhelm it again).

### 4. Real tokenizer for cost estimation
**Current:** ~4 chars/token heuristic.  
**Better:** Run `tiktoken` or `sentencepiece` on 1-10% of requests, calibrate the multiplier.  
**Why:** More accurate token estimates → better capacity planning.

### 5. Per-client rate limiting
**Current:** Global admission queue.  
**Better:** Token bucket per API key or IP.  
**Why:** Prevents one client from monopolizing the queue.

### 6. Priority queueing
**Current:** FIFO admission.  
**Better:** Two-tier queue (high/low priority).  
**Why:** Interactive requests (chatbot) should jump batch jobs (bulk embeddings).

### 7. OpenTelemetry tracing
**Current:** Prometheus metrics only.  
**Better:** Add distributed tracing (request ID flows through gateway → backend → back).  
**Why:** Diagnose "which backend is slow for this specific request" instead of aggregate metrics.

---

## FINAL UPDATED CHECKLIST

### Before the interview

✅ **Know the three Phase 1 features** — Token-aware scheduling, multi-backend routing, SSE streaming  
✅ **Know the routing score formula** — `(load/capacity) + (p95_latency * 10)`  
✅ **Know the circuit breaker logic** — 3 failures → open, 3 successes → close  
✅ **Know TTFT vs TPOT** — Perceived latency vs generation speed  
✅ **Know the 23 metrics** — List them by heart (10 original + 13 new)  
✅ **Know token estimation** — ~4 chars/token heuristic, why not real tokenizer  
✅ **Re-read PHASE1_COMPLETE.md** — Know the architecture diagrams  

### Practice saying

> "Token-based admission schedules by actual GPU work, not request count."

> "Circuit breakers prevent cascading failures—3 strikes open, 3 successes close."

> "TTFT measures perceived latency; TPOT measures generation quality."

> "Weighted routing is adaptive—if a backend slows down, traffic shifts away automatically."

> "SSE streaming shows the first token in 50ms instead of waiting 2 seconds."

### Red flags to avoid

🔴 Don't claim "I built the inference engine" — you built the platform layer  
🔴 Don't claim "zero latency overhead" — streaming and routing add overhead (measure it)  
🔴 Don't claim "perfect load balancing" — weighted least-loaded is heuristic, not optimal  
🔴 Don't claim "production-tested" — this is a personal project with mock backends  

---

**Phase 1 transformed this from "gateway" to "distributed platform." The technical depth is now at the level of production AI infrastructure. Use this guide to prepare for senior/staff-level systems interviews.**
