"""Service for resolving and persisting stock display names."""

from __future__ import annotations

import logging
from typing import TYPE_CHECKING

from portfolio_manager.services.stock_name_formatter import format_stock_name

if TYPE_CHECKING:
    from portfolio_manager.models import Stock
    from portfolio_manager.repositories.stock_repository import StockRepository
    from portfolio_manager.services.price_service import PriceService

logger = logging.getLogger(__name__)


class StockService:
    def __init__(
        self,
        stock_repository: StockRepository,
        price_service: PriceService | None = None,
    ):
        self._stock_repository = stock_repository
        self._price_service = price_service

    def resolve_and_persist_name(self, stock: Stock) -> str:
        """Return formatted display name, persisting it when newly resolved.

        Returns the formatted existing name if already set.
        Calls price_service to look up the name when stock.name is empty.
        Returns "" if no name can be resolved or price_service is unavailable.
        Mutates stock.name in-place when a new name is persisted.
        """
        if stock.name:
            return format_stock_name(stock.name)

        if self._price_service is None:
            return ""

        resolved_name = ""
        try:
            _, _, resolved_name, _ = self._price_service.get_stock_price(
                stock.ticker, preferred_exchange=stock.exchange
            )
        except Exception:
            logger.warning(
                "price_service.get_stock_price failed for %s",
                stock.ticker,
                exc_info=True,
            )
            return ""

        if not resolved_name:
            return ""

        formatted = format_stock_name(resolved_name)
        if not formatted:
            return ""

        self._stock_repository.update_name(stock.id, formatted)
        stock.name = formatted
        return formatted
