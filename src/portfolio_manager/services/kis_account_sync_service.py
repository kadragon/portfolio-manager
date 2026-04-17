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
from portfolio_manager.services.stock_name_utils import format_stock_name


@dataclass(frozen=True)
class HoldingSyncDetail:
    """Detail of a single holding change during sync."""

    ticker: str
    action: str  # "created", "updated", "deleted"
    old_quantity: Decimal | None = None
    new_quantity: Decimal | None = None


@dataclass(frozen=True)
class KisAccountSyncResult:
    """Result summary of KIS account synchronization."""

    account_id: UUID
    cash_balance: Decimal
    old_cash_balance: Decimal
    holding_count: int
    created_stock_count: int
    holding_changes: tuple[HoldingSyncDetail, ...]


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
        kis_balance_client: KisDomesticBalanceClient | None = None,
    ) -> KisAccountSyncResult:
        """Sync cash balance and holdings for one account."""
        balance_client = kis_balance_client or self.kis_balance_client
        snapshot = balance_client.fetch_account_snapshot(cano, acnt_prdt_cd)
        stocks_by_ticker = {
            stock.ticker: stock for stock in self.stock_repository.list_all()
        }
        stock_id_to_ticker: dict[UUID, str] = {
            stock.id: stock.ticker for stock in stocks_by_ticker.values()
        }
        sync_group_id: UUID | None = None
        created_stock_count = 0
        holding_changes: list[HoldingSyncDetail] = []

        target_quantities_by_stock_id: dict[UUID, Decimal] = defaultdict(Decimal)
        for position in snapshot.holdings:
            stock = stocks_by_ticker.get(position.ticker)
            if stock is None:
                if sync_group_id is None:
                    sync_group_id = self._get_or_create_sync_group_id()
                stock = self.stock_repository.create(
                    position.ticker,
                    sync_group_id,
                    name=format_stock_name(position.name),
                )
                stocks_by_ticker[stock.ticker] = stock
                stock_id_to_ticker[stock.id] = stock.ticker
                created_stock_count += 1
            elif not stock.name and position.name:
                stock = self.stock_repository.update_name(
                    stock.id, format_stock_name(position.name)
                )
                stocks_by_ticker[stock.ticker] = stock
            target_quantities_by_stock_id[stock.id] += position.quantity

        existing_holdings = self.holding_repository.list_by_account(account.id)
        existing_by_stock_id: dict[UUID, list[Holding]] = defaultdict(list)
        for holding in existing_holdings:
            existing_by_stock_id[holding.stock_id].append(holding)

        for stock_id, target_quantity in target_quantities_by_stock_id.items():
            ticker = stock_id_to_ticker.get(stock_id, "?")
            existing_for_stock = existing_by_stock_id.get(stock_id, [])
            if not existing_for_stock:
                self.holding_repository.create(
                    account_id=account.id,
                    stock_id=stock_id,
                    quantity=target_quantity,
                )
                holding_changes.append(
                    HoldingSyncDetail(
                        ticker=ticker,
                        action="created",
                        new_quantity=target_quantity,
                    )
                )
                continue

            primary_holding = existing_for_stock[0]
            if primary_holding.quantity != target_quantity:
                holding_changes.append(
                    HoldingSyncDetail(
                        ticker=ticker,
                        action="updated",
                        old_quantity=primary_holding.quantity,
                        new_quantity=target_quantity,
                    )
                )
                self.holding_repository.update(primary_holding.id, target_quantity)

            for duplicate_holding in existing_for_stock[1:]:
                self.holding_repository.delete(duplicate_holding.id)

        for stock_id, existing_for_stock in existing_by_stock_id.items():
            if stock_id in target_quantities_by_stock_id:
                continue
            ticker = stock_id_to_ticker.get(stock_id, "?")
            for holding in existing_for_stock:
                holding_changes.append(
                    HoldingSyncDetail(
                        ticker=ticker,
                        action="deleted",
                        old_quantity=holding.quantity,
                    )
                )
                self.holding_repository.delete(holding.id)

        old_cash_balance = account.cash_balance
        self.account_repository.update(
            account.id, name=account.name, cash_balance=snapshot.cash_balance
        )

        return KisAccountSyncResult(
            account_id=account.id,
            cash_balance=snapshot.cash_balance,
            old_cash_balance=old_cash_balance,
            holding_count=len(target_quantities_by_stock_id),
            created_stock_count=created_stock_count,
            holding_changes=tuple(holding_changes),
        )

    def validate_account(
        self,
        *,
        cano: str,
        acnt_prdt_cd: str,
        kis_balance_client: KisDomesticBalanceClient | None = None,
    ) -> None:
        """Validate KIS account credentials by fetching its snapshot."""
        balance_client = kis_balance_client or self.kis_balance_client
        balance_client.fetch_account_snapshot(cano, acnt_prdt_cd)

    def _get_or_create_sync_group_id(self) -> UUID:
        groups = self.group_repository.list_all()
        for group in groups:
            if group.name == self.default_group_name:
                return group.id
        created = self.group_repository.create(self.default_group_name)
        return created.id
