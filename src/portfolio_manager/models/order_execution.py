"""Order execution record model."""

from dataclasses import dataclass
from datetime import datetime
from typing import Optional
from uuid import UUID


@dataclass
class OrderExecutionRecord:
    """A persisted order execution record."""

    id: UUID
    ticker: str
    side: str
    quantity: int
    currency: str
    status: str
    message: str
    created_at: datetime
    exchange: Optional[str] = None
    raw_response: Optional[dict] = None
