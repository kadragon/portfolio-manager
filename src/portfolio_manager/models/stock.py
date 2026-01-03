"""Stock model."""

from dataclasses import dataclass
from datetime import datetime
from uuid import UUID


@dataclass
class Stock:
    """A stock ticker in a group."""

    id: UUID
    ticker: str
    group_id: UUID
    created_at: datetime
    updated_at: datetime
