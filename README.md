# Inference-Gateway

Inference-Gateway sits in front of an LLM inference engine (for example vLLM, llama.cpp, or TGI) and provides admission control, batching policy for embeddings, caching, and observability with OpenAI-compatible APIs.

## Who this is for

- **Primary:** platform engineers serving models to internal teams, who need predictable latency under load and defensible cost/accounting.
- **Secondary:** application developers who already speak OpenAI API shapes and do not want app-level changes.

## Goals

- Keep p95 latency within a target by controlling in-flight concurrency and bounded queueing.
- Reduce redundant computation via response caching and visible hit/miss metrics.
- Attribute spend as **dollars per million tokens**, per model and gateway configuration.
- Make behavior measurable so config changes can be evaluated with data.

## Non-goals (explicitly out of scope)

- Training or fine-tuning models.
- Building a new inference engine (the engine remains vLLM/llama.cpp/TGI).
- Multi-node distribution or autoscaling.
- Authentication beyond static API key patterns.
- Chat UI development.

## Implemented functionality

### P0

- `POST /v1/completions` and `POST /v1/chat/completions` with OpenAI-compatible request/response envelopes.
- Bounded queue + configurable max in-flight concurrency (`MAX_IN_FLIGHT`, `MAX_QUEUE`).
- Overload rejection with HTTP `429` when queue capacity is exhausted.
- Per-request token accounting (`usage.prompt_tokens`, `usage.completion_tokens`, `usage.total_tokens`).
- Prometheus metrics endpoint at `GET /metrics`.
- Graceful shutdown via FastAPI lifespan: stop accepting work, drain in-flight, then exit.

### P1

- `POST /v1/embeddings` with **true gateway-side request coalescing** into one engine call per batch window.
- TTL response cache keyed by full normalized request body (`CACHE_TTL_SECONDS`).
- Prefix-aware routing for chat requests with long system prompts (`PREFIX_MIN_CHARS`) to improve engine prefix-cache locality.
- Cost accounting with configurable hourly rate (`COST_HOURLY_RATE_USD`) exported as `gateway_cost_usd_per_million_tokens`.
- Server-sent event streaming for completions/chat when `stream: true`.

### Honest layer boundary

- For text/chat, the inference engine still performs low-level continuous batching.
- The gateway controls **admission, queue depth, and in-flight pressure**.
- For embeddings, the gateway performs genuine request coalescing before forwarding.

## Configuration

Environment variables:

- `MAX_IN_FLIGHT` (default `8`)
- `MAX_QUEUE` (default `64`)
- `CACHE_TTL_SECONDS` (default `30`)
- `EMBEDDING_BATCH_SIZE` (default `16`)
- `EMBEDDING_BATCH_WAIT_MS` (default `10`)
- `PREFIX_MIN_CHARS` (default `120`)
- `COST_HOURLY_RATE_USD` (default `2.0`)
- `GATEWAY_CONFIG_NAME` (default `default`)
- `ENGINE_BASE_URL` (optional; if unset, a deterministic local mock engine is used)

## Run

```bash
pip install -e ".[test]"
python main.py
```

## Metrics examples

- `gateway_requests_total{endpoint,status}`
- `gateway_latency_seconds_bucket{endpoint,...}`
- `gateway_cache_total{endpoint,status}`
- `gateway_in_flight_requests`
- `gateway_waiting_requests`
- `gateway_tokens_total{model,kind}`
- `gateway_cost_usd_per_million_tokens{model,config}`