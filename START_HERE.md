# Inference Gateway — Start Here

**Production-ready LLM serving layer in Go with admission control, batching, caching, and observability.**

---

## 🎯 What is This?

A Go-based gateway that sits in front of LLM inference engines (vLLM, llama.cpp) and provides:
- **Admission control** — Bounded queues prevent timeout cascades
- **Concurrency limiting** — Semaphore-based scheduling
- **Request batching** — Embeddings coalescing at gateway layer
- **Response caching** — LRU + TTL for repeated requests
- **Observability** — 12 Prometheus metrics with histograms
- **Cost tracking** — Token accounting and $/M tokens

**Benchmark highlights:** 44K req/s peak, 97% success at 32× overload, 67% cache hit rate

---

## 📚 Documentation Structure

### 🎤 **For Interviews & Portfolio**
Start here if showing this to recruiters or preparing for interviews:

- **[docs/interview/PROJECT_SUMMARY.md](docs/interview/PROJECT_SUMMARY.md)** — Complete project overview (one file, everything)
- **[docs/interview/INTERVIEW_GUIDE.md](docs/interview/INTERVIEW_GUIDE.md)** — 1,100+ lines, 30+ Q&A, comprehensive prep
- **[docs/interview/START_HERE_INTERVIEW.md](docs/interview/START_HERE_INTERVIEW.md)** — Quick prep guide (5min/30min/2hr paths)

### 🚀 **For Using/Testing**
Start here if you want to run the gateway:

- **[README.md](README.md)** — Architecture, API, configuration
- **[docs/development/QUICKSTART.md](docs/development/QUICKSTART.md)** — 5-minute getting started
- **[docs/development/TEST_NOW.md](docs/development/TEST_NOW.md)** — Immediate testing with mock engine
- **[docs/development/CONTRIBUTING.md](docs/development/CONTRIBUTING.md)** — Development guidelines

### 📊 **Test Results & Technical Details**
For the engineering deep-dive:

- **[docs/TEST_RESULTS.md](docs/TEST_RESULTS.md)** — Complete benchmark data
- **[docs/WHY_GO.md](docs/WHY_GO.md)** — Technical justification (Go vs Python)
- **[docs/COMPLETE.md](docs/COMPLETE.md)** — Project completion summary
- **[docs/SUMMARY.md](docs/SUMMARY.md)** — One-page overview
- **[docs/STATUS.md](docs/STATUS.md)** — Current state

### 🗺️ **Future Enhancements**
If you want to extend the project:

- **[docs/development/ROADMAP.md](docs/development/ROADMAP.md)** — Phase 1-3 enhancement plan (716 lines)
- **[docs/development/NEXT_STEPS.md](docs/development/NEXT_STEPS.md)** — Decision guide based on expert feedback

### 📁 **GitHub Meta**
Background on the Copilot PR situation:

- **[docs/github-meta/COPILOT_PR_ISSUES.md](docs/github-meta/COPILOT_PR_ISSUES.md)** — 0 bugs vs Copilot's 9
- **[docs/github-meta/REVERT_EXPLANATION.md](docs/github-meta/REVERT_EXPLANATION.md)** — Why Python PR was reverted

---

## ⚡ Quick Links by Use Case

### "I want to understand what you built" (5 minutes)
1. Read [docs/interview/PROJECT_SUMMARY.md](docs/interview/PROJECT_SUMMARY.md)
2. Look at [README.md](README.md) — Architecture section

### "I want to run it right now" (10 minutes)
1. Read [docs/development/TEST_NOW.md](docs/development/TEST_NOW.md)
2. Run:
```bash
python3 scripts/mock_engine.py  # Terminal 1
./bin/gateway                    # Terminal 2
./test.sh                        # Terminal 3
```

### "I'm preparing for an interview" (30-120 minutes)
1. Read [docs/interview/START_HERE_INTERVIEW.md](docs/interview/START_HERE_INTERVIEW.md)
2. Review [docs/interview/INTERVIEW_GUIDE.md](docs/interview/INTERVIEW_GUIDE.md)
3. Practice one-minute pitch

### "I want to see the test data" (15 minutes)
1. Read [docs/TEST_RESULTS.md](docs/TEST_RESULTS.md)
2. Check [benchmark_results.txt](benchmark_results.txt)

### "I want to extend this project" (30 minutes)
1. Read [docs/development/ROADMAP.md](docs/development/ROADMAP.md)
2. Review [docs/development/NEXT_STEPS.md](docs/development/NEXT_STEPS.md)

---

## 🎯 Key Results (All Measured)

| Metric | Value |
|--------|-------|
| Peak throughput | **44,386 req/s** |
| Success at overload | **97%** (32× optimal) |
| Cache hit rate | **67%** |
| Features tested | **10/10** |
| Bugs found | **0** (vs Copilot's 9) |
| p95 latency | **<1ms** |
| Documentation | **22 files, 6,721 lines** |

---

## 🏗️ Architecture Overview

```
Client → Handler → Cache → Admission Queue → Scheduler → Engine
              ↓              ↓                    ↓
           Miss          429 Reject          HTTP Call
```

**Key decisions:**
- Go over Python (no GIL, true concurrency)
- Semaphore over worker pool (dynamic slots)
- Bounded queue (prevents timeout cascades)
- Histograms (p95 matters, not averages)

---

## 📦 Repository Structure

```
.
├── README.md              ⭐ Technical overview
├── START_HERE.md          ⭐ This file
│
├── docs/
│   ├── interview/         📚 Interview prep
│   ├── development/       🔧 Development & testing
│   ├── github-meta/       📁 GitHub context
│   └── *.md               📊 Technical reports
│
├── cmd/                   Go applications
├── internal/              Core packages
├── scripts/               Utilities
├── bin/                   Compiled binaries
└── config.yaml           Configuration
```

---

## 🎤 The One-Minute Pitch

> "I built a production-ready LLM serving gateway in Go that handles 44,000 req/s and stays stable at 32× overload with 97% success. The core is admission control—bounded queues and semaphore scheduling prevent timeout cascades. It batches embeddings, caches responses (67% hit rate), and exports 12 Prometheus metrics with histograms because p95 latency is what users feel. GitHub Copilot created a Python PR and found 9 bugs in its own review—my Go implementation has zero."

---

## ✅ Project Status

**Code:** ✅ Complete, tested  
**Docs:** ✅ 22 files, 6,721 lines  
**Interview Ready:** ✅ Fully prepared  

---

**Start with:** [docs/interview/PROJECT_SUMMARY.md](docs/interview/PROJECT_SUMMARY.md) for complete overview
