"""Holding repository for database operations."""

from collections import defaultdict
from decimal import Decimal
from uuid import UUID, uuid4

from portfolio_manager.core.time import now_kst

from portfolio_manager.models import Holding
from portfolio_manager.services.database import HoldingModel, db


class HoldingRepository:
    """Repository for Holding database operations."""

    def create(self, account_id: UUID, stock_id: UUID, quantity: Decimal) -> Holding:
        """Create a new holding."""
        now = now_kst()
        row = HoldingModel.create(
            id=uuid4(),
            account=account_id,
            stock=stock_id,
            quantity=quantity,
            created_at=now,
            updated_at=now,
        )
        return self._to_domain(row)

    def list_by_account(self, account_id: UUID) -> list[Holding]:
        """List holdings for a specific account."""
        return [
            self._to_domain(row)
            for row in HoldingModel.select().where(HoldingModel.account == account_id)
        ]

    def delete_by_account(self, account_id: UUID) -> None:
        """Delete holdings for a specific account."""
        HoldingModel.delete().where(HoldingModel.account == account_id).execute()

    def delete(self, holding_id: UUID) -> None:
        """Delete a holding by ID."""
        HoldingModel.delete().where(HoldingModel.id == holding_id).execute()

    def update(self, holding_id: UUID, quantity: Decimal) -> Holding:
        """Update a holding quantity by ID."""
        now = now_kst()
        HoldingModel.update(quantity=quantity, updated_at=now).where(
            HoldingModel.id == holding_id
        ).execute()
        row = HoldingModel.get_by_id(holding_id)
        return self._to_domain(row)

    def bulk_update_by_account(
        self, account_id: UUID, updates: list[tuple[UUID, Decimal]]
    ) -> list[Holding]:
        """Update multiple holdings for one account atomically."""
        if not updates:
            return []

        holding_ids = [hid for hid, _ in updates]
        if len(set(holding_ids)) != len(holding_ids):
            raise ValueError("duplicate holding_ids are not allowed")

        for _, qty in updates:
            if qty <= 0:
                raise ValueError("quantity must be greater than zero")

        # Verify all holdings belong to the account
        existing = {
            row.id: row
            for row in HoldingModel.select().where(
                (HoldingModel.account == account_id)
                & (HoldingModel.id.in_(holding_ids))
            )
        }
        if len(existing) != len(holding_ids):
            raise ValueError("all holdings must belong to account")

        now = now_kst()
        results = []
        with db.atomic():
            for holding_id, quantity in updates:
                HoldingModel.update(quantity=quantity, updated_at=now).where(
                    HoldingModel.id == holding_id
                ).execute()

            for holding_id, _ in updates:
                row = HoldingModel.get_by_id(holding_id)
                results.append(self._to_domain(row))

        return results

    def get_aggregated_holdings_by_stock(self) -> dict[UUID, Decimal]:
        """Get aggregated holdings by stock across all accounts."""
        rows = HoldingModel.select(HoldingModel.stock, HoldingModel.quantity)

        aggregated: dict[UUID, Decimal] = defaultdict(Decimal)
        for row in rows:
            aggregated[row.stock_id] += Decimal(str(row.quantity))

        return dict(aggregated)

    @staticmethod
    def _to_domain(row: HoldingModel) -> Holding:
        return Holding(
            id=row.id,
            account_id=row.account_id,
            stock_id=row.stock_id,
            quantity=Decimal(str(row.quantity)),
            created_at=row.created_at,
            updated_at=row.updated_at,
        )
