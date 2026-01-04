"""Deposit model."""

from dataclasses import dataclass
from datetime import date, datetime
from decimal import Decimal
from typing import Optional
from uuid import UUID


@dataclass
class Deposit:
    """A deposit transaction."""

    id: UUID
    account_id: UUID
    amount: Decimal
    deposit_date: date
    created_at: datetime
    updated_at: datetime
    note: Optional[str] = None
