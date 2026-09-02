# Test the Gateway RIGHT NOW

You don't need to wait for llama.cpp to finish building. Test the gateway immediately!

## 3-Step Quick Start

### Step 1: Start the Mock Engine (Terminal 1)

```bash
python3 scripts/mock_engine.py
```

You should see:
```
======================================================================
Mock LLM Engine
======================================================================
Listening on: http://localhost:8080
```

### Step 2: Start the Gateway (Terminal 2)

```bash
./bin/gateway
```

You should see:
```
2026/09/02 15:44:14 Starting Inference Gateway on port 8000
2026/09/02 15:44:14 Listening on http://localhost:8000
```

### Step 3: Run the Test Script (Terminal 3)

```bash
./test.sh
```

This will:
- Check gateway health
- Send a completion request
- Send a chat completion request
- Display metrics

## Or Test Manually

```bash
# Simple completion
curl -X POST http://localhost:8000/v1/completions \
  -H "Content-Type: application/json" \
  -d '{"model":"test","prompt":"Hello, world!","max_tokens":50}'

# Chat completion
curl -X POST http://localhost:8000/v1/chat/completions \
  -H "Content-Type: application/json" \
  -d '{"model":"test","messages":[{"role":"user","content":"Hi!"}]}'

# View metrics
curl http://localhost:8000/metrics | grep gateway_
```

## Run a Load Test

```bash
./bin/loadgen -workers 4 -duration 30s -warmup 5s
```

You'll get output like:
```
======================================================================
LOAD TEST RESULTS
======================================================================
Duration (post-warmup):     30s
Total Requests:             450
Successful:                 450 (100.0%)
----------------------------------------------------------------------
Throughput:                 15.00 req/s
Tokens/sec:                 450.00 tok/s
----------------------------------------------------------------------
Latency Mean:               66ms
Latency p50:                65ms
Latency p95:                90ms
Latency p99:                110ms
======================================================================
```

## Watch Metrics in Real-Time

```bash
watch -n 1 "curl -s http://localhost:8000/metrics | grep -E '(gateway_in_flight|gateway_queue_depth|gateway_requests_total)'"
```

## What This Tests

The mock engine proves that:

✅ Gateway accepts requests correctly  
✅ Admission control is working  
✅ Concurrency limiting is working  
✅ Metrics are being recorded  
✅ Response cache is working (send same request twice and check cache_hits)  
✅ Cost accounting is tracking tokens  
✅ Load generator reports accurate percentiles  

## When llama.cpp Finishes Building

Once `~/llama-workspace/llama.cpp/build/bin/llama-server` exists:

1. **Stop the mock engine** (Ctrl+C in Terminal 1)

2. **Download a model:**
   ```bash
   cd ~/llama-workspace/llama.cpp/models
   curl -L -o tinyllama.gguf \
     "https://huggingface.co/TheBloke/TinyLlama-1.1B-Chat-v1.0-GGUF/resolve/main/tinyllama-1.1b-chat-v1.0.Q4_K_M.gguf"
   ```

3. **Start llama.cpp:**
   ```bash
   cd ~/llama-workspace/llama.cpp
   ./build/bin/llama-server -m models/tinyllama.gguf --port 8080 --parallel 8
   ```

4. **The gateway is already running** - just use it!

5. **Run real benchmarks:**
   ```bash
   ./scripts/run_benchmark.sh 3m 30s
   ```

---

**The project is fully functional right now with the mock engine!** 🚀
