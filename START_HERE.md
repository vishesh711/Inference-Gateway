# Inference Gateway — Start Here

**LLM serving gateway in Go with token-aware admission control.**

---

## 🎯 What is This?

An LLM serving gateway featuring **token-aware scheduling** (the key LLM-specific insight): requests consume scheduler budget proportional to estimated token cost rather than counting as one slot each, because a 50-token and a 2000-token completion tie up the engine for very different durations.

Also includes:
- **Admission control** — Bounded queue prevents timeout cascades
- **Request batching** — Embeddings coalescing with correct timer logic
- **Multi-backend routing** — Weighted selection with circuit breakers
- **SSE streaming** — TTFT and TPOT metrics
- **23 Prometheus metrics** — Histograms, not averages

**Status:** ✅ Functional tests pass, ⏳ performance benchmarks pending (see BENCHMARK_STATUS.md)

---

## 📚 Documentation Structure

### 🎤 **For Interviews & Portfolio**
Start here if showing this to recruiters or preparing for interviews:

- **[docs/interview/INTERVIEW_GUIDE_CORRECTED.md](docs/interview/INTERVIEW_GUIDE_CORRECTED.md)** ⭐ — **Use this one** (corrected, no invalid claims)
- **[docs/BENCHMARK_STATUS.md](docs/BENCHMARK_STATUS.md)** — Current testing status and real benchmark plan
- **[docs/interview/PHASE1_INTERVIEW_ADDITIONS.md](docs/interview/PHASE1_INTERVIEW_ADDITIONS.md)** — Phase 1 deep-dive
- ~~[docs/interview/INTERVIEW_GUIDE.md](docs/interview/INTERVIEW_GUIDE.md)~~ — Being updated, use CORRECTED version

### 🚀 **For Using/Testing**
Start here if you want to run the gateway:

- **[README.md](README.md)** — Architecture, API, configuration
- **[docs/development/QUICKSTART.md](docs/development/QUICKSTART.md)** — 5-minute getting started
- **[docs/development/TEST_NOW.md](docs/development/TEST_NOW.md)** — Immediate testing with mock engine
- **[docs/development/CONTRIBUTING.md](docs/development/CONTRIBUTING.md)** — Development guidelines

### 📊 **Test Results & Technical Details**
For the engineering deep-dive:

- **[docs/TEST_RESULTS.md](docs/TEST_RESULTS.md)** — Complete benchmark data (MVP)
- **[docs/PHASE1_COMPLETE.md](docs/PHASE1_COMPLETE.md)** ✨ — Phase 1 implementation summary
- **[docs/PHASE1_INTEGRATION_COMPLETE.md](docs/PHASE1_INTEGRATION_COMPLETE.md)** ✨ — Integration status
- **[docs/WHY_GO.md](docs/WHY_GO.md)** — Technical justification (Go vs Python)
- **[docs/COMPLETE.md](docs/COMPLETE.md)** — Project completion summary
- **[docs/SUMMARY.md](docs/SUMMARY.md)** — One-page overview
- **[docs/STATUS.md](docs/STATUS.md)** — Current state

### 🗺️ **Future Enhancements**
If you want to extend the project:

- **[docs/development/ROADMAP.md](docs/development/ROADMAP.md)** — Phase 1-3 enhancement plan (716 lines)
- **[docs/TODO_INTEGRATION.md](docs/TODO_INTEGRATION.md)** ✨ — Phase 1 testing guide
- **[docs/development/NEXT_STEPS.md](docs/development/NEXT_STEPS.md)** — Decision guide based on expert feedback

### 📁 **GitHub Meta**
Background on the Copilot PR situation:

- **[docs/github-meta/COPILOT_PR_ISSUES.md](docs/github-meta/COPILOT_PR_ISSUES.md)** — 0 bugs vs Copilot's 9
- **[docs/github-meta/REVERT_EXPLANATION.md](docs/github-meta/REVERT_EXPLANATION.md)** — Why Python PR was reverted

✨ = **Phase 1 additions** (distributed platform features)

---

## ⚡ Quick Links by Use Case

### "I want to understand what you built" (5 minutes)
1. Read [docs/BENCHMARK_STATUS.md](docs/BENCHMARK_STATUS.md) — Current status
2. Read [docs/interview/INTERVIEW_GUIDE_CORRECTED.md](docs/interview/INTERVIEW_GUIDE_CORRECTED.md) — One-minute pitch + bullets
3. Look at [README.md](README.md) — Architecture section

### "I want to run it right now" (10 minutes)
1. Read [docs/development/TEST_NOW.md](docs/development/TEST_NOW.md)
2. Read [docs/TODO_INTEGRATION.md](docs/TODO_INTEGRATION.md) — Multi-backend testing
3. Run:
```bash
python3 scripts/mock_engine.py --port 8080  # Terminal 1
python3 scripts/mock_engine.py --port 8081  # Terminal 2
./bin/gateway                                # Terminal 3
curl http://localhost:8000/v1/completions    # Terminal 4
```

### "I'm preparing for an interview" (30-120 minutes)
1. Read [docs/interview/INTERVIEW_GUIDE_CORRECTED.md](docs/interview/INTERVIEW_GUIDE_CORRECTED.md)
2. Review [docs/BENCHMARK_STATUS.md](docs/BENCHMARK_STATUS.md)
3. Practice the one-minute pitch (lead with token-aware scheduling)

### "I want to see the test data" (15 minutes)
1. Read [docs/BENCHMARK_STATUS.md](docs/BENCHMARK_STATUS.md) — Why mock benchmarks are invalid
2. Read [docs/FIXES_NEEDED.md](docs/FIXES_NEEDED.md) — What's being corrected
3. ~~[docs/TEST_RESULTS.md](docs/TEST_RESULTS.md)~~ — Mock results, being superseded

### "I want to extend this project" (30 minutes)
1. Read [docs/development/ROADMAP.md](docs/development/ROADMAP.md)
2. Review [docs/development/NEXT_STEPS.md](docs/development/NEXT_STEPS.md)

---

## 🎯 What's Been Validated

### ✅ Functional Tests (All Pass)
- Admission control: Queue full → 429 rejection
- Cache: Hit and miss paths work correctly
- Context cancellation: Client disconnect frees slot
- Graceful shutdown: In-flight work completes
- Metrics: All 23 metrics export correctly
- Streaming: SSE format and TTFT/TPOT tracking
- Multi-backend: Router selects backends, circuit breakers open/close
- Token scheduling: Budget acquire/release works

### ⏳ Performance Benchmarks (Pending)
- Mock benchmarks invalid (see [BENCHMARK_STATUS.md](docs/BENCHMARK_STATUS.md))
- Need real llama.cpp benchmarks (~4 hours work)
- Will measure concurrency knee
- Will measure embeddings batching improvement

**No performance numbers on resume until real benchmarks complete.**

---

## 🏗️ Architecture Overview

### Before Phase 1 (MVP)
```
Client → Handler → Cache → Admission Queue → Scheduler → Single Engine
```

### After Phase 1 (Distributed Platform) ✨
```
Client → Handler → Cache → Admission Queue
                                ↓
                        Token Budget (20K tokens)
                                ↓
                        Router (weighted routing)
                                ↓
                    ┌───────────┴───────────┐
                    ↓           ↓           ↓
               Backend 1   Backend 2   Backend 3
               (healthy)   (healthy)   (circuit open)
                    ↑           ↑           ↑
               Health Check (10s interval)
```

**Key decisions:**
- Go over Python (no GIL, true concurrency)
- Token-based over request-based (LLM-specific)
- Weighted routing over round-robin (adaptive)
- SSE streaming over buffering (perceived latency)
- Circuit breakers (3-strike fault tolerance)
- Semaphore over worker pool (dynamic slots)
- Bounded queue (prevents timeout cascades)
- Histograms (p95 matters, not averages)

---

## 📦 Repository Structure

```
.
├── README.md              ⭐ Technical overview
├── START_HERE.md          ⭐ This file
├── config.yaml            ⚙️ Multi-backend configuration
│
├── docs/
│   ├── interview/         📚 Interview prep (MVP + Phase 1)
│   ├── development/       🔧 Development & testing
│   ├── github-meta/       📁 GitHub context
│   ├── PHASE1_COMPLETE.md ✨ Phase 1 summary
│   └── *.md               📊 Technical reports
│
├── cmd/gateway/           Go application
├── internal/
│   ├── handler/           HTTP endpoints + streaming ✨
│   ├── backend/           Registry + Router ✨
│   ├── scheduler/         Token budget + HybridScheduler ✨
│   ├── engine/            Engine client
│   ├── cache/             Response cache
│   └── metrics/           23 Prometheus metrics ✨
│
├── scripts/               Mock engine, test utilities
└── bin/                   Compiled binaries
```

---

## 🎤 The One-Minute Pitch

> "I built an LLM serving gateway in Go. The core problem is that clients submit faster than a GPU can serve, and the naive fix—an unbounded queue—turns a capacity problem into a latency disaster where everything times out while the server keeps working.
>
> So admission is bounded and rejects with 429 immediately. But the piece I'd point at is **token-aware scheduling**: requests consume budget proportional to estimated token cost rather than counting as one slot each, because a 50-token and a 2000-token completion tie up the engine for very different lengths of time.
>
> I also implemented request coalescing for embeddings, where the batch timer starts on the first arrival rather than every arrival, since resetting per-arrival means a steady trickle never dispatches.
>
> Everything is instrumented with histograms rather than averages, separating queue wait from generation time so you can tell a backlog from slow inference. I've verified the control paths functionally. I still need to benchmark against a real engine, since my current numbers came from a mock responding in microseconds and aren't meaningful."

---

## ✅ Project Status

**Code:** ✅ Complete, builds successfully  
**Functional Tests:** ✅ All verified  
**Performance Tests:** ⏳ Pending real benchmarks (see BENCHMARK_STATUS.md)  
**Documentation:** ✅ Corrected, honest about limitations  
**Interview Ready:** ✅ Lead with token-aware scheduling + design reasoning  

---

**Quick start:**
1. **For interviews:** [docs/interview/INTERVIEW_GUIDE_CORRECTED.md](docs/interview/INTERVIEW_GUIDE_CORRECTED.md)
2. **For status:** [docs/BENCHMARK_STATUS.md](docs/BENCHMARK_STATUS.md)
3. **For testing plan:** [docs/FIXES_NEEDED.md](docs/FIXES_NEEDED.md)
