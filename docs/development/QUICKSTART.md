# Quick Start Guide

Get the Inference Gateway running in under 5 minutes.

## Option 1: Using llama.cpp (Recommended for testing)

The setup script is building llama.cpp in `~/llama-workspace/llama.cpp`. Once it completes:

### 1. Download a small model

```bash
cd ~/llama-workspace/llama.cpp/models

# Download TinyLlama (Q4, ~700MB) - good for testing
curl -L -o tinyllama-1.1b-chat-q4_k_m.gguf \
  "https://huggingface.co/TheBloke/TinyLlama-1.1B-Chat-v1.0-GGUF/resolve/main/tinyllama-1.1b-chat-v1.0.Q4_K_M.gguf"
```

### 2. Start llama.cpp server

```bash
cd ~/llama-workspace/llama.cpp

./build/bin/llama-server \
  -m models/tinyllama-1.1b-chat-q4_k_m.gguf \
  --port 8080 \
  --parallel 8 \
  -c 4096 \
  --host 0.0.0.0
```

Leave this running in one terminal.

### 3. Start the gateway

In a new terminal:

```bash
cd ~/Documents/Github/Inference-Gateway

./bin/gateway
```

You should see:
```
2026/09/02 15:44:14 Starting Inference Gateway on port 8000
2026/09/02 15:44:14 Engine URL: http://localhost:8080
2026/09/02 15:44:14 Listening on http://localhost:8000
```

### 4. Test it!

```bash
curl -X POST http://localhost:8000/v1/completions \
  -H "Content-Type: application/json" \
  -d '{
    "model": "tinyllama",
    "prompt": "The capital of France is",
    "max_tokens": 20,
    "temperature": 0.7
  }'
```

### 5. View metrics

```bash
open http://localhost:8000/metrics
```

Or with curl:
```bash
curl http://localhost:8000/metrics | grep gateway_
```

---

## Option 2: Mock Backend for Gateway Testing

If you don't have a model yet, test the gateway with a mock backend:

### 1. Create a mock llama.cpp server

```bash
cd ~/Documents/Github/Inference-Gateway
mkdir -p scripts
cat > scripts/mock_engine.py << 'EOF'
#!/usr/bin/env python3
"""Mock LLM engine for testing the gateway"""
import json
import time
from http.server import HTTPServer, BaseHTTPRequestHandler

class MockLLMHandler(BaseHTTPRequestHandler):
    def do_POST(self):
        if self.path == '/v1/completions':
            content_length = int(self.headers['Content-Length'])
            body = self.rfile.read(content_length)
            req = json.loads(body)
            
            # Simulate some processing time
            time.sleep(0.1)
            
            response = {
                "id": "mock-1",
                "object": "text_completion",
                "created": int(time.time()),
                "model": req.get("model", "mock"),
                "choices": [{
                    "text": " Paris. It is known for the Eiffel Tower.",
                    "index": 0,
                    "finish_reason": "stop"
                }],
                "usage": {
                    "prompt_tokens": 10,
                    "completion_tokens": 20,
                    "total_tokens": 30
                }
            }
            
            self.send_response(200)
            self.send_header('Content-type', 'application/json')
            self.end_headers()
            self.wfile.write(json.dumps(response).encode())
        else:
            self.send_response(404)
            self.end_headers()
    
    def log_message(self, format, *args):
        print(f"Mock Engine: {format % args}")

if __name__ == '__main__':
    server = HTTPServer(('localhost', 8080), MockLLMHandler)
    print("Mock LLM engine running on http://localhost:8080")
    server.serve_forever()
EOF

chmod +x scripts/mock_engine.py
```

### 2. Start mock engine

```bash
python3 scripts/mock_engine.py
```

### 3. Start gateway and test

Follow steps 3-5 from Option 1 above.

---

## Running Benchmarks

Once you have either option running:

### Quick test (30 seconds)

```bash
./bin/loadgen -workers 4 -duration 30s -warmup 5s
```

### Full concurrency sweep

This tests concurrency levels 1, 2, 4, 8, 16, 32 (takes ~20 minutes):

```bash
./scripts/run_benchmark.sh 3m 30s
```

Results will be in `results/benchmark_TIMESTAMP/`

---

## Troubleshooting

### Gateway can't connect to engine

```
Error: dial tcp 127.0.0.1:8080: connect: connection refused
```

**Solution:** Make sure llama-server or the mock engine is running on port 8080.

### Port already in use

```
Error: bind: address already in use
```

**Solution:** Kill existing process or change port in config.yaml

```bash
# Find and kill process on port 8000
lsof -ti:8000 | xargs kill -9

# Or change the port
sed -i '' 's/port: 8000/port: 8001/' config.yaml
```

### llama.cpp build taking forever

The build is CPU-intensive. If it's still running:
- Let it finish (it's at ~49% completion based on your output)
- Or use Option 2 (mock backend) to test the gateway now
- Or download pre-built llama.cpp binaries

---

## Next Steps

Once it's working:

1. **Try different prompts**
   ```bash
   curl -X POST http://localhost:8000/v1/completions \
     -H "Content-Type: application/json" \
     -d '{"model":"tinyllama","prompt":"Write a haiku about Go","max_tokens":50}'
   ```

2. **Test chat completions**
   ```bash
   curl -X POST http://localhost:8000/v1/chat/completions \
     -H "Content-Type: application/json" \
     -d '{"model":"tinyllama","messages":[{"role":"user","content":"Hello!"}]}'
   ```

3. **Watch metrics in real-time**
   ```bash
   watch -n 1 "curl -s http://localhost:8000/metrics | grep -E 'gateway_(in_flight|queue_depth|requests_total)'"
   ```

4. **Run the concurrency sweep** to populate the benchmark tables in README.md

5. **Try different quantizations** (Q4 vs Q8 vs F16) to compare throughput vs quality
