"""Portfolio service for aggregating holdings data."""

from dataclasses import dataclass
from decimal import Decimal

from portfolio_manager.models import Group, Stock
from portfolio_manager.repositories.group_repository import GroupRepository
from portfolio_manager.repositories.holding_repository import HoldingRepository
from portfolio_manager.repositories.stock_repository import StockRepository


@dataclass
class StockHolding:
    """Stock with aggregated quantity."""

    stock: Stock
    quantity: Decimal


@dataclass
class GroupHoldings:
    """Group with its stock holdings."""

    group: Group
    stock_holdings: list[StockHolding]


class PortfolioService:
    """Service for portfolio operations."""

    def __init__(
        self,
        group_repository: GroupRepository,
        stock_repository: StockRepository,
        holding_repository: HoldingRepository,
    ):
        """Initialize service with repositories."""
        self.group_repository = group_repository
        self.stock_repository = stock_repository
        self.holding_repository = holding_repository

    def get_holdings_by_group(self) -> list[GroupHoldings]:
        """Get holdings aggregated by group."""
        groups = self.group_repository.list_all()
        aggregated_holdings = self.holding_repository.get_aggregated_holdings_by_stock()

        result = []
        for group in groups:
            stocks = self.stock_repository.list_by_group(group.id)
            stock_holdings = []
            for stock in stocks:
                quantity = aggregated_holdings.get(stock.id, Decimal("0"))
                if quantity > 0:
                    stock_holdings.append(StockHolding(stock=stock, quantity=quantity))
            result.append(GroupHoldings(group=group, stock_holdings=stock_holdings))

        return result
