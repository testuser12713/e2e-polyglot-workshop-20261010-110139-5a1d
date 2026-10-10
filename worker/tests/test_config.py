"""Tests for the worker configuration validation."""

from __future__ import annotations

import pytest
from worker.config import DEFAULT_QUEUE_NAME, ConfigError, load_settings

VALID_ENV = {
    "DATABASE_URL": "postgresql://app:secret@127.0.0.1:5432/app",
    "VALKEY_URL": "redis://127.0.0.1:6379/0",
    "QUEUE_NAME": "workshop:invoices",
    "HOURLY_RATE_CENTS": "8900",
}


def test_load_settings_reads_and_validates_values() -> None:
    settings = load_settings(VALID_ENV)

    assert settings.database_url == VALID_ENV["DATABASE_URL"]
    assert settings.valkey_url == VALID_ENV["VALKEY_URL"]
    assert settings.queue_name == "workshop:invoices"
    assert settings.hourly_rate_cents == 8900


@pytest.mark.parametrize("missing", ["DATABASE_URL", "VALKEY_URL", "HOURLY_RATE_CENTS"])
def test_missing_required_variable_is_named(missing: str) -> None:
    env = dict(VALID_ENV)
    del env[missing]

    with pytest.raises(ConfigError) as excinfo:
        load_settings(env)

    assert missing in str(excinfo.value)


@pytest.mark.parametrize("name", ["DATABASE_URL", "VALKEY_URL", "HOURLY_RATE_CENTS"])
def test_blank_required_variable_is_named(name: str) -> None:
    env = dict(VALID_ENV)
    env[name] = "   "

    with pytest.raises(ConfigError) as excinfo:
        load_settings(env)

    assert name in str(excinfo.value)


def test_queue_name_falls_back_to_default() -> None:
    env = dict(VALID_ENV)
    del env["QUEUE_NAME"]

    assert load_settings(env).queue_name == DEFAULT_QUEUE_NAME


def test_queue_name_blank_falls_back_to_default() -> None:
    env = dict(VALID_ENV)
    env["QUEUE_NAME"] = ""

    assert load_settings(env).queue_name == DEFAULT_QUEUE_NAME


@pytest.mark.parametrize("value", ["abc", "89.0", "", "8.9", "many"])
def test_non_integer_rate_is_rejected_naming_the_variable(value: str) -> None:
    env = dict(VALID_ENV)
    env["HOURLY_RATE_CENTS"] = value

    with pytest.raises(ConfigError) as excinfo:
        load_settings(env)

    assert "HOURLY_RATE_CENTS" in str(excinfo.value)


def test_negative_rate_is_rejected() -> None:
    env = dict(VALID_ENV)
    env["HOURLY_RATE_CENTS"] = "-1"

    with pytest.raises(ConfigError) as excinfo:
        load_settings(env)

    assert "HOURLY_RATE_CENTS" in str(excinfo.value)
