"""Outbox for invoice notifications.

Every generated invoice gets one entry in the outbox holding the recipient
e-mail, the subject and the text, stamped with the current UTC time. No real
e-mail is ever sent — the entry is the delivery.

The entry is unique per invoice: the processor deliberately calls this on every
delivery, so a repeatedly delivered message must not queue a second
notification.
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
    """Persist an outbox entry (recipient, subject, text) for an invoice.

    Runs on the same connection (and therefore the same transaction) as the
    invoice insert, so an invoice and its notification are stored together. A
    second call for the same invoice is a no-op.
    """
    with conn.cursor() as cur:
        cur.execute("SELECT id FROM invoices WHERE invoice_number = %s", (invoice_number,))
        row = cur.fetchone()
        invoice_id = int(row[0]) if row is not None else None

        cur.execute(
            "INSERT INTO outbox (invoice_id, recipient_email, subject, body, created_at) "
            "SELECT %s, %s, %s, %s, now() "
            "WHERE %s IS NULL OR NOT EXISTS (SELECT 1 FROM outbox WHERE invoice_id = %s)",
            (invoice_id, recipient_email, subject, body, invoice_id, invoice_id),
        )
