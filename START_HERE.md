# Inference Gateway — Start Here

**Distributed LLM inference platform in Go with token-aware scheduling, SSE streaming, and multi-backend routing.**

---

## 🎯 What is This?

A Go-based **distributed platform** that sits in front of multiple LLM inference engines (vLLM, llama.cpp) and provides:

### Phase 1 Features (Distributed Platform) ✨
- **Token-aware scheduling** — Schedule by GPU work (tokens), not request count
- **Multi-backend routing** — Weighted load balancing with health checks
- **Circuit breakers** — 3-strike open/close for fault tolerance
- **SSE streaming** — Token-by-token with TTFT/TPOT metrics
- **23 Prometheus metrics** — Comprehensive observability

### Core Features (MVP)
- **Admission control** — Bounded queues prevent timeout cascades
- **Concurrency limiting** — Semaphore-based scheduling
- **Request batching** — Embeddings coalescing at gateway layer
- **Response caching** — LRU + TTL for repeated requests
- **Cost tracking** — Token accounting and $/M tokens

**Benchmark highlights:** 44K req/s peak, 97% success at 32× overload, 67% cache hit rate

---

## 📚 Documentation Structure

### 🎤 **For Interviews & Portfolio**
Start here if showing this to recruiters or preparing for interviews:

- **[docs/interview/PROJECT_SUMMARY.md](docs/interview/PROJECT_SUMMARY.md)** — Complete project overview (one file, everything)
- **[docs/interview/INTERVIEW_GUIDE.md](docs/interview/INTERVIEW_GUIDE.md)** — 1,100+ lines, 30+ Q&A, comprehensive MVP prep
- **[docs/interview/PHASE1_INTERVIEW_ADDITIONS.md](docs/interview/PHASE1_INTERVIEW_ADDITIONS.md)** ✨ — Phase 1 features (token scheduling, routing, streaming)
- **[docs/interview/START_HERE_INTERVIEW.md](docs/interview/START_HERE_INTERVIEW.md)** — Quick prep guide (5min/30min/2hr paths)

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
1. Read [docs/PHASE1_COMPLETE.md](docs/PHASE1_COMPLETE.md) — Phase 1 summary
2. Read [docs/interview/PROJECT_SUMMARY.md](docs/interview/PROJECT_SUMMARY.md) — Full overview
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
1. Read [docs/interview/START_HERE_INTERVIEW.md](docs/interview/START_HERE_INTERVIEW.md)
2. Review [docs/interview/INTERVIEW_GUIDE.md](docs/interview/INTERVIEW_GUIDE.md) — MVP features
3. Review [docs/interview/PHASE1_INTERVIEW_ADDITIONS.md](docs/interview/PHASE1_INTERVIEW_ADDITIONS.md) — Distributed features
4. Practice one-minute pitch (updated for Phase 1)

### "I want to see the test data" (15 minutes)
1. Read [docs/TEST_RESULTS.md](docs/TEST_RESULTS.md) — MVP benchmarks
2. Read [docs/PHASE1_INTEGRATION_COMPLETE.md](docs/PHASE1_INTEGRATION_COMPLETE.md) — Phase 1 status
3. Check [benchmark_results.txt](benchmark_results.txt)

### "I want to extend this project" (30 minutes)
1. Read [docs/development/ROADMAP.md](docs/development/ROADMAP.md)
2. Review [docs/development/NEXT_STEPS.md](docs/development/NEXT_STEPS.md)

---

## 🎯 Key Results (All Measured)

### MVP Benchmarks
| Metric | Value |
|--------|-------|
| Peak throughput | **44,386 req/s** |
| Success at overload | **97%** (32× optimal) |
| Cache hit rate | **67%** |
| Features tested | **10/10** |
| Bugs found | **0** (vs Copilot's 9) |
| p95 latency | **<1ms** |

### Phase 1 Additions ✨
| Metric | Value |
|--------|-------|
| New features | **3** (streaming, routing, token scheduling) |
| New metrics | **13** (23 total) |
| New code | **~1,336 lines** |
| Build status | **✅ All packages compile** |
| Documentation | **+1,378 lines** |

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

## 🎤 The Updated One-Minute Pitch

> "I built a **distributed LLM inference platform** in Go with token-aware admission control, SSE streaming, and multi-backend routing. The system uses estimated token budgets for scheduling—a 4,000-token request consumes proportionally more of the 20,000-token capacity than a 50-token request, so scheduling reflects actual GPU work instead of just request count.
>
> I implemented circuit breakers that open after 3 consecutive backend failures, weighted routing that considers both current load and recent p95 latency, and **comprehensive observability** with **23 Prometheus metrics** including **TTFT (time to first token)** and **TPOT (time per output token)**.
>
> The core MVP handles 44,000 req/s with 97% success at 32× overload. Phase 1 adds **distributed systems patterns** (fault tolerance, health checks, load balancing) and **LLM-specific optimization** (token-aware scheduling). GitHub Copilot created a Python PR and found 9 bugs in its own review—my Go implementation has zero."

---

## ✅ Project Status

**MVP:** ✅ Complete, tested, documented  
**Phase 1:** ✅ Implemented, integrated, builds successfully  
**Testing:** ⏳ Ready for multi-backend testing  
**Documentation:** ✅ 25 files, 8,099+ lines  
**Interview Ready:** ✅ Comprehensive prep for distributed systems & LLM optimization  

---

**Quick start:**
1. **For interviews:** [docs/interview/PHASE1_INTERVIEW_ADDITIONS.md](docs/interview/PHASE1_INTERVIEW_ADDITIONS.md)
2. **For overview:** [docs/PHASE1_COMPLETE.md](docs/PHASE1_COMPLETE.md)
3. **For MVP details:** [docs/interview/PROJECT_SUMMARY.md](docs/interview/PROJECT_SUMMARY.md)
