"""Account model."""

from dataclasses import dataclass
from datetime import datetime
from decimal import Decimal
from uuid import UUID


@dataclass
class Account:
    """A brokerage account."""

    id: UUID
    name: str
    cash_balance: Decimal
    created_at: datetime
    updated_at: datetime
    kis_account_no: str | None = None
    kis_api_key_id: int | None = None
