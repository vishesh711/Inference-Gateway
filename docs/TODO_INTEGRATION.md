# Phase 1 Integration TODO

**Status:** Core features implemented, need final wiring  
**Remaining work:** 2-4 hours

---

## What's Done ✅

1. **SSE Streaming** — Complete, builds successfully
2. **Backend Registry** — Complete, builds successfully  
3. **Router** — Complete, builds successfully
4. **Token Scheduler** — Complete, builds successfully
5. **Metrics** — All 13 new metrics defined
6. **Config** — Updated for backends, tokens, health

---

## What Needs Wiring 🔧

### 1. Update cmd/gateway/main.go

**Current state:** Partially updated, compilation errors

**Needs:**
```go
// Instead of single engine client:
engineClient := engine.NewClient(...)

// Use router:
router := backend.NewRouter(registry)
for _, bcfg := range cfg.Backends {
    client := engine.NewClient(bcfg.URL, ...)
    router.RegisterClient(bcfg.ID, client)
}
```

**Handler creation:**
```go
// Pass router instead of engine client
h := handler.New(
    router,          // <-- Change this
    admission,
    hybridScheduler, // <-- Use token-aware scheduler
    cache,
    metrics,
    accountant,
    timeout,
)
```

### 2. Update Handler Interface

**Current:** Handler expects `*engine.Client`

**Change to:** Handler expects `*backend.Router`

**Files to modify:**
- `internal/handler/handler.go` — Change struct field
- `internal/handler/handler.go` — Use `router.RouteCompletion()` instead of `engine.CreateCompletion()`
- `internal/handler/streaming.go` — Use `router.RouteCompletionStream()`

### 3. Token Budget Integration

**In handlers:**
```go
// Estimate token cost
promptText := req.Prompt // or concatenate messages
tokenCost := hybridScheduler.tokenBudget.EstimateCompletionCost(promptText, req.MaxTokens)

// Acquire with cost
requestID := generateRequestID()
if err := hybridScheduler.AcquireWithCost(ctx, requestID, tokenCost); err != nil {
    // Handle
}
defer hybridScheduler.ReleaseWithCost(requestID)

// Record metric
metrics.EstimatedTokensPerRequest.Observe(float64(tokenCost))
```

### 4. Metrics Reporting Loop

**Add to main.go:**
```go
// Update token budget metrics
if cfg.Scheduler.UseTokenScheduling {
    go func() {
        ticker := time.NewTicker(5 * time.Second)
        defer ticker.Stop()
        for range ticker.C {
            metricsInstance.TokenBudgetCapacity.Set(float64(hybridScheduler.tokenBudget.Capacity()))
            metricsInstance.TokenBudgetInUse.Set(float64(hybridScheduler.tokenBudget.InUse()))
            metricsInstance.TokenBudgetUtilization.Set(hybridScheduler.tokenBudget.Utilization())
        }
    }()
}
```

### 5. Backend Metrics Loop

**Already partially added, verify it works**

### 6. Embeddings Batcher

**Current:** Uses single engine client

**Change:** Should use router for multi-backend

**Option 1 (Simple):** Keep batcher using primary backend  
**Option 2 (Better):** Route embedding batches through router

---

## Quick Integration Path

### Step 1: Simplify main.go (30 min)
1. Keep backward compatibility — if no backends array, use legacy single engine
2. If backends array exists, use router
3. Use hybrid scheduler with token mode configurable

### Step 2: Update Handler (20 min)
1. Add Router field to Handler struct
2. Add helper method: `routeOrDirect()` — uses router if available, else direct engine
3. Minimal changes to existing handler logic

### Step 3: Test Basic Functionality (30 min)
1. Single backend config — verify still works
2. Multi-backend config — verify routing works
3. Token scheduling off — verify request-based works
4. Token scheduling on — verify token-based works

### Step 4: Test Streaming (20 min)
1. Non-streaming request — works
2. Streaming request with `stream=true` — works
3. Verify TTFT and TPOT metrics appear

### Step 5: Test Multi-Backend (30 min)
1. Start 3 mock engines on ports 8080, 8081, 8082
2. Configure 3 backends
3. Send requests, verify distribution
4. Kill one backend, verify circuit opens
5. Restart backend, verify circuit closes

### Step 6: Documentation (1 hour)
1. Update README with Phase 1 features
2. Create STREAMING.md
3. Create MULTI_BACKEND.md  
4. Update INTERVIEW_GUIDE.md

---

## Testing Checklist

### Basic Functionality
- [ ] Gateway starts successfully
- [ ] Single backend mode works (backward compat)
- [ ] Multi-backend mode works
- [ ] Health checks run every 10s
- [ ] Metrics endpoint shows all 23 metrics

### Streaming
- [ ] Non-streaming completions work
- [ ] Streaming completions with `stream=true` work
- [ ] TTFT metric recorded
- [ ] TPOT metric recorded
- [ ] Streaming chat completions work

### Routing
- [ ] Requests distribute across healthy backends
- [ ] Unhealthy backend not selected
- [ ] Circuit breaker opens after 3 failures
- [ ] Circuit breaker closes after 3 successes
- [ ] Backend metrics update every 5s

### Token Scheduling
- [ ] Token budget metrics appear
- [ ] Large request consumes more tokens
- [ ] Token budget prevents overload at capacity
- [ ] Request-based mode still works (backward compat)

### Failure Scenarios
- [ ] One backend dies → circuit opens
- [ ] All backends die → return 503
- [ ] Backend comes back → circuit closes
- [ ] Mid-stream failure → proper error handling

---

## Alternative: Simplified Integration

**If full integration is complex, do this:**

### Minimal Viable Integration
1. Keep existing handler working with single engine (backward compat)
2. Add separate endpoints for Phase 1 features:
   - `/v2/completions` — Uses router and token budget
   - `/v2/chat/completions` — Uses router and token budget
3. This way:
   - Old code still works
   - New features testable
   - Can document both versions
   - Interview talking points work

### Benefits
- Less risk of breaking existing functionality
- Can test Phase 1 features independently
- Easy rollback if needed
- Good story: "I built v2 with enhanced features while keeping v1 stable"

---

## Estimated Time

**Full Integration:** 2-4 hours  
**Simplified Integration:** 1-2 hours  
**Documentation:** 1-2 hours  

**Total:** 3-6 hours to complete Phase 1

---

## Decision Point

**Option A: Full Integration**
- Wire router into existing handlers
- Replace all engine client references
- Update all 3 endpoints
- More complex, higher risk
- One unified system

**Option B: Simplified (v1/v2)**
- Keep existing v1 endpoints
- Add new v2 endpoints with Phase 1 features
- Lower risk, faster completion
- Can demo both versions
- Better interview story ("iterative enhancement")

**Recommendation:** Start with Option B for speed, can merge later

---

## Next Immediate Steps

1. **Decide:** Full integration vs v1/v2 approach
2. **Wire:** Connect pieces in main.go
3. **Test:** Basic smoke test with mock backend
4. **Document:** Update key files
5. **Commit:** Push Phase 1 to GitHub

**You're 90% done. Just needs final wiring!** 🔌
