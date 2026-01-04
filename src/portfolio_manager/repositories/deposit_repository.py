"""Deposit repository."""

from datetime import date, datetime
from decimal import Decimal
from typing import Optional
from uuid import UUID

from portfolio_manager.models.deposit import Deposit


class DepositRepository:
    """Repository for managing deposits."""

    def __init__(self, supabase_client):
        """Initialize the repository."""
        self.client = supabase_client

    def create(
        self,
        account_id: UUID,
        amount: Decimal,
        deposit_date: date,
        note: Optional[str] = None,
    ) -> Deposit:
        """Create a new deposit."""
        data = {
            "account_id": str(account_id),
            "amount": str(amount),
            "deposit_date": deposit_date.isoformat(),
            "note": note,
        }

        response = self.client.table("deposits").insert(data).execute()
        item = response.data[0]

        return Deposit(
            id=UUID(item["id"]),
            account_id=UUID(item["account_id"]),
            amount=Decimal(item["amount"]),
            deposit_date=datetime.strptime(item["deposit_date"], "%Y-%m-%d").date(),
            note=item.get("note"),
            created_at=datetime.fromisoformat(item["created_at"]),
            updated_at=datetime.fromisoformat(item["updated_at"]),
        )

    def list_by_account(self, account_id: UUID) -> list[Deposit]:
        """List all deposits for an account."""
        response = (
            self.client.table("deposits")
            .select("*")
            .eq("account_id", str(account_id))
            .order("deposit_date", desc=True)
            .execute()
        )

        return [
            Deposit(
                id=UUID(item["id"]),
                account_id=UUID(item["account_id"]),
                amount=Decimal(item["amount"]),
                deposit_date=datetime.strptime(item["deposit_date"], "%Y-%m-%d").date(),
                note=item.get("note"),
                created_at=datetime.fromisoformat(item["created_at"]),
                updated_at=datetime.fromisoformat(item["updated_at"]),
            )
            for item in response.data
        ]

    def delete(self, deposit_id: UUID) -> None:
        """Delete a deposit."""
        self.client.table("deposits").delete().eq("id", str(deposit_id)).execute()

    def get_total_by_account(self, account_id: UUID) -> Decimal:
        """Get total deposit amount for an account."""
        deposits = self.list_by_account(account_id)
        return sum((d.amount for d in deposits), Decimal("0"))
