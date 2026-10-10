"""End-to-end tests of the invoice pipeline against PostgreSQL.

These tests create the schema they use in the test database and set up exactly
the rows they need; nothing is assumed to exist from an earlier run.
"""

from __future__ import annotations

import os
from datetime import UTC, datetime

import psycopg
import pytest
from worker.config import Settings
from worker.invoice_repo import derive_invoice_number, save_invoice
from worker.processor import process_message

DATABASE_URL = os.environ.get("DATABASE_URL")

pytestmark = pytest.mark.skipif(not DATABASE_URL, reason="DATABASE_URL must be set")

HOURLY_RATE_CENTS = 8900

# The schema below mirrors backend/internal/store/migrations.go exactly: the
# migration is the single source of truth for these tables. The test creates the
# same tables so it also runs stand-alone against an empty database, and the
# IF NOT EXISTS clauses make it a no-op when the API has already migrated.
SCHEMA = """
CREATE TABLE IF NOT EXISTS customers (
    id         serial PRIMARY KEY,
    name       text NOT NULL,
    email      text NOT NULL,
    phone      text NOT NULL,
    created_at timestamptz NOT NULL DEFAULT now()
);
CREATE TABLE IF NOT EXISTS vehicles (
    id               serial PRIMARY KEY,
    customer_id      integer NOT NULL REFERENCES customers(id) ON DELETE CASCADE,
    plate            text NOT NULL,
    plate_normalized text NOT NULL UNIQUE,
    brand            text NOT NULL,
    model            text NOT NULL,
    mileage_km       integer NOT NULL DEFAULT 0,
    created_at       timestamptz NOT NULL DEFAULT now()
);
CREATE TABLE IF NOT EXISTS orders (
    id                  serial PRIMARY KEY,
    order_number        text NOT NULL UNIQUE,
    vehicle_id          integer NOT NULL REFERENCES vehicles(id),
    requested_date      date NOT NULL,
    problem_description text NOT NULL,
    status              text NOT NULL DEFAULT 'requested'
        CHECK (status IN ('requested','confirmed','in_progress','completed','picked_up')),
    created_at          timestamptz NOT NULL DEFAULT now()
);
CREATE TABLE IF NOT EXISTS order_items (
    id               serial PRIMARY KEY,
    order_id         integer NOT NULL REFERENCES orders(id) ON DELETE CASCADE,
    kind             text NOT NULL CHECK (kind IN ('labor','part')),
    description      text NOT NULL,
    hours            numeric(10,2),
    quantity         integer,
    unit_price_cents integer,
    amount_cents     integer NOT NULL,
    created_at       timestamptz NOT NULL DEFAULT now()
);
CREATE TABLE IF NOT EXISTS invoices (
    id               serial PRIMARY KEY,
    invoice_number   text NOT NULL UNIQUE,
    order_id         integer NOT NULL UNIQUE REFERENCES orders(id),
    net_cents        integer NOT NULL,
    tax_rate_percent integer NOT NULL,
    tax_cents        integer NOT NULL,
    gross_cents      integer NOT NULL,
    created_at       timestamptz NOT NULL DEFAULT now()
);
CREATE TABLE IF NOT EXISTS invoice_items (
    id               serial PRIMARY KEY,
    invoice_id       integer NOT NULL REFERENCES invoices(id) ON DELETE CASCADE,
    kind             text NOT NULL,
    description      text NOT NULL,
    hours            numeric(10,2),
    quantity         integer,
    unit_price_cents integer,
    amount_cents     integer NOT NULL
);
CREATE TABLE IF NOT EXISTS outbox (
    id              serial PRIMARY KEY,
    recipient_email text NOT NULL,
    subject         text NOT NULL,
    body            text NOT NULL,
    status          text NOT NULL DEFAULT 'pending',
    created_at      timestamptz NOT NULL DEFAULT now()
);
"""


def _settings() -> Settings:
    return Settings(
        database_url=DATABASE_URL or "",
        valkey_url="redis://localhost:6379/0",
        queue_name="workshop:invoices",
        hourly_rate_cents=HOURLY_RATE_CENTS,
    )


def _seed_order(conn: psycopg.Connection, order_number: str) -> int:
    with conn.cursor() as cur:
        cur.execute(
            "INSERT INTO customers (name, email, phone) VALUES (%s, %s, %s) RETURNING id",
            ("Anna Beispiel", "anna@example.com", "0123456789"),
        )
        customer_id = cur.fetchone()[0]
        cur.execute(
            "INSERT INTO vehicles (customer_id, plate, plate_normalized, brand, model, mileage_km) "
            "VALUES (%s, %s, %s, %s, %s, %s) RETURNING id",
            (customer_id, "B-AB 1234", "BAB1234", "VW", "Golf", 120000),
        )
        vehicle_id = cur.fetchone()[0]
        cur.execute(
            "INSERT INTO orders "
            "(order_number, vehicle_id, requested_date, problem_description, status) "
            "VALUES (%s, %s, %s, %s, 'completed') RETURNING id",
            (order_number, vehicle_id, "2026-01-05", "Inspektion"),
        )
        order_id = cur.fetchone()[0]
        cur.execute(
            "INSERT INTO order_items "
            "(order_id, kind, description, hours, quantity, unit_price_cents, amount_cents) "
            "VALUES (%s, 'labor', 'Montage', 1.5, NULL, NULL, 13350)",
            (order_id,),
        )
        cur.execute(
            "INSERT INTO order_items "
            "(order_id, kind, description, hours, quantity, unit_price_cents, amount_cents) "
            "VALUES (%s, 'part', 'Filter', NULL, 2, 1999, 3998)",
            (order_id,),
        )
    conn.commit()
    return int(order_id)


@pytest.fixture()
def conn() -> psycopg.Connection:
    with psycopg.connect(DATABASE_URL) as setup:
        setup.execute(SCHEMA)
    connection = psycopg.connect(DATABASE_URL)
    try:
        with connection.cursor() as cur:
            cur.execute(
                "TRUNCATE outbox, invoice_items, invoices, order_items, orders, vehicles, customers "
                "RESTART IDENTITY CASCADE"
            )
        connection.commit()
        yield connection
    finally:
        connection.close()


@pytest.fixture(autouse=True)
def _configured_rate(monkeypatch: pytest.MonkeyPatch) -> None:
    monkeypatch.setenv("HOURLY_RATE_CENTS", str(HOURLY_RATE_CENTS))


def _count(conn: psycopg.Connection, table: str) -> int:
    with conn.cursor() as cur:
        cur.execute(f"SELECT count(*) FROM {table}")
        return int(cur.fetchone()[0])


def test_one_message_produces_one_invoice_with_totals_items_and_outbox(
    conn: psycopg.Connection,
) -> None:
    order_number = "AU-2026-0001"
    order_id = _seed_order(conn, order_number)
    message = f'{{"orderNumber":"{order_number}","orderId":{order_id}}}'

    invoice_number = process_message(conn, message, _settings())
    conn.commit()

    # Net: 1.5 h * 8900 = 13350 plus 2 * 1999 = 3998 -> 17348; tax 19 % = 3296.
    assert invoice_number == derive_invoice_number(order_number)
    with conn.cursor() as cur:
        cur.execute(
            "SELECT invoice_number, net_cents, tax_rate_percent, tax_cents, gross_cents, order_id "
            "FROM invoices"
        )
        row = cur.fetchone()
    assert row == (invoice_number, 17348, 19, 3296, 20644, order_id)

    with conn.cursor() as cur:
        cur.execute("SELECT kind, amount_cents FROM invoice_items ORDER BY kind")
        items = cur.fetchall()
    assert items == [("labor", 13350), ("part", 3998)]

    assert _count(conn, "invoices") == 1
    assert _count(conn, "outbox") == 1


def test_outbox_entry_carries_recipient_subject_text_and_utc_timestamp(
    conn: psycopg.Connection,
) -> None:
    order_number = "AU-2026-0002"
    order_id = _seed_order(conn, order_number)
    message = f'{{"orderNumber":"{order_number}","orderId":{order_id}}}'

    invoice_number = process_message(conn, message, _settings())
    conn.commit()

    with conn.cursor() as cur:
        cur.execute("SELECT recipient_email, subject, body, created_at FROM outbox")
        recipient, subject, body, created_at = cur.fetchone()

    assert recipient == "anna@example.com"
    assert order_number in subject
    assert invoice_number in body
    assert created_at.tzinfo is not None
    assert abs((datetime.now(UTC) - created_at).total_seconds()) < 60


def test_delivering_the_same_message_twice_creates_no_second_invoice(
    conn: psycopg.Connection,
) -> None:
    order_number = "AU-2026-0003"
    order_id = _seed_order(conn, order_number)
    message = f'{{"orderNumber":"{order_number}","orderId":{order_id}}}'

    first = process_message(conn, message, _settings())
    conn.commit()
    second = process_message(conn, message, _settings())
    conn.commit()

    assert first == second
    assert _count(conn, "invoices") == 1
    assert _count(conn, "invoice_items") == 2
    assert _count(conn, "outbox") == 1


def test_save_invoice_twice_returns_the_existing_invoice_number(
    conn: psycopg.Connection,
) -> None:
    order_number = "AU-2026-0004"
    _seed_order(conn, order_number)
    totals = {"netCents": 100, "taxRatePercent": 19, "taxCents": 19, "grossCents": 119}

    first = save_invoice(conn, order_number, totals)
    conn.commit()
    second = save_invoice(conn, order_number, totals)
    conn.commit()

    assert first == second
    assert _count(conn, "invoices") == 1
