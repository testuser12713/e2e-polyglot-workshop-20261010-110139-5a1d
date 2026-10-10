"""Entry point of the invoice worker.

Connects to PostgreSQL and Valkey, verifies both connections, reports
readiness, then pops one message at a time from the queue and hands it to the
processor. A message that fails to process is logged without customer data and
the loop continues.
"""

from __future__ import annotations

import logging
import signal
import time
from collections.abc import Callable
from types import FrameType

import psycopg
import redis

from .config import Settings, load_settings
from .processor import process_message

logger = logging.getLogger("worker")

BRPOP_TIMEOUT_SECONDS = 5
RECONNECT_DELAY_SECONDS = 1.0


def connect_database(settings: Settings) -> psycopg.Connection:
    """Open a PostgreSQL connection from the validated configuration."""
    return psycopg.connect(settings.database_url)


def connect_queue(settings: Settings) -> redis.Redis:
    """Open a Valkey client from the validated configuration.

    ``socket_timeout=None`` is required for a blocking consumer: redis-py's
    default client-side read timeout (5s) is at the same value as the blocking
    ``BRPOP`` timeout, so the socket read aborts the blocking command with a
    spurious ``TimeoutError``. A blocking read needs no client timeout.
    """
    return redis.Redis.from_url(
        settings.valkey_url,
        decode_responses=True,
        socket_timeout=None,
    )


def verify_connections(conn: psycopg.Connection, client: redis.Redis) -> None:
    """Verify that database and queue are reachable before serving."""
    conn.execute("SELECT 1")
    client.ping()


def run_loop(
    conn: psycopg.Connection,
    client: redis.Redis,
    settings: Settings,
    should_stop: Callable[[], bool],
) -> None:
    """Pop exactly one message at a time and pass it to the processor."""
    while not should_stop():
        try:
            item = client.brpop(settings.queue_name, timeout=BRPOP_TIMEOUT_SECONDS)
        except redis.RedisError as exc:
            # A transient queue error must never kill the worker; it keeps
            # draining once the queue is reachable again.
            logger.error("queue read failed (%s); retrying", type(exc).__name__)
            time.sleep(RECONNECT_DELAY_SECONDS)
            continue

        if item is None:
            continue

        _, raw_message = item
        try:
            process_message(conn, raw_message, settings)
            conn.commit()
        except Exception as exc:
            # A malformed or failing message must never kill the worker.
            conn.rollback()
            logger.error("failed to process a queue message (%s); continuing", type(exc).__name__)


def main() -> int:
    """Boot the worker: configure, connect, verify, then loop."""
    logging.basicConfig(
        level=logging.INFO,
        format="%(asctime)s %(levelname)s %(name)s %(message)s",
    )
    settings = load_settings()

    conn = connect_database(settings)
    client = connect_queue(settings)
    try:
        verify_connections(conn, client)
        logger.info("worker ready: database and queue %s reachable", settings.queue_name)

        stop = False

        def _handle_signal(signum: int, frame: FrameType | None) -> None:
            nonlocal stop
            stop = True
            logger.info("shutdown signal received")

        signal.signal(signal.SIGINT, _handle_signal)
        signal.signal(signal.SIGTERM, _handle_signal)

        run_loop(conn, client, settings, lambda: stop)
    finally:
        client.close()
        conn.close()

    logger.info("worker stopped")
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
