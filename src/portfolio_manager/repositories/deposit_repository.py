"""Deposit repository."""

from datetime import date
from decimal import Decimal
from typing import Optional
from uuid import UUID, uuid4

from peewee import fn

from portfolio_manager.core.time import now_kst

from portfolio_manager.models.deposit import Deposit
from portfolio_manager.services.database import DepositModel


class _UnsetNote:
    pass


_UNSET_NOTE = _UnsetNote()


class DepositRepository:
    """Repository for managing deposits."""

    def create(
        self,
        amount: Decimal,
        deposit_date: date,
        note: Optional[str] = None,
    ) -> Deposit:
        """Create a new deposit."""
        now = now_kst()
        row = DepositModel.create(
            id=uuid4(),
            amount=amount,
            deposit_date=deposit_date,
            note=note,
            created_at=now,
            updated_at=now,
        )
        return self._to_domain(row)

    def update(
        self,
        deposit_id: UUID,
        amount: Decimal | None = None,
        deposit_date: date | None = None,
        note: Optional[str] | _UnsetNote = _UNSET_NOTE,
    ) -> Deposit:
        """Update an existing deposit."""
        updates: dict = {}
        if amount is not None:
            updates["amount"] = amount
        if deposit_date is not None:
            updates["deposit_date"] = deposit_date
        if not isinstance(note, _UnsetNote):
            updates["note"] = note

        updates["updated_at"] = now_kst()
        DepositModel.update(updates).where(DepositModel.id == deposit_id).execute()

        row = DepositModel.get_by_id(deposit_id)
        return self._to_domain(row)

    def list_all(self) -> list[Deposit]:
        """List all deposits."""
        return [
            self._to_domain(row)
            for row in DepositModel.select().order_by(DepositModel.deposit_date.desc())
        ]

    def get_by_date(self, deposit_date: date) -> Deposit | None:
        """Get a deposit by date."""
        row = DepositModel.get_or_none(DepositModel.deposit_date == deposit_date)
        return self._to_domain(row) if row else None

    def delete(self, deposit_id: UUID) -> None:
        """Delete a deposit."""
        DepositModel.delete().where(DepositModel.id == deposit_id).execute()

    def get_total(self) -> Decimal:
        """Get total deposit amount."""
        result = DepositModel.select(fn.SUM(DepositModel.amount)).scalar()
        return Decimal(str(result)) if result else Decimal("0")

    def get_first_deposit_date(self) -> date | None:
        """Get the earliest deposit date."""
        row = (
            DepositModel.select(DepositModel.deposit_date)
            .order_by(DepositModel.deposit_date.asc())
            .limit(1)
            .first()
        )

        if not row:
            return None

        return row.deposit_date

    @staticmethod
    def _to_domain(row: DepositModel) -> Deposit:
        return Deposit(
            id=row.id,
            amount=Decimal(str(row.amount)),
            deposit_date=row.deposit_date,
            note=row.note,
            created_at=row.created_at,
            updated_at=row.updated_at,
        )
