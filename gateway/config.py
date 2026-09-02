from dataclasses import dataclass
import os


def _env_int(name: str, default: int) -> int:
    value = os.getenv(name)
    return int(value) if value is not None else default


def _env_float(name: str, default: float) -> float:
    value = os.getenv(name)
    return float(value) if value is not None else default


@dataclass(frozen=True)
class Settings:
    max_in_flight: int = _env_int("MAX_IN_FLIGHT", 8)
    max_queue: int = _env_int("MAX_QUEUE", 64)
    cache_ttl_seconds: float = _env_float("CACHE_TTL_SECONDS", 30.0)
    embedding_batch_size: int = _env_int("EMBEDDING_BATCH_SIZE", 16)
    embedding_batch_wait_ms: int = _env_int("EMBEDDING_BATCH_WAIT_MS", 10)
    prefix_min_chars: int = _env_int("PREFIX_MIN_CHARS", 120)
    cost_hourly_rate_usd: float = _env_float("COST_HOURLY_RATE_USD", 2.0)
    engine_base_url: str = os.getenv("ENGINE_BASE_URL", "").rstrip("/")
    config_name: str = os.getenv("GATEWAY_CONFIG_NAME", "default")

