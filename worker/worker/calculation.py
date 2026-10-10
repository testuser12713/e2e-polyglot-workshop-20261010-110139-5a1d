"""Invoice calculation.

Money is handled as integer cents throughout. Amounts are rounded commercially
(round half up) at the end of each computation, so a half cent always rounds up
to the next cent regardless of Python's default banker's rounding.
"""

from __future__ import annotations

from decimal import ROUND_HALF_UP, Decimal
from typing import Any, TypedDict

TAX_RATE_PERCENT = 19


class InvoiceTotals(TypedDict):
    """Money totals of an invoice, every amount an integer number of cents."""

    netCents: int
    taxRatePercent: int
    taxCents: int
    grossCents: int


def round_cents(value: float | int | Decimal) -> int:
    """Round a cents value commercially (half up) to a whole number of cents."""
    if not isinstance(value, Decimal):
        value = Decimal(str(value))
    return int(value.quantize(Decimal("1"), rounding=ROUND_HALF_UP))


def item_amount_cents(item: dict[str, Any], hourly_rate_cents: int | None) -> int:
    """Compute the invoice amount of a single order position in whole cents.

    Labor is billed as ``hours * hourly rate`` (the configured rate, never the
    stored unit price); parts are ``quantity * unit price``.
    """
    if item.get("kind") == "labor":
        hours = item.get("hours")
        if hours is None:
            return 0
        rate = hourly_rate_cents
        if rate is None:
            rate = item.get("unitPriceCents") or 0
        return round_cents(Decimal(str(hours)) * Decimal(int(rate)))

    quantity = item.get("quantity") or 0
    unit_price = item.get("unitPriceCents") or 0
    return int(quantity) * int(unit_price)


def compute_invoice(
    hours: float,
    items: list[dict[str, Any]],
    hourly_rate_cents: int,
) -> InvoiceTotals:
    """Compute the invoice totals for one order.

    ``net`` is the labor time (``hours``) at the configured ``hourly_rate_cents``
    plus every part position; ``tax`` is 19 % of the net amount as its own
    amount and ``gross`` is net plus tax. All amounts are whole cents.
    """
    labor_cents = round_cents(Decimal(str(hours)) * Decimal(int(hourly_rate_cents)))

    parts_cents = 0
    for item in items:
        if item.get("kind") == "part":
            parts_cents += int(item.get("quantity") or 0) * int(item.get("unitPriceCents") or 0)

    net_cents = labor_cents + parts_cents
    tax_cents = round_cents(Decimal(net_cents) * Decimal(TAX_RATE_PERCENT) / Decimal(100))

    return InvoiceTotals(
        netCents=net_cents,
        taxRatePercent=TAX_RATE_PERCENT,
        taxCents=tax_cents,
        grossCents=net_cents + tax_cents,
    )
