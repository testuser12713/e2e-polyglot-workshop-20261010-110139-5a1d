"""Persistence of generated invoices.

An invoice is written together with a snapshot of its order positions in one
transaction. The unique constraint on ``invoices.order_id`` makes a repeatedly
delivered queue message produce no second invoice: the conflict is caught and
the already existing invoice number is returned.
"""

from __future__ import annotations

import os
from typing import Any

from .calculation import InvoiceTotals, item_amount_cents


class OrderNotFoundError(LookupError):
    """Raised when the order an invoice belongs to does not exist."""


def _hourly_rate_cents() -> int | None:
    """Read the configured hourly rate lazily, or ``None`` when it is absent."""
    raw = os.environ.get("HOURLY_RATE_CENTS")
    if raw is None or raw.strip() == "":
        return None
    try:
        return int(raw)
    except ValueError:
        return None


def derive_invoice_number(order_number: str) -> str:
    """Derive the invoice number deterministically from the order number.

    Order numbers are unique, so the derived invoice number is unique too and a
    redelivered message maps to the same invoice.
    """
    suffix = order_number[3:] if order_number.startswith("AU-") else order_number
    return f"RE-{suffix}"


def _find_order_id(cur: Any, order_number: str) -> int:
    cur.execute("SELECT id FROM orders WHERE order_number = %s", (order_number,))
    row = cur.fetchone()
    if row is None:
        raise OrderNotFoundError(f"no order with order number {order_number!r}")
    return int(row[0])


def _snapshot_items(cur: Any, invoice_id: int, order_id: int) -> None:
    """Copy the order positions into ``invoice_items`` with their amounts."""
    cur.execute(
        "SELECT kind, description, hours, quantity, unit_price_cents "
        "FROM order_items WHERE order_id = %s ORDER BY id",
        (order_id,),
    )
    rows = cur.fetchall()
    hourly_rate_cents = _hourly_rate_cents()

    for kind, description, hours, quantity, unit_price_cents in rows:
        amount_cents = item_amount_cents(
            {
                "kind": kind,
                "hours": hours,
                "quantity": quantity,
                "unitPriceCents": unit_price_cents,
            },
            hourly_rate_cents,
        )
        cur.execute(
            "INSERT INTO invoice_items "
            "(invoice_id, kind, description, hours, quantity, unit_price_cents, amount_cents) "
            "VALUES (%s, %s, %s, %s, %s, %s, %s)",
            (invoice_id, kind, description, hours, quantity, unit_price_cents, amount_cents),
        )


def save_invoice(conn: Any, order_number: str, totals: InvoiceTotals) -> str:
    """Persist the invoice for ``order_number`` and return its invoice number.

    The insert, its positions and the unique link to the order happen in one
    transaction. If an invoice already exists for the order, no second one is
    created and the existing invoice number is returned.
    """
    with conn.cursor() as cur:
        order_id = _find_order_id(cur, order_number)

        cur.execute("SELECT invoice_number FROM invoices WHERE order_id = %s", (order_id,))
        existing = cur.fetchone()
        if existing is not None:
            return str(existing[0])

        cur.execute(
            "INSERT INTO invoices "
            "(invoice_number, order_id, net_cents, tax_rate_percent, tax_cents, gross_cents) "
            "VALUES (%s, %s, %s, %s, %s, %s) "
            "ON CONFLICT (order_id) DO NOTHING "
            "RETURNING id, invoice_number",
            (
                derive_invoice_number(order_number),
                order_id,
                totals["netCents"],
                totals["taxRatePercent"],
                totals["taxCents"],
                totals["grossCents"],
            ),
        )
        inserted = cur.fetchone()
        if inserted is None:
            # A concurrent delivery won the race: return its invoice number.
            cur.execute("SELECT invoice_number FROM invoices WHERE order_id = %s", (order_id,))
            raced = cur.fetchone()
            if raced is None:
                raise OrderNotFoundError(
                    f"invoice for order {order_number!r} vanished during insertion"
                )
            return str(raced[0])

        invoice_id, invoice_number = int(inserted[0]), str(inserted[1])
        _snapshot_items(cur, invoice_id, order_id)
        return invoice_number
