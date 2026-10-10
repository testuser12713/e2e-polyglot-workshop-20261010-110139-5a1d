"""Tests for the worker queue loop and its queue-client configuration."""

from __future__ import annotations

import redis
import worker.main as main
from worker.config import Settings

SETTINGS = Settings(
    database_url="postgresql://user:pw@localhost:5432/app",
    valkey_url="redis://localhost:6379/0",
    queue_name="workshop:invoices",
    hourly_rate_cents=8900,
)


class FakeClient:
    """A Valkey client whose ``brpop`` replays a scripted list of results."""

    def __init__(self, script: list[object]) -> None:
        self._script = list(script)

    def brpop(self, key: str, timeout: int = 0) -> object:
        if not self._script:
            return None
        item = self._script.pop(0)
        if isinstance(item, Exception):
            raise item
        return item


class FakeConnection:
    """Minimal stand-in for a psycopg connection tracking commits/rollbacks."""

    def __init__(self) -> None:
        self.commits = 0
        self.rollbacks = 0

    def commit(self) -> None:
        self.commits += 1

    def rollback(self) -> None:
        self.rollbacks += 1


def test_connect_queue_has_no_client_read_timeout() -> None:
    """The blocking BRPOP must not race a client-side socket timeout."""
    client = main.connect_queue(SETTINGS)
    try:
        assert client.connection_pool.connection_kwargs["socket_timeout"] is None
    finally:
        client.close()


def test_run_loop_continues_after_a_queue_error(monkeypatch) -> None:
    processed: list[object] = []
    monkeypatch.setattr(main, "process_message", lambda conn, raw, settings: processed.append(raw))
    monkeypatch.setattr(main.time, "sleep", lambda seconds: None)

    message = ("workshop:invoices", '{"orderNumber":"AU-1","orderId":1}')
    client = FakeClient([redis.exceptions.TimeoutError("timeout"), message, None])
    conn = FakeConnection()

    checks = {"n": 0}

    def should_stop() -> bool:
        checks["n"] += 1
        return checks["n"] > 3

    main.run_loop(conn, client, SETTINGS, should_stop)

    assert processed == [message[1]]
    assert conn.commits == 1
