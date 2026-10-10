"""Configuration for the invoice worker.

Every value the worker needs to boot is read from the environment here and
validated exactly once at startup. Missing values are reported by name, so a
missing variable never surfaces as a bare traceback.
"""

from __future__ import annotations

import os
from collections.abc import Mapping
from dataclasses import dataclass

DEFAULT_QUEUE_NAME = "workshop:invoices"

_REQUIRED_VARIABLES = ("DATABASE_URL", "VALKEY_URL", "HOURLY_RATE_CENTS")


class ConfigError(RuntimeError):
    """Raised when a required environment variable is missing or unusable."""


@dataclass(frozen=True)
class Settings:
    """Validated runtime configuration of the worker."""

    database_url: str
    valkey_url: str
    queue_name: str
    hourly_rate_cents: int


def _require(env: Mapping[str, str], name: str) -> str:
    value = env.get(name)
    if value is None or value.strip() == "":
        raise ConfigError(f"missing required environment variable {name}")
    return value.strip()


def load_settings(env: Mapping[str, str] | None = None) -> Settings:
    """Read and validate the worker configuration.

    Called once at startup. Raises :class:`ConfigError` naming the offending
    variable when a required value is absent or unusable.
    """
    source: Mapping[str, str] = os.environ if env is None else env

    database_url = _require(source, "DATABASE_URL")
    valkey_url = _require(source, "VALKEY_URL")
    queue_name = source.get("QUEUE_NAME") or DEFAULT_QUEUE_NAME

    raw_rate = _require(source, "HOURLY_RATE_CENTS")
    try:
        hourly_rate_cents = int(raw_rate)
    except ValueError as exc:
        raise ConfigError(
            f"environment variable HOURLY_RATE_CENTS must be an integer number of cents, "
            f"got {raw_rate!r}"
        ) from exc
    if hourly_rate_cents < 0:
        raise ConfigError("environment variable HOURLY_RATE_CENTS must not be negative")

    return Settings(
        database_url=database_url,
        valkey_url=valkey_url,
        queue_name=queue_name,
        hourly_rate_cents=hourly_rate_cents,
    )
