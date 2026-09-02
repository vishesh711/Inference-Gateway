from __future__ import annotations

import hashlib
import time
import uuid

import httpx

from .tokenizer import count_tokens, count_tokens_in_messages


class EngineClient:
    def __init__(self, base_url: str = "") -> None:
        self.base_url = base_url
        self.embedding_calls = 0

    async def completions(self, payload: dict) -> dict:
        if self.base_url:
            async with httpx.AsyncClient(timeout=60) as client:
                response = await client.post(f"{self.base_url}/v1/completions", json=payload)
                response.raise_for_status()
                return response.json()

        prompt = payload.get("prompt", "")
        text = f"Echo: {prompt}" if prompt else "Echo:"
        prompt_tokens = count_tokens(prompt)
        completion_tokens = count_tokens(text)
        return {
            "id": f"cmpl-{uuid.uuid4().hex}",
            "object": "text_completion",
            "created": int(time.time()),
            "model": payload.get("model", "mock-model"),
            "choices": [{"index": 0, "text": text, "finish_reason": "stop"}],
            "usage": {
                "prompt_tokens": prompt_tokens,
                "completion_tokens": completion_tokens,
                "total_tokens": prompt_tokens + completion_tokens,
            },
        }

    async def chat_completions(self, payload: dict) -> dict:
        if self.base_url:
            async with httpx.AsyncClient(timeout=60) as client:
                response = await client.post(f"{self.base_url}/v1/chat/completions", json=payload)
                response.raise_for_status()
                return response.json()

        messages = payload.get("messages", [])
        prompt = " ".join((m.get("content", "") for m in messages if isinstance(m.get("content"), str)))
        text = f"Echo: {prompt}".strip()
        prompt_tokens = count_tokens_in_messages(messages)
        completion_tokens = count_tokens(text)
        return {
            "id": f"chatcmpl-{uuid.uuid4().hex}",
            "object": "chat.completion",
            "created": int(time.time()),
            "model": payload.get("model", "mock-model"),
            "choices": [
                {
                    "index": 0,
                    "message": {"role": "assistant", "content": text},
                    "finish_reason": "stop",
                }
            ],
            "usage": {
                "prompt_tokens": prompt_tokens,
                "completion_tokens": completion_tokens,
                "total_tokens": prompt_tokens + completion_tokens,
            },
        }

    async def embeddings(self, payload: dict) -> dict:
        if self.base_url:
            async with httpx.AsyncClient(timeout=60) as client:
                response = await client.post(f"{self.base_url}/v1/embeddings", json=payload)
                response.raise_for_status()
                return response.json()

        self.embedding_calls += 1
        model = payload.get("model", "mock-embedding")
        inputs = payload.get("input", [])
        if isinstance(inputs, str):
            inputs = [inputs]
        data = []
        prompt_tokens = 0
        for idx, item in enumerate(inputs):
            text = item if isinstance(item, str) else str(item)
            prompt_tokens += count_tokens(text)
            digest = hashlib.sha256(text.encode("utf-8")).digest()
            vector = [round(b / 255.0, 6) for b in digest[:8]]
            data.append({"object": "embedding", "index": idx, "embedding": vector})
        return {
            "object": "list",
            "model": model,
            "data": data,
            "usage": {"prompt_tokens": prompt_tokens, "total_tokens": prompt_tokens},
        }
