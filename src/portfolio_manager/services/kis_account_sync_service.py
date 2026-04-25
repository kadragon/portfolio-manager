"""KIS account synchronization service."""

from __future__ import annotations

import json
import logging
from collections import defaultdict
from dataclasses import dataclass, field
from datetime import datetime
from decimal import Decimal
from pathlib import Path
from typing import TYPE_CHECKING, Callable
from uuid import UUID

from portfolio_manager.core.time import now_kst
from portfolio_manager.models import Account, Holding

from portfolio_manager.repositories.account_repository import AccountRepository
from portfolio_manager.repositories.group_repository import GroupRepository
from portfolio_manager.repositories.holding_repository import HoldingRepository
from portfolio_manager.repositories.stock_repository import StockRepository
from portfolio_manager.services.kis.kis_domestic_balance_client import (
    KisDomesticBalanceClient,
)
from portfolio_manager.services.stock_name_formatter import format_stock_name

if TYPE_CHECKING:
    from portfolio_manager.services.stock_service import StockService


class KisEmptySnapshotError(RuntimeError):
    """Raised when a KIS snapshot returns no holdings while the account has some.

    Guards against silently wiping existing holdings because of an unsupported
    account type, misrouted balance client (e.g. domestic client fetching an
    overseas-only account), or transient API issues. Pass
    ``allow_empty_snapshot=True`` to opt into clearing holdings when the user
    has genuinely liquidated the account.
    """


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


logger = logging.getLogger(__name__)


@dataclass
class KisAccountSyncService:
    """Synchronize an internal account with KIS balance data."""

    account_repository: AccountRepository
    holding_repository: HoldingRepository
    stock_repository: StockRepository
    group_repository: GroupRepository
    kis_balance_client: KisDomesticBalanceClient
    stock_service: StockService | None = None
    default_group_name: str = "KIS 자동동기화"
    sync_log_path: Path | None = None
    _now: Callable[[], datetime] = field(default=now_kst, repr=False)

    def sync_account(
        self,
        *,
        account: Account,
        cano: str,
        acnt_prdt_cd: str,
        kis_balance_client: KisDomesticBalanceClient | None = None,
        allow_empty_snapshot: bool = False,
    ) -> KisAccountSyncResult:
        """Sync cash balance and holdings for one account."""
        balance_client = kis_balance_client or self.kis_balance_client
        base_event = {
            "account_id": str(account.id),
            "cano": cano,
            "acnt_prdt_cd": acnt_prdt_cd,
        }
        try:
            snapshot = balance_client.fetch_account_snapshot(cano, acnt_prdt_cd)
        except Exception as exc:
            self._log_event(
                {
                    **base_event,
                    "event": "sync_snapshot_error",
                    "error_type": type(exc).__name__,
                    "error": str(exc),
                }
            )
            raise

        existing_holdings = self.holding_repository.list_by_account(account.id)
        if not snapshot.holdings and existing_holdings and not allow_empty_snapshot:
            self._log_event(
                {
                    **base_event,
                    "event": "sync_guard_empty_snapshot",
                    "existing_holding_count": len(existing_holdings),
                }
            )
            raise KisEmptySnapshotError(
                "KIS 스냅샷이 비어 있어 기존 보유 내역을 보호합니다. "
                "실제로 전량 매도된 경우 allow_empty_snapshot=True로 재실행하세요."
            )

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
                if self.stock_service is not None:
                    self.stock_service.persist_name(stock, position.name)
                else:
                    stock = self.stock_repository.update_name(
                        stock.id, format_stock_name(position.name)
                    )
                stocks_by_ticker[stock.ticker] = stock
            target_quantities_by_stock_id[stock.id] += position.quantity

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

        self._log_event(
            {
                **base_event,
                "event": "sync_success",
                "old_cash_balance": str(old_cash_balance),
                "cash_balance": str(snapshot.cash_balance),
                "holding_count": len(target_quantities_by_stock_id),
                "created_stock_count": created_stock_count,
                "allow_empty_snapshot": allow_empty_snapshot,
                "holding_changes": [
                    {
                        "ticker": change.ticker,
                        "action": change.action,
                        "old_quantity": (
                            str(change.old_quantity)
                            if change.old_quantity is not None
                            else None
                        ),
                        "new_quantity": (
                            str(change.new_quantity)
                            if change.new_quantity is not None
                            else None
                        ),
                    }
                    for change in holding_changes
                ],
            }
        )

        return KisAccountSyncResult(
            account_id=account.id,
            cash_balance=snapshot.cash_balance,
            old_cash_balance=old_cash_balance,
            holding_count=len(target_quantities_by_stock_id),
            created_stock_count=created_stock_count,
            holding_changes=tuple(holding_changes),
        )

    def _log_event(self, event: dict) -> None:
        """Append a JSONL event. Never raises — logging must not break sync."""
        payload = {"timestamp": self._now().isoformat(), **event}
        if self.sync_log_path is None:
            return
        try:
            self.sync_log_path.parent.mkdir(parents=True, exist_ok=True)
            with self.sync_log_path.open("a", encoding="utf-8") as f:
                f.write(json.dumps(payload, ensure_ascii=False) + "\n")
        except OSError as exc:
            logger.warning("Failed to write KIS sync log: %s", exc)

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
