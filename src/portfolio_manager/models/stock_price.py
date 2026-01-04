"""Stock price cache model."""

from dataclasses import dataclass
from datetime import date, datetime
from decimal import Decimal
from uuid import UUID


@dataclass
class StockPrice:
    """Cached stock price for a given date."""

    id: UUID
    ticker: str
    price: Decimal
    currency: str
    name: str
    exchange: str | None
    price_date: date
    created_at: datetime
    updated_at: datetime
