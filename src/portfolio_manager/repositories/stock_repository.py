"""Stock repository for database operations."""

from uuid import UUID, uuid4

from portfolio_manager.core.time import now_kst
from portfolio_manager.models import Stock
from portfolio_manager.services.database import StockModel


class StockRepository:
    """Repository for Stock database operations."""

    def create(self, ticker: str, group_id: UUID, name: str = "") -> Stock:
        """Create a new stock."""
        now = now_kst()
        row = StockModel.create(
            id=uuid4(),
            ticker=ticker,
            name=name,
            group=group_id,
            created_at=now,
            updated_at=now,
        )
        return self._to_domain(row)

    def list_by_group(self, group_id: UUID) -> list[Stock]:
        """List all stocks for a specific group."""
        return [
            self._to_domain(row)
            for row in StockModel.select().where(StockModel.group == group_id)
        ]

    def list_all(self) -> list[Stock]:
        """List all stocks."""
        return [self._to_domain(row) for row in StockModel.select()]

    def delete(self, stock_id: UUID) -> None:
        """Delete a stock by ID."""
        StockModel.delete().where(StockModel.id == stock_id).execute()

    def update(self, stock_id: UUID, ticker: str) -> Stock:
        """Update a stock ticker by ID."""
        now = now_kst()
        StockModel.update(ticker=ticker, updated_at=now).where(
            StockModel.id == stock_id
        ).execute()
        row = StockModel.get_by_id(stock_id)
        return self._to_domain(row)

    def get_by_id(self, stock_id: UUID) -> Stock | None:
        """Get a stock by ID."""
        row = StockModel.get_or_none(StockModel.id == stock_id)
        return self._to_domain(row) if row else None

    def get_by_ticker(self, ticker: str) -> Stock | None:
        """Get a stock by ticker."""
        row = StockModel.get_or_none(StockModel.ticker == ticker)
        return self._to_domain(row) if row else None

    def update_group(self, stock_id: UUID, group_id: UUID) -> Stock:
        """Update a stock's group by ID."""
        now = now_kst()
        StockModel.update(group=group_id, updated_at=now).where(
            StockModel.id == stock_id
        ).execute()
        row = StockModel.get_by_id(stock_id)
        return self._to_domain(row)

    def update_exchange(self, stock_id: UUID, exchange: str) -> Stock:
        """Update a stock's preferred exchange by ID."""
        now = now_kst()
        StockModel.update(exchange=exchange, updated_at=now).where(
            StockModel.id == stock_id
        ).execute()
        row = StockModel.get_by_id(stock_id)
        return self._to_domain(row)

    def update_name(self, stock_id: UUID, name: str) -> Stock:
        """Update a stock's display name by ID."""
        now = now_kst()
        StockModel.update(name=name, updated_at=now).where(
            StockModel.id == stock_id
        ).execute()
        row = StockModel.get_by_id(stock_id)
        return self._to_domain(row)

    @staticmethod
    def _to_domain(row: StockModel) -> Stock:
        return Stock(
            id=row.id,
            ticker=row.ticker,
            group_id=row.group_id,
            created_at=row.created_at,
            updated_at=row.updated_at,
            exchange=row.exchange,
            name=row.name,
        )
