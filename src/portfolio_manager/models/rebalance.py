"""Rebalance recommendation models."""

from dataclasses import dataclass, field
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


@dataclass
class AccountRebalanceSummary:
    """Per-account rebalance summary for UI grouping and cash tracking."""

    account_id: str
    account_name: str
    starting_cash_krw: Decimal
    sell_cash_krw: Decimal
    total_buy_krw: Decimal
    unused_cash_krw: Decimal
    unmet_groups: list[str]
    sell_recommendations: list[RebalanceRecommendation] = field(default_factory=list)
    buy_recommendations: list[RebalanceRecommendation] = field(default_factory=list)
