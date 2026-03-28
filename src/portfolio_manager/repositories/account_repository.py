"""Account repository for database operations."""

from datetime import datetime, timezone
from decimal import Decimal
from uuid import UUID, uuid4

from portfolio_manager.models import Account
from portfolio_manager.services.database import AccountModel


class AccountRepository:
    """Repository for Account database operations."""

    def create(self, name: str, cash_balance: Decimal) -> Account:
        """Create a new account."""
        now = datetime.now(timezone.utc)
        row = AccountModel.create(
            id=uuid4(),
            name=name,
            cash_balance=cash_balance,
            created_at=now,
            updated_at=now,
        )
        return self._to_domain(row)

    def list_all(self) -> list[Account]:
        """List all accounts."""
        return [self._to_domain(row) for row in AccountModel.select()]

    def delete_with_holdings(self, account_id: UUID, holding_repository) -> None:
        """Delete an account and its holdings."""
        holding_repository.delete_by_account(account_id)
        AccountModel.delete().where(AccountModel.id == account_id).execute()

    def update(self, account_id: UUID, name: str, cash_balance: Decimal) -> Account:
        """Update an account name and cash balance."""
        now = datetime.now(timezone.utc)
        AccountModel.update(name=name, cash_balance=cash_balance, updated_at=now).where(
            AccountModel.id == account_id
        ).execute()

        row = AccountModel.get_by_id(account_id)
        return self._to_domain(row)

    @staticmethod
    def _to_domain(row: AccountModel) -> Account:
        return Account(
            id=row.id,
            name=row.name,
            cash_balance=Decimal(str(row.cash_balance)),
            created_at=row.created_at,
            updated_at=row.updated_at,
        )
