"""Rebalance recommendation models."""

from dataclasses import dataclass
from decimal import Decimal
from enum import Enum


class RebalanceAction(Enum):
    """Action type for rebalancing."""

    BUY = "buy"
    SELL = "sell"


@dataclass
class RebalanceRecommendation:
    """Recommendation for rebalancing a stock position."""

    ticker: str
    action: RebalanceAction
    amount: Decimal
    priority: int
    currency: str | None = None
    quantity: Decimal | None = None
    stock_name: str | None = None
    group_name: str | None = None
    account_name: str | None = None
    sleeve_name: str | None = None
    reason: str | None = None
    trigger_type: str | None = None
    amount_krw: Decimal | None = None
    amount_local: Decimal | None = None
