import asyncio
from contextlib import asynccontextmanager


class AdmissionController:
    def __init__(self, max_in_flight: int, max_queue: int) -> None:
        self.max_in_flight = max_in_flight
        self.max_queue = max_queue
        self.capacity = max_in_flight + max_queue
        self._semaphore = asyncio.Semaphore(max_in_flight)
        self._pending = 0
        self._lock = asyncio.Lock()
        self._accepting = True

    @property
    def in_flight(self) -> int:
        return self.max_in_flight - self._semaphore._value  # noqa: SLF001

    @property
    def waiting(self) -> int:
        return max(self._pending - self.in_flight, 0)

    def stop_accepting(self) -> None:
        self._accepting = False

    async def drain(self) -> None:
        while self.in_flight > 0 or self.waiting > 0:
            await asyncio.sleep(0.01)

    @asynccontextmanager
    async def slot(self):
        if not self._accepting:
            raise OverflowError("gateway is shutting down")
        async with self._lock:
            if self._pending >= self.capacity:
                raise OverflowError("request queue is full")
            self._pending += 1
        try:
            await self._semaphore.acquire()
            try:
                yield
            finally:
                self._semaphore.release()
                async with self._lock:
                    self._pending -= 1
        except Exception:
            async with self._lock:
                if self._pending > 0:
                    self._pending -= 1
            raise
