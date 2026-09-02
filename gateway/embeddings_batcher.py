from __future__ import annotations

import asyncio
from dataclasses import dataclass
from typing import Any

from .tokenizer import count_tokens


@dataclass
class _PendingEmbedding:
    payload: dict[str, Any]
    future: asyncio.Future


class EmbeddingsBatcher:
    def __init__(self, engine_client, max_batch_size: int, max_wait_ms: int) -> None:
        self.engine_client = engine_client
        self.max_batch_size = max_batch_size
        self.max_wait_ms = max_wait_ms
        self._queue: asyncio.Queue[_PendingEmbedding] = asyncio.Queue()
        self._task: asyncio.Task | None = None
        self._stop = asyncio.Event()

    async def start(self) -> None:
        self._task = asyncio.create_task(self._worker())

    async def stop(self) -> None:
        self._stop.set()
        if self._task:
            await self._task

    async def submit(self, payload: dict) -> dict:
        loop = asyncio.get_running_loop()
        fut = loop.create_future()
        await self._queue.put(_PendingEmbedding(payload=payload, future=fut))
        return await fut

    async def _worker(self) -> None:
        while not self._stop.is_set():
            try:
                first = await asyncio.wait_for(self._queue.get(), timeout=0.05)
            except asyncio.TimeoutError:
                continue
            batch = [first]
            deadline = asyncio.get_running_loop().time() + (self.max_wait_ms / 1000.0)
            while len(batch) < self.max_batch_size:
                timeout = deadline - asyncio.get_running_loop().time()
                if timeout <= 0:
                    break
                try:
                    batch.append(await asyncio.wait_for(self._queue.get(), timeout=timeout))
                except asyncio.TimeoutError:
                    break
            await self._flush(batch)
        while not self._queue.empty():
            batch = [self._queue.get_nowait()]
            while len(batch) < self.max_batch_size and not self._queue.empty():
                batch.append(self._queue.get_nowait())
            await self._flush(batch)

    async def _flush(self, batch: list[_PendingEmbedding]) -> None:
        flattened_inputs = []
        ranges: list[tuple[int, int, _PendingEmbedding]] = []
        for pending in batch:
            item = pending.payload.get("input", [])
            model = pending.payload.get("model")
            if isinstance(item, list):
                inputs = item
            else:
                inputs = [item]
            start = len(flattened_inputs)
            flattened_inputs.extend(inputs)
            ranges.append((start, len(flattened_inputs), pending))
            pending.payload["_batch_model"] = model

        model = next((x.payload.get("model") for x in batch if x.payload.get("model")), "mock-embedding")
        try:
            merged = await self.engine_client.embeddings({"model": model, "input": flattened_inputs})
            merged_data = merged.get("data", [])
            for start, end, pending in ranges:
                segment = merged_data[start:end]
                result = {
                    "object": "list",
                    "model": pending.payload.get("model", model),
                    "data": [
                        {"object": "embedding", "index": idx, "embedding": row["embedding"]}
                        for idx, row in enumerate(segment)
                    ],
                    "usage": {
                        "prompt_tokens": sum(
                            count_tokens(str(item))
                            for item in (pending.payload.get("input") if isinstance(pending.payload.get("input"), list) else [pending.payload.get("input")])
                        ),
                        "total_tokens": sum(
                            count_tokens(str(item))
                            for item in (pending.payload.get("input") if isinstance(pending.payload.get("input"), list) else [pending.payload.get("input")])
                        ),
                    },
                }
                pending.future.set_result(result)
        except Exception as exc:  # pragma: no cover
            for _, _, pending in ranges:
                if not pending.future.done():
                    pending.future.set_exception(exc)
