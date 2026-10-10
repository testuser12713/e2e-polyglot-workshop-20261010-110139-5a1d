"""Invoice calculation.

Stub with the final signature: the concrete money logic is delivered by ticket
#17 ("Implement invoice generation from the queue").
"""

from __future__ import annotations

from typing import Any, TypedDict


class InvoiceTotals(TypedDict):
    """Money totals of an invoice, every amount an integer number of cents."""

    netCents: int
    taxRatePercent: int
    taxCents: int
    grossCents: int


def compute_invoice(
    hours: float,
    items: list[dict[str, Any]],
    hourly_rate_cents: int,
) -> InvoiceTotals:
    """Compute the invoice totals for one order."""
    ...
