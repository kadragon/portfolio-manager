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
    group_name: str | None = None
