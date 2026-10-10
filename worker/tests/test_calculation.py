"""Tests for the invoice money calculation."""

from __future__ import annotations

from worker.calculation import TAX_RATE_PERCENT, compute_invoice, item_amount_cents


def test_empty_order_is_all_zero() -> None:
    totals = compute_invoice(0.0, [], 8900)

    assert totals == {
        "netCents": 0,
        "taxRatePercent": 19,
        "taxCents": 0,
        "grossCents": 0,
    }


def test_labor_only_uses_configured_hourly_rate() -> None:
    totals = compute_invoice(1.5, [], 8900)

    assert totals["netCents"] == 13350
    assert totals["taxRatePercent"] == TAX_RATE_PERCENT == 19
    assert totals["taxCents"] == 2537
    assert totals["grossCents"] == 15887


def test_parts_only_sum_quantity_times_unit_price() -> None:
    items = [
        {"kind": "part", "description": "Filter", "quantity": 3, "unitPriceCents": 1999},
    ]
    totals = compute_invoice(0.0, items, 8900)

    assert totals["netCents"] == 5997
    assert totals["taxCents"] == 1139
    assert totals["grossCents"] == 7136


def test_labor_and_parts_add_up_with_tax() -> None:
    items = [
        {"kind": "part", "description": "Bremsscheibe", "quantity": 2, "unitPriceCents": 4999},
    ]
    totals = compute_invoice(2.5, items, 8900)

    assert totals["netCents"] == 22250 + 9998
    assert totals["taxCents"] == 6127
    assert totals["grossCents"] == 32248 + 6127


def test_tax_rounds_half_up_not_banker_rounding() -> None:
    # 150 cents net -> 28.5 cents tax, which must round up to 29.
    totals = compute_invoice(1.5, [], 100)

    assert totals["netCents"] == 150
    assert totals["taxCents"] == 29
    assert totals["grossCents"] == 179


def test_tax_below_half_cent_rounds_down() -> None:
    # 100 cents net -> 19 cents tax exactly.
    totals = compute_invoice(1.0, [], 100)

    assert totals["taxCents"] == 19


def test_labor_items_in_the_list_are_not_counted_twice() -> None:
    items = [
        {"kind": "labor", "description": "Montage", "hours": 2.0, "unitPriceCents": 1},
    ]
    totals = compute_invoice(2.0, items, 8900)

    assert totals["netCents"] == 17800


def test_item_amount_bills_labor_with_the_configured_rate() -> None:
    item = {"kind": "labor", "description": "Montage", "hours": 1.5, "unitPriceCents": 1}

    assert item_amount_cents(item, 8900) == 13350


def test_item_amount_uses_unit_price_when_no_rate_is_configured() -> None:
    item = {"kind": "labor", "description": "Montage", "hours": 1.5, "unitPriceCents": 200}

    assert item_amount_cents(item, None) == 300


def test_item_amount_for_part_is_quantity_times_unit_price() -> None:
    item = {"kind": "part", "description": "Filter", "quantity": 4, "unitPriceCents": 250}

    assert item_amount_cents(item, 8900) == 1000
