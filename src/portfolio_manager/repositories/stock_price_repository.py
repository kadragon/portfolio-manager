"""Stock price repository for database operations."""

from __future__ import annotations

from datetime import date, datetime, timezone
from decimal import Decimal
from uuid import uuid4

from peewee import IntegrityError

from portfolio_manager.models.stock_price import StockPrice
from portfolio_manager.services.database import StockPriceModel


class StockPriceRepository:
    """Repository for cached stock prices."""

    def get_by_ticker_and_date(
        self, ticker: str, price_date: date
    ) -> StockPrice | None:
        """Fetch cached price for a ticker and date."""
        row = StockPriceModel.get_or_none(
            (StockPriceModel.ticker == ticker)
            & (StockPriceModel.price_date == price_date)
        )
        return self._to_domain(row) if row else None

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
        now = datetime.now(timezone.utc)
        try:
            row = StockPriceModel.create(
                id=uuid4(),
                ticker=ticker,
                price_date=price_date,
                price=price,
                currency=currency,
                name=name,
                exchange=exchange,
                created_at=now,
                updated_at=now,
            )
        except IntegrityError:
            existing = StockPriceModel.get(
                (StockPriceModel.ticker == ticker)
                & (StockPriceModel.price_date == price_date)
            )
            preserved_name = name if name else existing.name
            StockPriceModel.update(
                price=price,
                currency=currency,
                name=preserved_name,
                exchange=exchange,
                updated_at=now,
            ).where(
                (StockPriceModel.ticker == ticker)
                & (StockPriceModel.price_date == price_date)
            ).execute()
            row = StockPriceModel.get(
                (StockPriceModel.ticker == ticker)
                & (StockPriceModel.price_date == price_date)
            )
        return self._to_domain(row)

    @staticmethod
    def _to_domain(row: StockPriceModel) -> StockPrice:
        return StockPrice(
            id=row.id,
            ticker=row.ticker,
            price=Decimal(str(row.price)),
            currency=row.currency,
            name=row.name,
            exchange=row.exchange,
            price_date=row.price_date,
            created_at=row.created_at,
            updated_at=row.updated_at,
        )
