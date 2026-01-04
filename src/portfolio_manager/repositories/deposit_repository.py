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
        amount: Decimal,
        deposit_date: date,
        note: Optional[str] = None,
    ) -> Deposit:
        """Create a new deposit."""
        data = {
            "amount": str(amount),
            "deposit_date": deposit_date.isoformat(),
            "note": note,
        }

        response = self.client.table("deposits").insert(data).execute()
        item = response.data[0]

        return self._to_domain(item)

    def update(
        self,
        deposit_id: UUID,
        amount: Decimal | None = None,
        deposit_date: date | None = None,
        note: Optional[str] = None,
    ) -> Deposit:
        """Update an existing deposit."""
        data = {}
        if amount is not None:
            data["amount"] = str(amount)
        if deposit_date is not None:
            data["deposit_date"] = deposit_date.isoformat()
        if note is not None:
            data["note"] = note

        response = (
            self.client.table("deposits")
            .update(data)
            .eq("id", str(deposit_id))
            .execute()
        )
        item = response.data[0]
        return self._to_domain(item)

    def list_all(self) -> list[Deposit]:
        """List all deposits."""
        response = (
            self.client.table("deposits")
            .select("*")
            .order("deposit_date", desc=True)
            .execute()
        )

        return [self._to_domain(item) for item in response.data]

    def get_by_date(self, deposit_date: date) -> Deposit | None:
        """Get a deposit by date."""
        response = (
            self.client.table("deposits")
            .select("*")
            .eq("deposit_date", deposit_date.isoformat())
            .execute()
        )

        if not response.data:
            return None

        return self._to_domain(response.data[0])

    def delete(self, deposit_id: UUID) -> None:
        """Delete a deposit."""
        self.client.table("deposits").delete().eq("id", str(deposit_id)).execute()

    def get_total(self) -> Decimal:
        """Get total deposit amount."""
        deposits = self.list_all()
        return sum((d.amount for d in deposits), Decimal("0"))

    def _to_domain(self, item: dict) -> Deposit:
        return Deposit(
            id=UUID(item["id"]),
            amount=Decimal(item["amount"]),
            deposit_date=datetime.strptime(item["deposit_date"], "%Y-%m-%d").date(),
            note=item.get("note"),
            created_at=datetime.fromisoformat(item["created_at"]),
            updated_at=datetime.fromisoformat(item["updated_at"]),
        )
