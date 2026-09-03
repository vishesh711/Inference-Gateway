# Quick Start - Web UI

Get the web interface running in 5 minutes.

## Prerequisites

1. **llama.cpp running** on port 8080
2. **Go 1.21+** installed
3. **Node.js 18+** installed

## Option 1: Automated Start (Easiest)

```bash
# Start everything with one command
./start-demo.sh
```

This will:
- ✅ Check if llama.cpp is running
- ✅ Build the gateway if needed
- ✅ Start gateway on port 8000
- ✅ Install frontend dependencies
- ✅ Start frontend on port 3000

Then open: **http://localhost:3000**

---

## Option 2: Manual Start (Step by Step)

### 1. Start llama.cpp (Terminal 1)

```bash
cd ~/Documents/Github/llama.cpp
./build/bin/llama-server \
  -m models/tinyllama-1.1b-chat-v1.0.Q4_K_M.gguf \
  --port 8080
```

### 2. Start Gateway (Terminal 2)

```bash
cd ~/Documents/Github/Inference-Gateway
go build -o bin/gateway cmd/gateway/main.go
./bin/gateway
```

Gateway runs on: **http://localhost:8000**

### 3. Start Frontend (Terminal 3)

```bash
cd ~/Documents/Github/Inference-Gateway/frontend
npm install  # First time only
npm run dev
```

Frontend runs on: **http://localhost:3000**

### 4. Open Browser

Visit: **http://localhost:3000**

---

## What You'll See

![Chat Interface](docs/images/screenshot.png)

### Features:

✅ **Real-time chat** with streaming responses  
✅ **Live metrics** - requests, tokens, latency  
✅ **Adjustable settings** - temperature, max tokens  
✅ **Dark mode** support  
✅ **Responsive design** - works on mobile  

---

## Testing the Interface

### 1. Basic Chat

Type: "Hello, tell me a joke"

You should see:
- Streaming response (text appears word-by-word)
- Updated metrics (tokens generated, latency)

### 2. Adjust Parameters

Try:
- **Temperature 0.0** - Deterministic responses
- **Temperature 2.0** - Creative/random responses
- **Max Tokens 10** - Short responses
- **Max Tokens 200** - Longer responses

### 3. Test Streaming Toggle

- **Streaming ON**: See text appear gradually
- **Streaming OFF**: See full response at once

---

## Architecture

```
┌──────────┐
│ Browser  │  ← You interact here
└────┬─────┘
     │
     ▼
┌──────────────┐
│  Frontend    │  Port 3000 (Next.js)
│  (React UI)  │
└──────┬───────┘
       │ HTTP API calls
       ▼
┌──────────────┐
│   Gateway    │  Port 8000 (Go)
│ (Admission   │  • Token scheduling
│  Control)    │  • Queue management
└──────┬───────┘
       │
       ▼
┌──────────────┐
│  llama.cpp   │  Port 8080
│  (TinyLlama) │  • 75-80 tok/s
└──────────────┘
```

---

## Troubleshooting

### Frontend shows "Failed to fetch"

**Problem**: Can't connect to backend

**Solution**:
```bash
# Check gateway is running
curl http://localhost:8000/health
# Should return: {"status":"ok"}

# If not, start gateway
./bin/gateway
```

### Responses are slow (>2 seconds)

**Problem**: Backend may be cold or CPU-bound

**Solution**:
- llama.cpp is CPU-only (no GPU)
- First request is slower (model loading)
- Subsequent requests should be faster
- Consider GPU build of llama.cpp for production

### Port already in use

**Problem**: Port 3000 or 8000 already taken

**Solution**:
```bash
# Frontend (change port)
cd frontend
PORT=3001 npm run dev

# Gateway (edit config.yaml)
# Change server.port to 8001
```

### Streaming not working

**Problem**: See full response at once even with streaming ON

**Solution**:
1. Check browser console for errors
2. Verify request includes `"stream": true`
3. Check Network tab shows "text/event-stream"

---

## API Examples

### Test with curl

```bash
# Health check
curl http://localhost:8000/health

# Non-streaming completion
curl -X POST http://localhost:8000/v1/completions \
  -H "Content-Type: application/json" \
  -d '{
    "model": "tinyllama",
    "prompt": "Hello, how are you?",
    "max_tokens": 50,
    "temperature": 0.7
  }'

# Streaming completion
curl -X POST http://localhost:8000/v1/completions \
  -H "Content-Type: application/json" \
  -d '{
    "model": "tinyllama",
    "prompt": "Tell me a story",
    "max_tokens": 100,
    "stream": true
  }'

# Embeddings
curl -X POST http://localhost:8000/v1/embeddings \
  -H "Content-Type: application/json" \
  -d '{
    "model": "tinyllama",
    "input": "Hello world"
  }'
```

---

## Metrics Dashboard

Visit: **http://localhost:8000/metrics**

You'll see Prometheus metrics:
- `gateway_requests_total` - Request counts
- `gateway_generation_seconds` - Latency histogram
- `gateway_queue_wait_seconds` - Queue time
- `gateway_in_flight` - Current requests

**Visualize with Grafana**:
1. Add Prometheus datasource pointing to http://localhost:8000/metrics
2. Create dashboard with panels for throughput, latency, queue depth

---

## Next Steps

### 1. Customize the UI

Edit `frontend/app/page.tsx` to add features:
- Prompt templates
- Chat history export
- Multiple model selection
- Advanced settings panel

### 2. Deploy to Production

See [DEPLOYMENT.md](DEPLOYMENT.md) for:
- Vercel deployment (frontend)
- DigitalOcean/AWS deployment (backend)
- Docker compose setup
- Monitoring setup

### 3. Add Authentication

For production, add API keys:
1. Generate keys for users
2. Add middleware in gateway
3. Update frontend to include auth header

---

## Performance Notes

With this setup you'll see:
- **Token generation**: 75-80 tokens/second (CPU)
- **Gateway overhead**: <50ms additional latency
- **Embeddings throughput**: 117+ req/s
- **Queue wait**: <5ms at p95

These are **real measured numbers** from TinyLlama 1.1B Q4 on Apple M4.

For faster performance:
- Use GPU-accelerated llama.cpp build
- Use larger/faster models
- Deploy on dedicated GPU server

---

## Support

**Issues?**
- Check [DEPLOYMENT.md](DEPLOYMENT.md) - Troubleshooting section
- Check [README.md](README.md) - Architecture overview
- Check logs: `gateway.log` and browser console

**Want to contribute?**
- See [frontend/README.md](frontend/README.md) for UI development
- See [docs/development/CONTRIBUTING.md](docs/development/CONTRIBUTING.md) for backend

---

**Enjoy your LLM Gateway! 🚀**

