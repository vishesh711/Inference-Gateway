import asyncio
from contextlib import asynccontextmanager

import pytest
from asgi_lifespan import LifespanManager
from httpx import ASGITransport, AsyncClient

from gateway.app import create_app
from gateway.config import Settings
from gateway.engine import EngineClient


@asynccontextmanager
async def app_client(app):
    async with LifespanManager(app):
        transport = ASGITransport(app=app)
        async with AsyncClient(transport=transport, base_url="http://test") as client:
            yield client


@pytest.mark.asyncio
async def test_completions_cache_and_metrics():
    app = create_app(Settings())
    async with app_client(app) as client:
        payload = {"model": "m1", "prompt": "hello world"}
        first = await client.post("/v1/completions", json=payload)
        second = await client.post("/v1/completions", json=payload)
        assert first.status_code == 200
        assert second.status_code == 200
        assert first.json() == second.json()

        metrics = await client.get("/metrics")
        body = metrics.text
        assert 'gateway_cache_total{endpoint="/v1/completions",status="miss"}' in body
        assert 'gateway_cache_total{endpoint="/v1/completions",status="hit"}' in body
        assert '"usage"' in first.text


class SlowEngine(EngineClient):
    async def completions(self, payload: dict) -> dict:
        await asyncio.sleep(0.2)
        return await super().completions(payload)


@pytest.mark.asyncio
async def test_queue_rejects_with_429_when_full():
    app = create_app(Settings(max_in_flight=1, max_queue=0), engine_client=SlowEngine())
    async with app_client(app) as client:
        payload = {"model": "m1", "prompt": "slow"}

        async def send():
            return await client.post("/v1/completions", json=payload)

        first, second = await asyncio.gather(send(), send())
        statuses = sorted([first.status_code, second.status_code])
        assert statuses == [200, 429]


@pytest.mark.asyncio
async def test_embeddings_coalesced_batching():
    engine = EngineClient()
    app = create_app(Settings(embedding_batch_size=8, embedding_batch_wait_ms=30), engine_client=engine)
    async with app_client(app) as client:
        payloads = [
            {"model": "embed-1", "input": "alpha"},
            {"model": "embed-1", "input": "beta"},
            {"model": "embed-1", "input": "gamma"},
        ]
        responses = await asyncio.gather(*[client.post("/v1/embeddings", json=p) for p in payloads])
        assert all(r.status_code == 200 for r in responses)
        assert engine.embedding_calls == 1
        assert all(len(r.json()["data"]) == 1 for r in responses)


@pytest.mark.asyncio
async def test_chat_streaming_sse():
    app = create_app(Settings())
    async with app_client(app) as client:
        response = await client.post(
            "/v1/chat/completions",
            json={
                "model": "m1",
                "stream": True,
                "messages": [
                    {"role": "system", "content": "s" * 130},
                    {"role": "user", "content": "hello"},
                ],
            },
        )
        assert response.status_code == 200
        assert response.headers["content-type"].startswith("text/event-stream")
        assert "data:" in response.text
        assert "[DONE]" in response.text
