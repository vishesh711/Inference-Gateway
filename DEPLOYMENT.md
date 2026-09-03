# Deployment Guide

## Architecture

```
┌─────────────┐     ┌──────────────┐     ┌─────────────┐
│   Browser   │────▶│   Frontend   │────▶│   Gateway   │
│  (Client)   │◀────│  (Next.js)   │◀────│  (Go 8000)  │
└─────────────┘     └──────────────┘     └─────────────┘
                    Vercel/Port 3000            │
                                                 ▼
                                         ┌─────────────┐
                                         │  llama.cpp  │
                                         │  (Port 8080)│
                                         └─────────────┘
```

## Quick Start (Local Development)

### 1. Start Backend (Gateway)

```bash
# Terminal 1: Start llama.cpp
cd ~/Documents/Github/llama.cpp
./build/bin/llama-server -m models/tinyllama-1.1b-chat-v1.0.Q4_K_M.gguf --port 8080

# Terminal 2: Start Gateway
cd ~/Documents/Github/Inference-Gateway
./bin/gateway
# Gateway runs on http://localhost:8000
```

### 2. Start Frontend

```bash
# Terminal 3: Install dependencies (first time only)
cd ~/Documents/Github/Inference-Gateway/frontend
npm install

# Start development server
npm run dev
# Frontend runs on http://localhost:3000
```

### 3. Open Browser

Visit: http://localhost:3000

You should see the chat interface connected to the gateway.

---

## Production Deployment

### Option 1: Deploy to Vercel (Frontend) + DigitalOcean/AWS (Backend)

#### Frontend on Vercel

1. **Push to GitHub:**
   ```bash
   git add .
   git commit -m "Add frontend"
   git push origin main
   ```

2. **Deploy on Vercel:**
   - Go to [vercel.com](https://vercel.com)
   - Import your GitHub repository
   - Set **Root Directory**: `frontend`
   - Add environment variable:
     - `NEXT_PUBLIC_API_URL` = `https://your-backend-domain.com`
   - Deploy

3. **Update CORS in Gateway:**
   Edit `cmd/gateway/main.go` to allow your Vercel domain:
   ```go
   // Allow your Vercel domain
   allowedOrigins := []string{
       "http://localhost:3000",
       "https://your-app.vercel.app",
   }
   ```

#### Backend on DigitalOcean/AWS

1. **Create a server** (Ubuntu 22.04)

2. **Install dependencies:**
   ```bash
   # Install Go
   wget https://go.dev/dl/go1.21.linux-amd64.tar.gz
   sudo tar -C /usr/local -xzf go1.21.linux-amd64.tar.gz
   export PATH=$PATH:/usr/local/go/bin
   
   # Install llama.cpp
   git clone https://github.com/ggerganov/llama.cpp
   cd llama.cpp && make
   ```

3. **Build Gateway:**
   ```bash
   cd ~/Inference-Gateway
   go build -o gateway cmd/gateway/main.go
   ```

4. **Run with systemd:**
   Create `/etc/systemd/system/inference-gateway.service`:
   ```ini
   [Unit]
   Description=Inference Gateway
   After=network.target

   [Service]
   Type=simple
   User=ubuntu
   WorkingDirectory=/home/ubuntu/Inference-Gateway
   ExecStart=/home/ubuntu/Inference-Gateway/gateway
   Restart=always

   [Install]
   WantedBy=multi-user.target
   ```

   ```bash
   sudo systemctl enable inference-gateway
   sudo systemctl start inference-gateway
   ```

5. **Setup Nginx reverse proxy:**
   ```nginx
   server {
       listen 80;
       server_name your-domain.com;

       location / {
           proxy_pass http://localhost:8000;
           proxy_http_version 1.1;
           proxy_set_header Upgrade $http_upgrade;
           proxy_set_header Connection 'upgrade';
           proxy_set_header Host $host;
           proxy_cache_bypass $http_upgrade;
       }
   }
   ```

---

### Option 2: Deploy Both on Single Server

#### Using Docker Compose

Create `docker-compose.yml`:

```yaml
version: '3.8'

services:
  gateway:
    build:
      context: .
      dockerfile: Dockerfile
    ports:
      - "8000:8000"
    environment:
      - CONFIG_PATH=/app/config.yaml
    volumes:
      - ./config.yaml:/app/config.yaml
    restart: always

  frontend:
    build:
      context: ./frontend
      dockerfile: Dockerfile
    ports:
      - "3000:3000"
    environment:
      - NEXT_PUBLIC_API_URL=http://localhost:8000
    depends_on:
      - gateway
    restart: always

  nginx:
    image: nginx:alpine
    ports:
      - "80:80"
      - "443:443"
    volumes:
      - ./nginx.conf:/etc/nginx/nginx.conf:ro
    depends_on:
      - frontend
      - gateway
    restart: always
```

Run:
```bash
docker-compose up -d
```

---

### Option 3: Serverless (Frontend) + Traditional Server (Backend)

**Frontend**: Vercel (serverless, auto-scaling)  
**Backend**: VPS with GPU (for llama.cpp)

This is the recommended setup for production:
- Frontend scales automatically
- Backend runs on hardware you control
- Clear separation of concerns

---

## Environment Variables

### Frontend

Create `frontend/.env.local`:
```env
NEXT_PUBLIC_API_URL=http://localhost:8000
```

For production, set in Vercel dashboard or your hosting platform.

### Backend

Edit `config.yaml`:
```yaml
server:
  port: 8000
  
backends:
  - id: llama1
    url: http://localhost:8080
    model: tinyllama
    capacity: 8
```

---

## Testing the Deployment

### 1. Health Check

```bash
# Backend
curl http://localhost:8000/health

# Should return: {"status":"ok"}
```

### 2. Test Completion

```bash
curl -X POST http://localhost:8000/v1/completions \
  -H "Content-Type: application/json" \
  -d '{
    "model": "tinyllama",
    "prompt": "Hello, how are you?",
    "max_tokens": 50
  }'
```

### 3. Test Frontend

1. Open browser: http://localhost:3000
2. Type a message
3. Should see streaming response

---

## Monitoring

### Prometheus Metrics

Gateway exposes metrics at: `http://localhost:8000/metrics`

**Key metrics:**
- `gateway_requests_total` - Total requests by status
- `gateway_generation_seconds` - Generation latency histogram
- `gateway_queue_wait_seconds` - Queue wait time histogram
- `gateway_in_flight` - Current in-flight requests

### Setup Grafana Dashboard

1. **Add Prometheus data source** pointing to gateway metrics
2. **Import dashboard** or create custom panels
3. **Monitor:**
   - Request rate
   - p95 latency
   - Queue depth
   - Error rate

---

## Troubleshooting

### Frontend can't reach backend

**Symptom:** "Failed to fetch" errors in browser console

**Solutions:**
1. Check backend is running: `curl http://localhost:8000/health`
2. Verify CORS is enabled in gateway
3. Check `next.config.js` proxy configuration
4. Open browser console and check network tab

### Streaming not working

**Symptom:** Response comes all at once, not streaming

**Solutions:**
1. Verify backend supports streaming (check `internal/handler/streaming.go`)
2. Check request includes `"stream": true`
3. Verify SSE format in response
4. Check browser network tab shows "text/event-stream"

### High latency

**Symptom:** Responses take >2 seconds

**Solutions:**
1. Check llama.cpp is using GPU if available
2. Reduce `max_tokens` in request
3. Check gateway metrics for queue wait time
4. Monitor backend CPU/memory usage

---

## Security Considerations

### For Production

1. **Enable HTTPS:**
   ```bash
   # Use Let's Encrypt
   sudo certbot --nginx -d your-domain.com
   ```

2. **Set strong CORS policy:**
   ```go
   // Only allow your frontend domain
   allowedOrigins := []string{
       "https://your-app.vercel.app",
   }
   ```

3. **Add rate limiting** (optional):
   ```go
   // Use go-rate library or nginx limit_req
   ```

4. **Environment secrets:**
   - Never commit `.env` files
   - Use secret management (Vercel secrets, AWS Secrets Manager)

5. **API authentication** (future enhancement):
   - Add API keys
   - Implement JWT tokens
   - Rate limit per user

---

## Cost Estimation

### Vercel (Frontend)
- **Free tier:** Perfect for personal projects
- **Pro:** $20/month for commercial use

### Backend Server (AWS/DigitalOcean)
- **CPU-only (t3.medium):** ~$30/month
- **GPU (g4dn.xlarge):** ~$500/month
- **Spot instances:** 70% cheaper

### Recommended for Testing
- **Frontend:** Vercel Free Tier
- **Backend:** DigitalOcean CPU Droplet $12/month
- **Total:** $12/month

---

## Next Steps

1. **Deploy frontend** to Vercel
2. **Deploy backend** to your server
3. **Point frontend** to backend URL
4. **Test end-to-end**
5. **Monitor metrics**
6. **Add authentication** (if needed)

For questions, check:
- [README.md](README.md) - Project overview
- [BENCHMARK_SUMMARY.md](BENCHMARK_SUMMARY.md) - Performance data
- Frontend README: [frontend/README.md](frontend/README.md)

