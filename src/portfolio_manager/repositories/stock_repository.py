"""Stock repository for database operations."""

from typing import Any, cast
from uuid import UUID
from datetime import datetime
from supabase import Client

from portfolio_manager.models import Stock


class StockRepository:
    """Repository for Stock database operations."""

    def __init__(self, client: Client):
        """Initialize repository with Supabase client.

        Args:
            client: Supabase client instance.
        """
        self.client = client

    def create(self, ticker: str, group_id: UUID) -> Stock:
        """Create a new stock.

        Args:
            ticker: Stock ticker symbol.
            group_id: ID of the group this stock belongs to.

        Returns:
            Created Stock instance.
        """
        response = (
            self.client.table("stocks")
            .insert({"ticker": ticker, "group_id": str(group_id)})
            .execute()
        )
        if not response.data or len(response.data) == 0:
            raise ValueError("Failed to create stock")
        data = cast(dict[str, Any], response.data[0])

        return Stock(
            id=UUID(str(data["id"])),
            ticker=str(data["ticker"]),
            group_id=UUID(str(data["group_id"])),
            created_at=datetime.fromisoformat(str(data["created_at"])),
            updated_at=datetime.fromisoformat(str(data["updated_at"])),
            exchange=str(data["exchange"]) if data.get("exchange") else None,
        )

    def list_by_group(self, group_id: UUID) -> list[Stock]:
        """List all stocks for a specific group.

        Args:
            group_id: ID of the group to list stocks for.

        Returns:
            List of Stock instances for the group.
        """
        response = (
            self.client.table("stocks")
            .select("*")
            .eq("group_id", str(group_id))
            .execute()
        )
        if not response.data:
            return []

        return [
            Stock(
                id=UUID(str(item["id"])),
                ticker=str(item["ticker"]),
                group_id=UUID(str(item["group_id"])),
                created_at=datetime.fromisoformat(str(item["created_at"])),
                updated_at=datetime.fromisoformat(str(item["updated_at"])),
                exchange=str(item["exchange"]) if item.get("exchange") else None,
            )
            for item in cast(list[dict[str, Any]], response.data)
        ]

    def list_all(self) -> list[Stock]:
        """List all stocks."""
        response = self.client.table("stocks").select("*").execute()
        if not response.data:
            return []

        return [
            Stock(
                id=UUID(str(item["id"])),
                ticker=str(item["ticker"]),
                group_id=UUID(str(item["group_id"])),
                created_at=datetime.fromisoformat(str(item["created_at"])),
                updated_at=datetime.fromisoformat(str(item["updated_at"])),
                exchange=str(item["exchange"]) if item.get("exchange") else None,
            )
            for item in cast(list[dict[str, Any]], response.data)
        ]

    def delete(self, stock_id: UUID) -> None:
        """Delete a stock by ID.

        Args:
            stock_id: ID of the stock to delete.
        """
        self.client.table("stocks").delete().eq("id", str(stock_id)).execute()

    def update(self, stock_id: UUID, ticker: str) -> Stock:
        """Update a stock ticker by ID.

        Args:
            stock_id: ID of the stock to update.
            ticker: Updated ticker.

        Returns:
            Updated Stock instance.
        """
        response = (
            self.client.table("stocks")
            .update({"ticker": ticker})
            .eq("id", str(stock_id))
            .execute()
        )
        if not response.data:
            raise ValueError("Failed to update stock")
        item = cast(dict[str, Any], response.data[0])
        return Stock(
            id=UUID(str(item["id"])),
            ticker=str(item["ticker"]),
            group_id=UUID(str(item["group_id"])),
            created_at=datetime.fromisoformat(str(item["created_at"])),
            updated_at=datetime.fromisoformat(str(item["updated_at"])),
            exchange=str(item["exchange"]) if item.get("exchange") else None,
        )

    def get_by_id(self, stock_id: UUID) -> Stock | None:
        """Get a stock by ID.

        Args:
            stock_id: ID of the stock to fetch.
        """
        response = (
            self.client.table("stocks").select("*").eq("id", str(stock_id)).execute()
        )
        if not response.data:
            return None
        item = cast(dict[str, Any], response.data[0])
        return Stock(
            id=UUID(str(item["id"])),
            ticker=str(item["ticker"]),
            group_id=UUID(str(item["group_id"])),
            created_at=datetime.fromisoformat(str(item["created_at"])),
            updated_at=datetime.fromisoformat(str(item["updated_at"])),
            exchange=str(item["exchange"]) if item.get("exchange") else None,
        )

    def get_by_ticker(self, ticker: str) -> Stock | None:
        """Get a stock by ticker."""
        response = (
            self.client.table("stocks").select("*").eq("ticker", ticker).execute()
        )
        if not response.data:
            return None
        item = cast(dict[str, Any], response.data[0])
        return Stock(
            id=UUID(str(item["id"])),
            ticker=str(item["ticker"]),
            group_id=UUID(str(item["group_id"])),
            created_at=datetime.fromisoformat(str(item["created_at"])),
            updated_at=datetime.fromisoformat(str(item["updated_at"])),
            exchange=str(item["exchange"]) if item.get("exchange") else None,
        )

    def update_group(self, stock_id: UUID, group_id: UUID) -> Stock:
        """Update a stock's group by ID."""
        response = (
            self.client.table("stocks")
            .update({"group_id": str(group_id)})
            .eq("id", str(stock_id))
            .execute()
        )
        if not response.data:
            raise ValueError("Failed to move stock")
        item = cast(dict[str, Any], response.data[0])
        return Stock(
            id=UUID(str(item["id"])),
            ticker=str(item["ticker"]),
            group_id=UUID(str(item["group_id"])),
            created_at=datetime.fromisoformat(str(item["created_at"])),
            updated_at=datetime.fromisoformat(str(item["updated_at"])),
            exchange=str(item["exchange"]) if item.get("exchange") else None,
        )

    def update_exchange(self, stock_id: UUID, exchange: str) -> Stock:
        """Update a stock's preferred exchange by ID."""
        response = (
            self.client.table("stocks")
            .update({"exchange": exchange})
            .eq("id", str(stock_id))
            .execute()
        )
        if not response.data:
            raise ValueError("Failed to update stock exchange")
        item = cast(dict[str, Any], response.data[0])
        return Stock(
            id=UUID(str(item["id"])),
            ticker=str(item["ticker"]),
            group_id=UUID(str(item["group_id"])),
            created_at=datetime.fromisoformat(str(item["created_at"])),
            updated_at=datetime.fromisoformat(str(item["updated_at"])),
            exchange=str(item["exchange"]) if item.get("exchange") else None,
        )
