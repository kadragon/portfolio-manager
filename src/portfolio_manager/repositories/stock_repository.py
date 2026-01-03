"""Stock repository for database operations."""

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
        data = response.data[0]

        return Stock(
            id=UUID(str(data["id"])),
            ticker=str(data["ticker"]),
            group_id=UUID(str(data["group_id"])),
            created_at=datetime.fromisoformat(str(data["created_at"])),
            updated_at=datetime.fromisoformat(str(data["updated_at"])),
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
            )
            for item in response.data
        ]
