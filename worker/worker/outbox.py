"""Outbox for invoice notifications.

Stub with the final signature: the concrete persistence is delivered by ticket
#17 ("Implement invoice generation from the queue"). No real e-mail is ever sent.
"""

from __future__ import annotations

from typing import Any


def save_notification(
    conn: Any,
    invoice_number: str,
    recipient_email: str,
    subject: str,
    body: str,
) -> None:
    """Persist an outbox entry (recipient, subject, text) for an invoice."""
    ...
