# Inference Gateway - Frontend

Modern web interface for the LLM Inference Gateway.

## Features

- 💬 Real-time chat interface with streaming support
- 📊 Live metrics dashboard (requests, tokens, latency)
- ⚙️ Adjustable parameters (temperature, max tokens)
- 🎨 Dark mode support
- 📱 Responsive design

## Getting Started

### Prerequisites

- Node.js 18+ installed
- Backend gateway running on http://localhost:8000

### Installation

```bash
npm install
```

### Development

```bash
npm run dev
```

Open [http://localhost:3000](http://localhost:3000) in your browser.

### Production Build

```bash
npm run build
npm start
```

## Architecture

- **Framework**: Next.js 14 with App Router
- **Styling**: Tailwind CSS
- **HTTP Client**: Axios + Fetch API (for streaming)
- **Type Safety**: TypeScript

## API Integration

The frontend connects to the gateway backend:

- `POST /api/v1/completions` - Non-streaming completions
- `POST /api/v1/chat/completions` - Chat completions (with streaming)
- `GET /api/v1/health` - Health check

## Configuration

API endpoint is configured in `next.config.js`:

```javascript
async rewrites() {
  return [
    {
      source: '/api/v1/:path*',
      destination: 'http://localhost:8000/v1/:path*',
    },
  ]
}
```

For production, update this to your deployed backend URL.

## Deployment

### Vercel (Recommended)

1. Push to GitHub
2. Import project in Vercel
3. Set environment variable `NEXT_PUBLIC_API_URL` to your backend URL
4. Deploy

### Docker

```bash
docker build -t inference-gateway-ui .
docker run -p 3000:3000 inference-gateway-ui
```

## Features Demonstrated

This UI showcases the gateway's capabilities:

- **Token-aware scheduling**: Requests are scheduled by estimated token cost
- **Streaming responses**: Low Time-To-First-Token (TTFT)
- **Real-time metrics**: Queue depth, latency, throughput
- **Admission control**: Handles overload gracefully with 429 responses

## Development Roadmap

- [ ] Embeddings interface
- [ ] Multi-backend selection
- [ ] Prompt templates library
- [ ] Export chat history
- [ ] Advanced settings panel
