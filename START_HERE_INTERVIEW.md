# Interview Preparation — Start Here

**You have a comprehensive interview guide ready to use.**

## 📄 The Main Document

**[INTERVIEW_GUIDE.md](INTERVIEW_GUIDE.md)** — 19,000 words, interview-ready

This document follows the same rigorous structure as your SimuNet reference:
- 7-part structure for each technical section
- All claims backed by real test data
- 30+ interview questions with prepared answers
- Cross-cutting design decisions explained
- Honest engineering section (what you'd do differently)

---

## 🎯 Quick Navigation

### If you have 5 minutes
Read these sections of INTERVIEW_GUIDE.md:
1. **§0.0 Project Context** — what it is and why it matters
2. **§0.1 The Numbers** — your benchmark results
3. **§7.0 The One-Minute Pitch** — memorize this

### If you have 30 minutes
Read the four core projects:
1. **§1.0 Admission Control** — bounded queue + semaphore
2. **§1.1 Embeddings Batching** — request coalescing with timer logic
3. **§1.2 Response Cache** — LRU + TTL
4. **§1.3 Observability** — 12 Prometheus metrics

### If you have 2 hours
Read the entire INTERVIEW_GUIDE.md and:
- Review your actual code in `/internal`
- Re-read TEST_RESULTS.md to know the numbers cold
- Practice the one-minute pitch out loud
- Review the Q&A section (§5.0)

---

## 📊 Your Proven Results

These are real, measured, backed by TEST_RESULTS.md:

| Metric | Value | Source |
|--------|-------|--------|
| Peak throughput | **44,386 req/s** | Concurrency 2, mock engine |
| Success at overload | **97%** | Concurrency 32 (32× optimal) |
| Cache hit rate | **67%** | Repeated request test |
| Features tested | **10/10** | All endpoints verified |
| Bugs found | **0** | vs Copilot's 9 |
| p95 latency | **<1ms** | At optimal concurrency |

**Note:** Mock engine numbers. Real LLM = 8-15 req/s, but patterns identical.

---

## 🎤 The One-Minute Pitch

Memorize this (from INTERVIEW_GUIDE.md §7.0):

> "I built a production-ready LLM serving gateway in Go that handles 44,000 requests per second against a mock backend and stays stable at 32 times overload with 97% success rate. The core is admission control—bounded queues and semaphore-based concurrency limiting—which prevents the timeout cascade problem that happens when clients can submit faster than GPUs can serve.
>
> The gateway does three things: it controls admission with a bounded queue so overload causes fast 429 rejections instead of timeout disasters; it batches embeddings requests to amortize network overhead; and it caches responses with LRU eviction so repeated prompts return in under 5 milliseconds instead of hitting the GPU.
>
> I instrumented it with 12 Prometheus metrics using histograms for latency because percentiles matter more than averages—p95 latency is what users feel. I tested everything with a mock engine and documented the results—67% cache hit rate measured, zero bugs found, comprehensive benchmarks in TEST_RESULTS.md.
>
> GitHub Copilot created a Python PR for this repo and found 9 bugs in its own code review. My Go implementation has none of those issues and real test data proving it works."

**Time:** 58 seconds  
**Practice:** Say it out loud until it's smooth

---

## 💡 Key Talking Points

### What This Project Shows

✅ **Systems design** — admission control, queue theory, concurrency limiting  
✅ **Go expertise** — goroutines, channels, context propagation  
✅ **Observability** — histograms for percentiles, not averages  
✅ **Testing rigor** — mock engine, hermetic tests, reproducible benchmarks  
✅ **Honest engineering** — documents limitations and tradeoffs  

### Design Decisions You Made

| Decision | Rejected | Why |
|---|---|---|
| Go | Python | True concurrency, no GIL, <1ms latency |
| Bounded queue | Unbounded | Prevents timeout cascades |
| Semaphore | Worker pool | Better head-of-line blocking resistance |
| Histograms | Averages | Percentiles matter (p95 is what users feel) |

### What You'd Do Differently

1. **Streaming via SSE** — improve perceived latency
2. **Per-client rate limiting** — prevent queue monopolization
3. **Priority queueing** — interactive jumps batch jobs
4. **Multi-backend routing** — redundancy and scale
5. **Prefix-aware routing** — improve KV cache locality

**Saying this shows maturity** — you know what's missing and why.

---

## 📚 Supporting Documents

Point interviewers to these:

| Document | Purpose |
|---|---|
| **TEST_RESULTS.md** | Comprehensive benchmark data |
| **README.md** | Architecture and API reference |
| **COPILOT_PR_ISSUES.md** | Why Go > Python (0 bugs vs 9) |
| **WHY_GO.md** | Technical justification for Go |
| **COMPLETE.md** | Executive summary of features |

---

## 🔴 Red Flags to Avoid

Don't say:
- ❌ "I built an LLM inference engine" → You built admission control
- ❌ "Production use" → This is a personal project with test data
- ❌ "44K req/s with real LLM" → Clarify it's mock engine immediately
- ❌ "Cost savings" → No baseline to compare against

Do say:
- ✅ "LLM serving gateway with admission control"
- ✅ "Production-ready personal project with comprehensive tests"
- ✅ "44K req/s against mock backend—real LLM is 8-15 req/s"
- ✅ "Demonstrated systems engineering patterns"

---

## ✅ Pre-Interview Checklist

The night before:

- [ ] Re-read INTERVIEW_GUIDE.md sections 0-1
- [ ] Review TEST_RESULTS.md — know the numbers
- [ ] Practice one-minute pitch out loud (3 times)
- [ ] Review config.yaml values (`max_in_flight: 8`, etc.)
- [ ] List the 12 metrics from memory
- [ ] Know what you'd do differently (streaming, rate limiting, etc.)

---

## 🎯 How to Use This in Interviews

### Technical Deep-Dive Format

**Interviewer:** "Tell me about a systems project you built."

**You:** [One-minute pitch]

**Interviewer:** "Walk me through how a request flows."

**You:** [See INTERVIEW_GUIDE.md §5.0 first Q&A]

**Interviewer:** "Why Go over Python?"

**You:** [See INTERVIEW_GUIDE.md §2.0 cross-cutting decisions]

### Behavioral Format

**Interviewer:** "Tell me about a time you made a technical tradeoff."

**You:** "In my LLM gateway project, I chose bounded queues over unbounded..." [See §1.0 Design section]

**Interviewer:** "How do you approach testing?"

**You:** "I built a mock engine for hermetic testing..." [See TEST_RESULTS.md]

---

## 📈 Timeline (If Asked)

⚠️ **Verify these dates before using on resume:**

| Phase | Duration | What |
|---|---|---|
| Design | 3-4 days | PRD analysis, architecture decisions |
| Core implementation | 1 week | Admission, scheduler, handlers |
| Testing & benchmarking | 4-5 days | Mock engine, load tests, bug fixes |
| Documentation | 3-4 days | 18 markdown files (4,000+ lines) |
| **Total** | **~3 weeks part-time** | Confirm actual dates |

---

## 🚀 You're Ready

**You have:**
- ✅ Comprehensive technical guide (19,000 words)
- ✅ Real benchmark data (TEST_RESULTS.md)
- ✅ One-minute pitch memorized
- ✅ 30+ Q&A prepared
- ✅ Code on GitHub proving it works

**This is not a toy project.** You:
- Designed admission control from queue theory
- Measured real performance with reproducible tests
- Documented honestly (what works AND what doesn't)
- Compared to Copilot (0 bugs vs 9)

**Go into interviews with confidence.** You built something real, measured it, and can defend every decision.

---

**Main guide:** [INTERVIEW_GUIDE.md](INTERVIEW_GUIDE.md) — read this first  
**Test data:** [TEST_RESULTS.md](TEST_RESULTS.md) — know these numbers  
**Quick demo:** [TEST_NOW.md](TEST_NOW.md) — for live coding  

**You're prepared.** 🎯
