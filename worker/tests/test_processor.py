"""Tests for queue message parsing in the worker processor."""

from __future__ import annotations

import pytest
from worker.processor import MessageError, parse_message


def test_parse_message_reads_order_number_and_id() -> None:
    message = parse_message('{"orderNumber":"AU-2026-0001","orderId":42}')

    assert message.order_number == "AU-2026-0001"
    assert message.order_id == 42


def test_parse_message_accepts_utf8_bytes() -> None:
    message = parse_message(b'{"orderNumber":"AU-1","orderId":7}')

    assert message.order_number == "AU-1"
    assert message.order_id == 7


def test_parse_message_ignores_extra_fields() -> None:
    message = parse_message('{"orderNumber":"AU-1","orderId":1,"extra":"ignored"}')

    assert message.order_number == "AU-1"
    assert message.order_id == 1


@pytest.mark.parametrize(
    "payload",
    [
        "not json",
        "{",
        "[]",
        '"a string"',
        "null",
        "42",
        "{}",
        '{"orderId":1}',
        '{"orderNumber":"AU-1"}',
        '{"orderNumber":"","orderId":1}',
        '{"orderNumber":123,"orderId":1}',
        '{"orderNumber":"AU-1","orderId":"1"}',
        '{"orderNumber":"AU-1","orderId":true}',
        '{"orderNumber":"AU-1","orderId":1.5}',
    ],
)
def test_parse_message_rejects_malformed_payloads(payload: str) -> None:
    with pytest.raises(MessageError):
        parse_message(payload)


def test_parse_message_rejects_invalid_utf8_bytes() -> None:
    with pytest.raises(MessageError):
        parse_message(b"\xff\xfe\x00")
