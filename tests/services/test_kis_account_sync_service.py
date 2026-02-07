"""Tests for KIS account synchronization service."""

from datetime import datetime
from decimal import Decimal
from uuid import uuid4
from unittest.mock import Mock

import pytest

from portfolio_manager.models import Account, Group, Holding, Stock
from portfolio_manager.services.kis.kis_domestic_balance_client import (
    KisAccountSnapshot,
    KisHoldingPosition,
)
from portfolio_manager.services.kis_account_sync_service import KisAccountSyncService


def _make_holding(account: Account, stock: Stock, quantity: str) -> Holding:
    now = datetime.now()
    return Holding(
        id=uuid4(),
        account_id=account.id,
        stock_id=stock.id,
        quantity=Decimal(quantity),
        created_at=now,
        updated_at=now,
    )


def test_sync_account_updates_cash_and_applies_diff_to_holdings():
    now = datetime.now()
    account = Account(
        id=uuid4(),
        name="한국투자증권",
        cash_balance=Decimal("0"),
        created_at=now,
        updated_at=now,
    )
    existing_group = Group(
        id=uuid4(),
        name="국내",
        target_percentage=0,
        created_at=now,
        updated_at=now,
    )
    existing_stock = Stock(
        id=uuid4(),
        ticker="005930",
        group_id=existing_group.id,
        created_at=now,
        updated_at=now,
    )
    removed_stock = Stock(
        id=uuid4(),
        ticker="035720",
        group_id=existing_group.id,
        created_at=now,
        updated_at=now,
    )
    sync_group = Group(
        id=uuid4(),
        name="KIS 자동동기화",
        target_percentage=0,
        created_at=now,
        updated_at=now,
    )
    created_stock = Stock(
        id=uuid4(),
        ticker="000660",
        group_id=sync_group.id,
        created_at=now,
        updated_at=now,
    )
    existing_holding = _make_holding(account, existing_stock, "5")
    removed_holding = _make_holding(account, removed_stock, "2")

    balance_client = Mock()
    balance_client.fetch_account_snapshot.return_value = KisAccountSnapshot(
        cash_balance=Decimal("750000"),
        holdings=[
            KisHoldingPosition(ticker="005930", quantity=Decimal("10")),
            KisHoldingPosition(ticker="000660", quantity=Decimal("3")),
        ],
    )
    account_repository = Mock()
    holding_repository = Mock()
    holding_repository.list_by_account.return_value = [
        existing_holding,
        removed_holding,
    ]
    stock_repository = Mock()
    stock_repository.list_all.return_value = [existing_stock, removed_stock]
    stock_repository.create.return_value = created_stock
    group_repository = Mock()
    group_repository.list_all.return_value = []
    group_repository.create.return_value = sync_group

    service = KisAccountSyncService(
        account_repository=account_repository,
        holding_repository=holding_repository,
        stock_repository=stock_repository,
        group_repository=group_repository,
        kis_balance_client=balance_client,
    )

    result = service.sync_account(account=account, cano="12345678", acnt_prdt_cd="01")

    balance_client.fetch_account_snapshot.assert_called_once_with("12345678", "01")
    holding_repository.list_by_account.assert_called_once_with(account.id)
    holding_repository.update.assert_called_once_with(
        existing_holding.id, Decimal("10")
    )
    holding_repository.create.assert_called_once_with(
        account_id=account.id,
        stock_id=created_stock.id,
        quantity=Decimal("3"),
    )
    holding_repository.delete.assert_called_once_with(removed_holding.id)
    holding_repository.delete_by_account.assert_not_called()
    group_repository.create.assert_called_once_with("KIS 자동동기화")
    account_repository.update.assert_called_once_with(
        account.id,
        name="한국투자증권",
        cash_balance=Decimal("750000"),
    )
    assert result.cash_balance == Decimal("750000")
    assert result.holding_count == 2
    assert result.created_stock_count == 1


def test_sync_account_clears_holdings_when_kis_has_no_positions():
    now = datetime.now()
    account = Account(
        id=uuid4(),
        name="한국투자증권",
        cash_balance=Decimal("0"),
        created_at=now,
        updated_at=now,
    )
    existing_group = Group(
        id=uuid4(),
        name="국내",
        target_percentage=0,
        created_at=now,
        updated_at=now,
    )
    existing_stock = Stock(
        id=uuid4(),
        ticker="005930",
        group_id=existing_group.id,
        created_at=now,
        updated_at=now,
    )
    existing_holding = _make_holding(account, existing_stock, "1")

    balance_client = Mock()
    balance_client.fetch_account_snapshot.return_value = KisAccountSnapshot(
        cash_balance=Decimal("100000"),
        holdings=[],
    )

    account_repository = Mock()
    holding_repository = Mock()
    holding_repository.list_by_account.return_value = [existing_holding]
    stock_repository = Mock()
    stock_repository.list_all.return_value = [existing_stock]
    group_repository = Mock()

    service = KisAccountSyncService(
        account_repository=account_repository,
        holding_repository=holding_repository,
        stock_repository=stock_repository,
        group_repository=group_repository,
        kis_balance_client=balance_client,
    )

    result = service.sync_account(account=account, cano="12345678", acnt_prdt_cd="01")

    account_repository.update.assert_called_once_with(
        account.id,
        name="한국투자증권",
        cash_balance=Decimal("100000"),
    )
    holding_repository.list_by_account.assert_called_once_with(account.id)
    holding_repository.delete.assert_called_once_with(existing_holding.id)
    holding_repository.create.assert_not_called()
    stock_repository.create.assert_not_called()
    group_repository.create.assert_not_called()
    holding_repository.delete_by_account.assert_not_called()
    assert result.holding_count == 0
    assert result.created_stock_count == 0


def test_sync_account_does_not_wipe_holdings_before_creates():
    now = datetime.now()
    account = Account(
        id=uuid4(),
        name="한국투자증권",
        cash_balance=Decimal("0"),
        created_at=now,
        updated_at=now,
    )
    existing_group = Group(
        id=uuid4(),
        name="국내",
        target_percentage=0,
        created_at=now,
        updated_at=now,
    )
    existing_stock = Stock(
        id=uuid4(),
        ticker="005930",
        group_id=existing_group.id,
        created_at=now,
        updated_at=now,
    )
    existing_holding = _make_holding(account, existing_stock, "5")

    sync_group = Group(
        id=uuid4(),
        name="KIS 자동동기화",
        target_percentage=0,
        created_at=now,
        updated_at=now,
    )
    missing_stock = Stock(
        id=uuid4(),
        ticker="000660",
        group_id=sync_group.id,
        created_at=now,
        updated_at=now,
    )

    balance_client = Mock()
    balance_client.fetch_account_snapshot.return_value = KisAccountSnapshot(
        cash_balance=Decimal("200000"),
        holdings=[
            KisHoldingPosition(ticker="005930", quantity=Decimal("5")),
            KisHoldingPosition(ticker="000660", quantity=Decimal("3")),
        ],
    )
    account_repository = Mock()
    holding_repository = Mock()
    holding_repository.list_by_account.return_value = [existing_holding]
    holding_repository.create.side_effect = RuntimeError("temporary db failure")
    stock_repository = Mock()
    stock_repository.list_all.return_value = [existing_stock]
    stock_repository.create.return_value = missing_stock
    group_repository = Mock()
    group_repository.list_all.return_value = []
    group_repository.create.return_value = sync_group

    service = KisAccountSyncService(
        account_repository=account_repository,
        holding_repository=holding_repository,
        stock_repository=stock_repository,
        group_repository=group_repository,
        kis_balance_client=balance_client,
    )

    with pytest.raises(RuntimeError):
        service.sync_account(account=account, cano="12345678", acnt_prdt_cd="01")

    holding_repository.delete_by_account.assert_not_called()
    holding_repository.delete.assert_not_called()
    account_repository.update.assert_not_called()
