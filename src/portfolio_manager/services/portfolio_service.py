"""Portfolio service for aggregating holdings data."""

from dataclasses import dataclass
from decimal import Decimal
from typing import TYPE_CHECKING

from portfolio_manager.models import Group, Stock
from portfolio_manager.repositories.group_repository import GroupRepository
from portfolio_manager.repositories.holding_repository import HoldingRepository
from portfolio_manager.repositories.stock_repository import StockRepository

if TYPE_CHECKING:
    from portfolio_manager.services.exchange.exchange_rate_service import (
        ExchangeRateService,
    )
    from portfolio_manager.services.price_service import PriceService


@dataclass
class StockHolding:
    """Stock with aggregated quantity."""

    stock: Stock
    quantity: Decimal


@dataclass
class StockHoldingWithPrice:
    """Stock with aggregated quantity and price information."""

    stock: Stock
    quantity: Decimal
    price: Decimal
    currency: str
    name: str
    value_krw: Decimal | None = None

    @property
    def value(self) -> Decimal:
        """Calculate total value (quantity × price)."""
        return self.quantity * self.price


@dataclass
class GroupHoldings:
    """Group with its stock holdings."""

    group: Group
    stock_holdings: list[StockHolding]


@dataclass
class PortfolioSummary:
    """Portfolio summary with price information."""

    holdings: list[tuple[Group, StockHoldingWithPrice]]
    total_value: Decimal


class PortfolioService:
    """Service for portfolio operations."""

    def __init__(
        self,
        group_repository: GroupRepository,
        stock_repository: StockRepository,
        holding_repository: HoldingRepository,
        price_service: "PriceService | None" = None,
        exchange_rate_service: "ExchangeRateService | None" = None,
    ):
        """Initialize service with repositories."""
        self.group_repository = group_repository
        self.stock_repository = stock_repository
        self.holding_repository = holding_repository
        self.price_service = price_service
        self.exchange_rate_service = exchange_rate_service

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

    def get_portfolio_summary(self) -> PortfolioSummary:
        """Get portfolio summary with valuations."""
        if self.price_service is None:
            raise ValueError("Price service is required for portfolio summary")

        groups = self.group_repository.list_all()
        aggregated_holdings = self.holding_repository.get_aggregated_holdings_by_stock()

        holdings = []
        total_value = Decimal("0")
        usd_krw_rate: Decimal | None = None

        for group in groups:
            stocks = self.stock_repository.list_by_group(group.id)
            for stock in stocks:
                quantity = aggregated_holdings.get(stock.id, Decimal("0"))
                if quantity > 0:
                    price, currency, name = self.price_service.get_stock_price(
                        stock.ticker
                    )
                    value_krw: Decimal | None = None
                    holding_value = quantity * price
                    if currency == "USD":
                        if self.exchange_rate_service is None:
                            raise ValueError(
                                "Exchange rate service is required for USD"
                            )
                        if usd_krw_rate is None:
                            usd_krw_rate = self.exchange_rate_service.get_usd_krw_rate()
                        value_krw = holding_value * usd_krw_rate
                    else:
                        value_krw = holding_value
                    holding_with_price = StockHoldingWithPrice(
                        stock=stock,
                        quantity=quantity,
                        price=price,
                        currency=currency,
                        name=name,
                        value_krw=value_krw,
                    )
                    holdings.append((group, holding_with_price))
                    if value_krw is not None:
                        total_value += value_krw

        return PortfolioSummary(holdings=holdings, total_value=total_value)
