"""Stock price repository for database operations."""

from __future__ import annotations

from datetime import date, datetime
from decimal import Decimal
from typing import Any, cast
from uuid import UUID

from supabase import Client

from portfolio_manager.models.stock_price import StockPrice


class StockPriceRepository:
    """Repository for cached stock prices."""

    def __init__(self, client: Client):
        """Initialize repository with Supabase client."""
        self.client = client

    def get_by_ticker_and_date(
        self, ticker: str, price_date: date
    ) -> StockPrice | None:
        """Fetch cached price for a ticker and date."""
        response = (
            self.client.table("stock_prices")
            .select("*")
            .eq("ticker", ticker)
            .eq("price_date", price_date.isoformat())
            .execute()
        )
        if not response.data:
            return None
        item = cast(dict[str, Any], response.data[0])
        return self._to_model(item)

    def save(
        self,
        *,
        ticker: str,
        price_date: date,
        price: Decimal,
        currency: str,
        name: str,
        exchange: str | None,
    ) -> StockPrice:
        """Upsert cached price for a ticker and date."""
        payload = {
            "ticker": ticker,
            "price_date": price_date.isoformat(),
            "price": str(price),
            "currency": currency,
            "name": name,
            "exchange": exchange,
        }
        response = (
            self.client.table("stock_prices")
            .upsert(payload, on_conflict="ticker,price_date")
            .execute()
        )
        if not response.data:
            raise ValueError("Failed to save stock price")
        item = cast(dict[str, Any], response.data[0])
        return self._to_model(item)

    @staticmethod
    def _to_model(item: dict[str, Any]) -> StockPrice:
        exchange = item.get("exchange")
        return StockPrice(
            id=UUID(str(item["id"])),
            ticker=str(item["ticker"]),
            price=Decimal(str(item["price"])),
            currency=str(item["currency"]),
            name=str(item["name"]),
            exchange=str(exchange) if exchange is not None else None,
            price_date=date.fromisoformat(str(item["price_date"])),
            created_at=datetime.fromisoformat(str(item["created_at"])),
            updated_at=datetime.fromisoformat(str(item["updated_at"])),
        )
