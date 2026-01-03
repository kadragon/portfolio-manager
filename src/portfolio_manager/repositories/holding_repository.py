"""Holding repository for database operations."""

from datetime import datetime
from decimal import Decimal
from typing import Any, cast
from uuid import UUID

from supabase import Client

from portfolio_manager.models import Holding


class HoldingRepository:
    """Repository for Holding database operations."""

    def __init__(self, client: Client):
        """Initialize repository with Supabase client."""
        self.client = client

    def create(self, account_id: UUID, stock_id: UUID, quantity: Decimal) -> Holding:
        """Create a new holding."""
        response = (
            self.client.table("holdings")
            .insert(
                {
                    "account_id": str(account_id),
                    "stock_id": str(stock_id),
                    "quantity": str(quantity),
                }
            )
            .execute()
        )
        if not response.data or len(response.data) == 0:
            raise ValueError("Failed to create holding")
        data = cast(dict[str, Any], response.data[0])
        return Holding(
            id=UUID(str(data["id"])),
            account_id=UUID(str(data["account_id"])),
            stock_id=UUID(str(data["stock_id"])),
            quantity=Decimal(str(data["quantity"])),
            created_at=datetime.fromisoformat(str(data["created_at"])),
            updated_at=datetime.fromisoformat(str(data["updated_at"])),
        )

    def list_by_account(self, account_id: UUID) -> list[Holding]:
        """List holdings for a specific account."""
        response = (
            self.client.table("holdings")
            .select("*")
            .eq("account_id", str(account_id))
            .execute()
        )
        if not response.data:
            return []
        return [
            Holding(
                id=UUID(str(item["id"])),
                account_id=UUID(str(item["account_id"])),
                stock_id=UUID(str(item["stock_id"])),
                quantity=Decimal(str(item["quantity"])),
                created_at=datetime.fromisoformat(str(item["created_at"])),
                updated_at=datetime.fromisoformat(str(item["updated_at"])),
            )
            for item in cast(list[dict[str, Any]], response.data)
        ]

    def delete_by_account(self, account_id: UUID) -> None:
        """Delete holdings for a specific account."""
        self.client.table("holdings").delete().eq(
            "account_id", str(account_id)
        ).execute()

    def delete(self, holding_id: UUID) -> None:
        """Delete a holding by ID."""
        self.client.table("holdings").delete().eq("id", str(holding_id)).execute()

    def update(self, holding_id: UUID, quantity: Decimal) -> Holding:
        """Update a holding quantity by ID."""
        response = (
            self.client.table("holdings")
            .update({"quantity": str(quantity)})
            .eq("id", str(holding_id))
            .execute()
        )
        if not response.data:
            raise ValueError("Failed to update holding")
        data = cast(dict[str, Any], response.data[0])
        return Holding(
            id=UUID(str(data["id"])),
            account_id=UUID(str(data["account_id"])),
            stock_id=UUID(str(data["stock_id"])),
            quantity=Decimal(str(data["quantity"])),
            created_at=datetime.fromisoformat(str(data["created_at"])),
            updated_at=datetime.fromisoformat(str(data["updated_at"])),
        )
