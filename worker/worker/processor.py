"""Queue message handling.

Parses one invoice job from the queue and wires it through the invoice
pipeline: compute the totals, persist the invoice, then queue the customer
notification. Deliberately free of logging of customer data (AC-29).
"""

from __future__ import annotations

import json
import logging
from dataclasses import dataclass
from typing import Any

from . import calculation, invoice_repo, outbox
from .config import Settings

logger = logging.getLogger("worker.processor")


class MessageError(ValueError):
    """Raised when a queue message is not a valid invoice job."""


@dataclass(frozen=True)
class QueueMessage:
    """The queue payload published when an order reaches ``completed``."""

    order_number: str
    order_id: int


def parse_message(raw: str | bytes) -> QueueMessage:
    """Parse a raw Valkey message into a :class:`QueueMessage`.

    Raises :class:`MessageError` for anything that is not a JSON object with a
    non-empty string ``orderNumber`` and an integer ``orderId``.
    """
    if isinstance(raw, bytes):
        try:
            raw = raw.decode("utf-8")
        except UnicodeDecodeError as exc:
            raise MessageError("queue message is not valid UTF-8") from exc

    try:
        data = json.loads(raw)
    except (json.JSONDecodeError, TypeError) as exc:
        raise MessageError("queue message is not valid JSON") from exc

    if not isinstance(data, dict):
        raise MessageError("queue message must be a JSON object")

    order_number = data.get("orderNumber")
    order_id = data.get("orderId")

    if not isinstance(order_number, str) or order_number == "":
        raise MessageError("queue message is missing a non-empty orderNumber")
    if not isinstance(order_id, int) or isinstance(order_id, bool):
        raise MessageError("queue message is missing an integer orderId")

    return QueueMessage(order_number=order_number, order_id=order_id)


def _load_order_inputs(conn: Any, order_id: int) -> tuple[float, list[dict[str, Any]]]:
    """Load the labor hours and positions of an order from PostgreSQL."""
    with conn.cursor() as cur:
        cur.execute(
            "SELECT kind, description, hours, quantity, unit_price_cents "
            "FROM order_items WHERE order_id = %s ORDER BY id",
            (order_id,),
        )
        rows = cur.fetchall()

    hours = 0.0
    items: list[dict[str, Any]] = []
    for kind, description, item_hours, quantity, unit_price_cents in rows:
        items.append(
            {
                "kind": kind,
                "description": description,
                "hours": item_hours,
                "quantity": quantity,
                "unitPriceCents": unit_price_cents,
            }
        )
        if kind == "labor" and item_hours is not None:
            hours += float(item_hours)
    return hours, items


def _load_recipient_email(conn: Any, order_id: int) -> str:
    """Look up the customer e-mail an invoice notification is addressed to."""
    with conn.cursor() as cur:
        cur.execute(
            "SELECT c.email FROM orders o JOIN customers c ON c.id = o.customer_id WHERE o.id = %s",
            (order_id,),
        )
        row = cur.fetchone()
    return row[0] if row else ""


def process_message(conn: Any, raw_message: str | bytes, settings: Settings) -> str:
    """Process one queue message from start to finish.

    Computes the invoice, persists it and queues the notification, in that
    order, and returns the created invoice number.
    """
    message = parse_message(raw_message)
    hours, items = _load_order_inputs(conn, message.order_id)

    totals = calculation.compute_invoice(hours, items, settings.hourly_rate_cents)
    invoice_number = invoice_repo.save_invoice(conn, message.order_number, totals)

    recipient_email = _load_recipient_email(conn, message.order_id)
    subject = f"Rechnung zu Auftrag {message.order_number}"
    body = (
        "Guten Tag,\n\n"
        f"zu Ihrem Auftrag {message.order_number} wurde die Rechnung "
        f"{invoice_number} erstellt.\n"
        f"Nettobetrag: {totals['netCents']} Cent\n"
        f"Mehrwertsteuer ({totals['taxRatePercent']} %): {totals['taxCents']} Cent\n"
        f"Bruttobetrag: {totals['grossCents']} Cent\n"
    )
    outbox.save_notification(conn, invoice_number, recipient_email, subject, body)

    return invoice_number
