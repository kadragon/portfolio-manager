"""Rebalance recommendation models."""

from dataclasses import dataclass
from decimal import Decimal
from enum import Enum

from portfolio_manager.models import Group


class RebalanceAction(Enum):
    """Action type for rebalancing."""

    BUY = "buy"
    SELL = "sell"


class GroupRebalanceAction(Enum):
    """Action type for group-level rebalancing."""

    NO_ACTION = "no_action"
    BUY = "buy"
    SELL_CANDIDATE = "sell_candidate"


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


@dataclass
class GroupRebalanceSignal:
    """Group-level rebalancing signal."""

    group: Group
    action: GroupRebalanceAction
    delta: Decimal = Decimal("0")  # Current weight - target percentage
    manual_review_required: bool = False
