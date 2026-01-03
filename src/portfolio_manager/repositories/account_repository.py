"""Account repository for database operations."""

from datetime import datetime
from decimal import Decimal
from typing import Any, cast
from uuid import UUID

from supabase import Client

from portfolio_manager.models import Account


class AccountRepository:
    """Repository for Account database operations."""

    def __init__(self, client: Client):
        """Initialize repository with Supabase client."""
        self.client = client

    def create(self, name: str, cash_balance: Decimal) -> Account:
        """Create a new account."""
        response = (
            self.client.table("accounts")
            .insert({"name": name, "cash_balance": str(cash_balance)})
            .execute()
        )
        if not response.data or len(response.data) == 0:
            raise ValueError("Failed to create account")
        data = cast(dict[str, Any], response.data[0])
        return Account(
            id=UUID(str(data["id"])),
            name=str(data["name"]),
            cash_balance=Decimal(str(data["cash_balance"])),
            created_at=datetime.fromisoformat(str(data["created_at"])),
            updated_at=datetime.fromisoformat(str(data["updated_at"])),
        )

    def list_all(self) -> list[Account]:
        """List all accounts."""
        response = self.client.table("accounts").select("*").execute()
        if not response.data:
            return []
        return [
            Account(
                id=UUID(str(item["id"])),
                name=str(item["name"]),
                cash_balance=Decimal(str(item["cash_balance"])),
                created_at=datetime.fromisoformat(str(item["created_at"])),
                updated_at=datetime.fromisoformat(str(item["updated_at"])),
            )
            for item in cast(list[dict[str, Any]], response.data)
        ]

    def delete_with_holdings(self, account_id: UUID, holding_repository) -> None:
        """Delete an account and its holdings."""
        holding_repository.delete_by_account(account_id)
        self.client.table("accounts").delete().eq("id", str(account_id)).execute()
