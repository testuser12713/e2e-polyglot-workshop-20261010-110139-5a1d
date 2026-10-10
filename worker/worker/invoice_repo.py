"""Persistence of generated invoices.

Stub with the final signature: the concrete persistence is delivered by ticket
#17 ("Implement invoice generation from the queue").
"""

from __future__ import annotations

from typing import Any

from .calculation import InvoiceTotals


def save_invoice(conn: Any, order_number: str, totals: InvoiceTotals) -> str:
    """Persist the invoice for ``order_number`` and return its invoice number."""
    ...
