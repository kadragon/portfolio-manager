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
    rebalance_group_name: str | None = None
    sleeve_name: str | None = None
    reason: str | None = None
    trigger_type: str | None = None
    amount_krw: Decimal | None = None
    amount_local: Decimal | None = None

    def __post_init__(self) -> None:
        if self.rebalance_group_name is not None:
            self.sleeve_name = self.rebalance_group_name
        elif self.sleeve_name is not None:
            self.rebalance_group_name = self.sleeve_name
