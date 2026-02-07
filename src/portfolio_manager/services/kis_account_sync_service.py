"""KIS account synchronization service."""

from __future__ import annotations

from collections import defaultdict
from dataclasses import dataclass
from decimal import Decimal
from uuid import UUID

from portfolio_manager.models import Account, Holding
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
        stocks_by_ticker = {
            stock.ticker: stock for stock in self.stock_repository.list_all()
        }
        sync_group_id: UUID | None = None
        created_stock_count = 0

        target_quantities_by_stock_id: dict[UUID, Decimal] = defaultdict(Decimal)
        for position in snapshot.holdings:
            stock = stocks_by_ticker.get(position.ticker)
            if stock is None:
                if sync_group_id is None:
                    sync_group_id = self._get_or_create_sync_group_id()
                stock = self.stock_repository.create(position.ticker, sync_group_id)
                stocks_by_ticker[stock.ticker] = stock
                created_stock_count += 1
            target_quantities_by_stock_id[stock.id] += position.quantity

        existing_holdings = self.holding_repository.list_by_account(account.id)
        existing_by_stock_id: dict[UUID, list[Holding]] = defaultdict(list)
        for holding in existing_holdings:
            existing_by_stock_id[holding.stock_id].append(holding)

        for stock_id, target_quantity in target_quantities_by_stock_id.items():
            existing_for_stock = existing_by_stock_id.get(stock_id, [])
            if not existing_for_stock:
                self.holding_repository.create(
                    account_id=account.id,
                    stock_id=stock_id,
                    quantity=target_quantity,
                )
                continue

            primary_holding = existing_for_stock[0]
            if primary_holding.quantity != target_quantity:
                self.holding_repository.update(primary_holding.id, target_quantity)

            for duplicate_holding in existing_for_stock[1:]:
                self.holding_repository.delete(duplicate_holding.id)

        for stock_id, existing_for_stock in existing_by_stock_id.items():
            if stock_id in target_quantities_by_stock_id:
                continue
            for holding in existing_for_stock:
                self.holding_repository.delete(holding.id)

        self.account_repository.update(
            account.id, name=account.name, cash_balance=snapshot.cash_balance
        )

        return KisAccountSyncResult(
            account_id=account.id,
            cash_balance=snapshot.cash_balance,
            holding_count=len(target_quantities_by_stock_id),
            created_stock_count=created_stock_count,
        )

    def _get_or_create_sync_group_id(self) -> UUID:
        groups = self.group_repository.list_all()
        for group in groups:
            if group.name == self.default_group_name:
                return group.id
        created = self.group_repository.create(self.default_group_name)
        return created.id
