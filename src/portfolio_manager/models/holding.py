"""Holding model."""

from dataclasses import dataclass
from datetime import datetime
from decimal import Decimal
from uuid import UUID


@dataclass
class Holding:
    """A holding of a stock in an account."""

    id: UUID
    account_id: UUID
    stock_id: UUID
    quantity: Decimal
    created_at: datetime
    updated_at: datetime
