"""KIS account synchronization service."""

from __future__ import annotations

from dataclasses import dataclass
from decimal import Decimal
from uuid import UUID

from portfolio_manager.models import Account
from portfolio_manager.repositories.account_repository import AccountRepository
from portfolio_manager.repositories.group_repository import GroupRepository
from portfolio_manager.repositories.holding_repository import HoldingRepository
from portfolio_manager.repositories.stock_repository import StockRepository
from portfolio_manager.services.kis.kis_domestic_balance_client import (
    KisDomesticBalanceClient,
)


@dataclass(frozen=True)
class KisAccountSyncResult:
    """Result summary of KIS account synchronization."""

    account_id: UUID
    cash_balance: Decimal
    holding_count: int
    created_stock_count: int


@dataclass
class KisAccountSyncService:
    """Synchronize an internal account with KIS balance data."""

    account_repository: AccountRepository
    holding_repository: HoldingRepository
    stock_repository: StockRepository
    group_repository: GroupRepository
    kis_balance_client: KisDomesticBalanceClient
    default_group_name: str = "KIS 자동동기화"

    def sync_account(
        self,
        *,
        account: Account,
        cano: str,
        acnt_prdt_cd: str,
    ) -> KisAccountSyncResult:
        """Sync cash balance and holdings for one account."""
        snapshot = self.kis_balance_client.fetch_account_snapshot(cano, acnt_prdt_cd)

        self.account_repository.update(
            account.id, name=account.name, cash_balance=snapshot.cash_balance
        )

        self.holding_repository.delete_by_account(account.id)

        stocks_by_ticker = {
            stock.ticker: stock for stock in self.stock_repository.list_all()
        }
        sync_group_id: UUID | None = None
        created_stock_count = 0
        holding_count = 0

        for position in snapshot.holdings:
            stock = stocks_by_ticker.get(position.ticker)
            if stock is None:
                if sync_group_id is None:
                    sync_group_id = self._get_or_create_sync_group_id()
                stock = self.stock_repository.create(position.ticker, sync_group_id)
                stocks_by_ticker[stock.ticker] = stock
                created_stock_count += 1

            self.holding_repository.create(
                account_id=account.id,
                stock_id=stock.id,
                quantity=position.quantity,
            )
            holding_count += 1

        return KisAccountSyncResult(
            account_id=account.id,
            cash_balance=snapshot.cash_balance,
            holding_count=holding_count,
            created_stock_count=created_stock_count,
        )

    def _get_or_create_sync_group_id(self) -> UUID:
        groups = self.group_repository.list_all()
        for group in groups:
            if group.name == self.default_group_name:
                return group.id
        created = self.group_repository.create(self.default_group_name)
        return created.id
