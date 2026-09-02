from __future__ import annotations

import asyncio
import json
import time
from collections import defaultdict
from contextlib import asynccontextmanager
from typing import Any

from fastapi import FastAPI, HTTPException, Request
from fastapi.responses import JSONResponse, PlainTextResponse, StreamingResponse
from prometheus_client import CONTENT_TYPE_LATEST, Counter, Gauge, Histogram, generate_latest

from .admission import AdmissionController
from .cache import TTLCache
from .config import Settings
from .embeddings_batcher import EmbeddingsBatcher
from .engine import EngineClient
from .tokenizer import count_tokens, count_tokens_in_messages

REQ_TOTAL = Counter("gateway_requests_total", "Gateway requests", ["endpoint", "status"])
CACHE_TOTAL = Counter("gateway_cache_total", "Gateway cache requests", ["endpoint", "status"])
TOKENS_TOTAL = Counter("gateway_tokens_total", "Token usage", ["model", "kind"])
LATENCY = Histogram("gateway_latency_seconds", "Endpoint latency seconds", ["endpoint"])
IN_FLIGHT = Gauge("gateway_in_flight_requests", "In-flight request count")
WAITING = Gauge("gateway_waiting_requests", "Queued request count")
COST_PER_M = Gauge("gateway_cost_usd_per_million_tokens", "USD per million tokens", ["model", "config"])


def _system_prefix(payload: dict[str, Any], min_chars: int) -> str | None:
    messages = payload.get("messages")
    if not isinstance(messages, list):
        return None
    for message in messages:
        if message.get("role") == "system" and isinstance(message.get("content"), str):
            content = message["content"].strip()
            if len(content) >= min_chars:
                return content
    return None


class CostBook:
    def __init__(self, hourly_rate_usd: float, config_name: str) -> None:
        self.hourly_rate_usd = hourly_rate_usd
        self.config_name = config_name
        self.started = time.time()
        self.tokens_by_model = defaultdict(int)
        self._lock = asyncio.Lock()

    async def add_tokens(self, model: str, total_tokens: int) -> None:
        async with self._lock:
            self.tokens_by_model[model] += max(total_tokens, 0)
            elapsed_hours = max((time.time() - self.started) / 3600, 1e-9)
            for current_model, tokens in self.tokens_by_model.items():
                if tokens > 0:
                    dollars_per_million = (self.hourly_rate_usd * elapsed_hours / tokens) * 1_000_000
                    COST_PER_M.labels(model=current_model, config=self.config_name).set(dollars_per_million)


def create_app(settings: Settings | None = None, engine_client: EngineClient | None = None) -> FastAPI:
    settings = settings or Settings()
    engine_client = engine_client or EngineClient(settings.engine_base_url)
    admission = AdmissionController(settings.max_in_flight, settings.max_queue)
    cache = TTLCache(settings.cache_ttl_seconds)
    embeddings_batcher = EmbeddingsBatcher(
        engine_client=engine_client,
        max_batch_size=settings.embedding_batch_size,
        max_wait_ms=settings.embedding_batch_wait_ms,
    )
    prefix_locks: dict[str, asyncio.Lock] = defaultdict(asyncio.Lock)
    cost_book = CostBook(settings.cost_hourly_rate_usd, settings.config_name)

    @asynccontextmanager
    async def lifespan(app: FastAPI):
        await embeddings_batcher.start()
        app.state.accepting = True
        try:
            yield
        finally:
            app.state.accepting = False
            admission.stop_accepting()
            await admission.drain()
            await embeddings_batcher.stop()

    app = FastAPI(lifespan=lifespan)
    app.state.admission = admission
    app.state.engine_client = engine_client

    async def run_with_admission(endpoint: str, fn):
        started = time.time()
        try:
            async with admission.slot():
                IN_FLIGHT.set(admission.in_flight)
                WAITING.set(admission.waiting)
                result = await fn()
                REQ_TOTAL.labels(endpoint=endpoint, status="ok").inc()
                return result
        except OverflowError as exc:
            REQ_TOTAL.labels(endpoint=endpoint, status="rejected").inc()
            raise HTTPException(status_code=429, detail=str(exc)) from exc
        finally:
            LATENCY.labels(endpoint=endpoint).observe(time.time() - started)
            IN_FLIGHT.set(admission.in_flight)
            WAITING.set(admission.waiting)

    def cache_get(endpoint: str, payload: dict) -> dict | None:
        result = cache.get(payload)
        CACHE_TOTAL.labels(endpoint=endpoint, status="hit" if result else "miss").inc()
        return result

    async def account_usage(model: str, usage: dict) -> None:
        prompt = int(usage.get("prompt_tokens", 0))
        completion = int(usage.get("completion_tokens", 0))
        total = int(usage.get("total_tokens", prompt + completion))
        TOKENS_TOTAL.labels(model=model, kind="prompt").inc(prompt)
        TOKENS_TOTAL.labels(model=model, kind="completion").inc(completion)
        await cost_book.add_tokens(model=model, total_tokens=total)

    async def stream_text(model: str, text: str):
        words = text.split()
        for idx, word in enumerate(words):
            payload = {
                "id": f"chatcmpl-stream-{idx}",
                "object": "chat.completion.chunk",
                "choices": [{"index": 0, "delta": {"content": f"{word} "}, "finish_reason": None}],
                "model": model,
            }
            yield f"data: {json.dumps(payload)}\n\n"
            await asyncio.sleep(0)
        yield "data: [DONE]\n\n"

    @app.get("/healthz")
    async def healthz():
        return {"ok": True}

    @app.get("/metrics")
    async def metrics():
        return PlainTextResponse(generate_latest().decode("utf-8"), media_type=CONTENT_TYPE_LATEST)

    @app.post("/v1/completions")
    async def completions(request: Request):
        payload = await request.json()
        endpoint = "/v1/completions"
        if cached := cache_get(endpoint, payload):
            return JSONResponse(content=cached)

        async def execute():
            response = await engine_client.completions(payload)
            usage = response.get("usage", {})
            prompt = usage.get("prompt_tokens", count_tokens(str(payload.get("prompt", ""))))
            completion = usage.get("completion_tokens", count_tokens(response["choices"][0].get("text", "")))
            usage = {
                "prompt_tokens": int(prompt),
                "completion_tokens": int(completion),
                "total_tokens": int(prompt) + int(completion),
            }
            response["usage"] = usage
            await account_usage(response.get("model", payload.get("model", "unknown")), usage)
            return response

        response = await run_with_admission(endpoint, execute)
        if payload.get("stream"):
            text = response["choices"][0].get("text", "")
            return StreamingResponse(stream_text(response.get("model", "unknown"), text), media_type="text/event-stream")
        cache.set(payload, response)
        return JSONResponse(content=response)

    @app.post("/v1/chat/completions")
    async def chat_completions(request: Request):
        payload = await request.json()
        endpoint = "/v1/chat/completions"
        if cached := cache_get(endpoint, payload):
            return JSONResponse(content=cached)

        async def execute():
            prefix = _system_prefix(payload, settings.prefix_min_chars)
            if prefix:
                async with prefix_locks[prefix]:
                    response = await engine_client.chat_completions(payload)
            else:
                response = await engine_client.chat_completions(payload)
            usage = response.get("usage", {})
            prompt = usage.get("prompt_tokens", count_tokens_in_messages(payload.get("messages", [])))
            text = response["choices"][0].get("message", {}).get("content", "")
            completion = usage.get("completion_tokens", count_tokens(text))
            usage = {
                "prompt_tokens": int(prompt),
                "completion_tokens": int(completion),
                "total_tokens": int(prompt) + int(completion),
            }
            response["usage"] = usage
            await account_usage(response.get("model", payload.get("model", "unknown")), usage)
            return response

        response = await run_with_admission(endpoint, execute)
        if payload.get("stream"):
            text = response["choices"][0].get("message", {}).get("content", "")
            return StreamingResponse(stream_text(response.get("model", "unknown"), text), media_type="text/event-stream")
        cache.set(payload, response)
        return JSONResponse(content=response)

    @app.post("/v1/embeddings")
    async def embeddings(request: Request):
        payload = await request.json()
        endpoint = "/v1/embeddings"
        if cached := cache_get(endpoint, payload):
            return JSONResponse(content=cached)

        async def execute():
            response = await embeddings_batcher.submit(payload)
            usage = response.get("usage", {})
            prompt = int(usage.get("prompt_tokens", 0))
            usage = {"prompt_tokens": prompt, "total_tokens": int(usage.get("total_tokens", prompt))}
            response["usage"] = usage
            await account_usage(response.get("model", payload.get("model", "unknown")), usage)
            return response

        response = await run_with_admission(endpoint, execute)
        cache.set(payload, response)
        return JSONResponse(content=response)

    return app
