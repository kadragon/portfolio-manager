"""Tests for KIS account synchronization service."""

import json
from datetime import datetime
from decimal import Decimal
from pathlib import Path
from uuid import uuid4
from unittest.mock import Mock

import pytest

from portfolio_manager.models import Account, Group, Holding, Stock
from portfolio_manager.services.kis.kis_domestic_balance_client import (
    KisAccountSnapshot,
    KisHoldingPosition,
)
from portfolio_manager.services.kis_account_sync_service import (
    KisAccountSyncService,
    KisEmptySnapshotError,
    _MAX_SYNC_LOG_BYTES,
)


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

    result = service.sync_account(
        account=account,
        cano="12345678",
        acnt_prdt_cd="01",
        allow_empty_snapshot=True,
    )

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


def test_sync_account_refuses_to_wipe_when_snapshot_empty_by_default():
    now = datetime.now()
    account = Account(
        id=uuid4(),
        name="한국투자증권",
        cash_balance=Decimal("500000"),
        created_at=now,
        updated_at=now,
    )
    group = Group(
        id=uuid4(),
        name="국내",
        target_percentage=0,
        created_at=now,
        updated_at=now,
    )
    stock = Stock(
        id=uuid4(),
        ticker="005930",
        group_id=group.id,
        created_at=now,
        updated_at=now,
    )
    existing_holding = _make_holding(account, stock, "7")

    balance_client = Mock()
    balance_client.fetch_account_snapshot.return_value = KisAccountSnapshot(
        cash_balance=Decimal("0"),
        holdings=[],
    )
    account_repository = Mock()
    holding_repository = Mock()
    holding_repository.list_by_account.return_value = [existing_holding]
    stock_repository = Mock()
    stock_repository.list_all.return_value = [stock]
    group_repository = Mock()

    service = KisAccountSyncService(
        account_repository=account_repository,
        holding_repository=holding_repository,
        stock_repository=stock_repository,
        group_repository=group_repository,
        kis_balance_client=balance_client,
    )

    with pytest.raises(KisEmptySnapshotError):
        service.sync_account(account=account, cano="12345678", acnt_prdt_cd="01")

    # No mutation should have occurred.
    holding_repository.delete.assert_not_called()
    holding_repository.update.assert_not_called()
    holding_repository.create.assert_not_called()
    account_repository.update.assert_not_called()


def test_sync_account_allows_empty_snapshot_when_no_existing_holdings():
    now = datetime.now()
    account = Account(
        id=uuid4(),
        name="신규계좌",
        cash_balance=Decimal("0"),
        created_at=now,
        updated_at=now,
    )

    balance_client = Mock()
    balance_client.fetch_account_snapshot.return_value = KisAccountSnapshot(
        cash_balance=Decimal("50000"),
        holdings=[],
    )
    account_repository = Mock()
    holding_repository = Mock()
    holding_repository.list_by_account.return_value = []
    stock_repository = Mock()
    stock_repository.list_all.return_value = []
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
        account.id, name="신규계좌", cash_balance=Decimal("50000")
    )
    holding_repository.delete.assert_not_called()
    assert result.holding_count == 0


def test_sync_account_back_fills_name_on_existing_stock_with_no_name():
    now = datetime.now()
    account = Account(
        id=uuid4(),
        name="한국투자증권",
        cash_balance=Decimal("0"),
        created_at=now,
        updated_at=now,
    )
    group = Group(
        id=uuid4(),
        name="국내",
        target_percentage=0,
        created_at=now,
        updated_at=now,
    )
    stock = Stock(
        id=uuid4(),
        ticker="005930",
        group_id=group.id,
        created_at=now,
        updated_at=now,
    )
    holding = _make_holding(account, stock, "5")
    updated_stock = Stock(
        id=stock.id,
        ticker=stock.ticker,
        group_id=stock.group_id,
        name="삼성전자",
        created_at=now,
        updated_at=now,
    )

    balance_client = Mock()
    balance_client.fetch_account_snapshot.return_value = KisAccountSnapshot(
        cash_balance=Decimal("100000"),
        holdings=[
            KisHoldingPosition(ticker="005930", quantity=Decimal("5"), name="삼성전자")
        ],
    )
    account_repository = Mock()
    holding_repository = Mock()
    holding_repository.list_by_account.return_value = [holding]
    stock_repository = Mock()
    stock_repository.list_all.return_value = [stock]
    stock_repository.update_name.return_value = updated_stock
    group_repository = Mock()

    service = KisAccountSyncService(
        account_repository=account_repository,
        holding_repository=holding_repository,
        stock_repository=stock_repository,
        group_repository=group_repository,
        kis_balance_client=balance_client,
    )

    service.sync_account(account=account, cano="12345678", acnt_prdt_cd="01")

    stock_repository.update_name.assert_called_once_with(stock.id, "삼성전자")


def test_sync_account_delegates_name_persist_to_stock_service():
    now = datetime.now()
    account = Account(
        id=uuid4(),
        name="한국투자증권",
        cash_balance=Decimal("0"),
        created_at=now,
        updated_at=now,
    )
    group = Group(
        id=uuid4(),
        name="국내",
        target_percentage=0,
        created_at=now,
        updated_at=now,
    )
    stock = Stock(
        id=uuid4(),
        ticker="005930",
        group_id=group.id,
        created_at=now,
        updated_at=now,
    )
    holding = _make_holding(account, stock, "5")

    balance_client = Mock()
    balance_client.fetch_account_snapshot.return_value = KisAccountSnapshot(
        cash_balance=Decimal("100000"),
        holdings=[
            KisHoldingPosition(ticker="005930", quantity=Decimal("5"), name="삼성전자")
        ],
    )
    account_repository = Mock()
    holding_repository = Mock()
    holding_repository.list_by_account.return_value = [holding]
    stock_repository = Mock()
    stock_repository.list_all.return_value = [stock]
    group_repository = Mock()
    stock_service = Mock()

    service = KisAccountSyncService(
        account_repository=account_repository,
        holding_repository=holding_repository,
        stock_repository=stock_repository,
        group_repository=group_repository,
        kis_balance_client=balance_client,
        stock_service=stock_service,
    )

    service.sync_account(account=account, cano="12345678", acnt_prdt_cd="01")

    stock_service.persist_name.assert_called_once_with(stock, "삼성전자")
    stock_repository.update_name.assert_not_called()


def test_sync_account_does_not_overwrite_existing_stock_name():
    now = datetime.now()
    account = Account(
        id=uuid4(),
        name="한국투자증권",
        cash_balance=Decimal("0"),
        created_at=now,
        updated_at=now,
    )
    group = Group(
        id=uuid4(),
        name="국내",
        target_percentage=0,
        created_at=now,
        updated_at=now,
    )
    stock = Stock(
        id=uuid4(),
        ticker="005930",
        group_id=group.id,
        name="삼성전자(수동입력)",
        created_at=now,
        updated_at=now,
    )
    holding = _make_holding(account, stock, "5")

    balance_client = Mock()
    balance_client.fetch_account_snapshot.return_value = KisAccountSnapshot(
        cash_balance=Decimal("100000"),
        holdings=[
            KisHoldingPosition(ticker="005930", quantity=Decimal("5"), name="삼성전자")
        ],
    )
    account_repository = Mock()
    holding_repository = Mock()
    holding_repository.list_by_account.return_value = [holding]
    stock_repository = Mock()
    stock_repository.list_all.return_value = [stock]
    group_repository = Mock()

    service = KisAccountSyncService(
        account_repository=account_repository,
        holding_repository=holding_repository,
        stock_repository=stock_repository,
        group_repository=group_repository,
        kis_balance_client=balance_client,
    )

    service.sync_account(account=account, cano="12345678", acnt_prdt_cd="01")

    stock_repository.update_name.assert_not_called()


def test_sync_account_passes_formatted_name_on_stock_create():
    now = datetime.now()
    account = Account(
        id=uuid4(),
        name="한국투자증권",
        cash_balance=Decimal("0"),
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
        ticker="069500",
        group_id=sync_group.id,
        name="KODEX 200",
        created_at=now,
        updated_at=now,
    )

    balance_client = Mock()
    balance_client.fetch_account_snapshot.return_value = KisAccountSnapshot(
        cash_balance=Decimal("0"),
        holdings=[
            KisHoldingPosition(
                ticker="069500",
                quantity=Decimal("1"),
                name="KODEX 200증권상장지수투자신탁(주식)",
            )
        ],
    )
    account_repository = Mock()
    holding_repository = Mock()
    holding_repository.list_by_account.return_value = []
    stock_repository = Mock()
    stock_repository.list_all.return_value = []
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

    service.sync_account(account=account, cano="12345678", acnt_prdt_cd="01")

    stock_repository.create.assert_called_once_with(
        "069500", sync_group.id, name="KODEX 200"
    )


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


def _make_service(balance_client, log_path=None):
    return KisAccountSyncService(
        account_repository=Mock(),
        holding_repository=Mock(),
        stock_repository=Mock(),
        group_repository=Mock(),
        kis_balance_client=balance_client,
        sync_log_path=log_path,
    )


def test_sync_writes_success_event_to_log(tmp_path: Path):
    now = datetime.now()
    account = Account(
        id=uuid4(),
        name="TOSS",
        cash_balance=Decimal("0"),
        created_at=now,
        updated_at=now,
    )
    log_path = tmp_path / "kis_sync.log"

    balance_client = Mock()
    balance_client.fetch_account_snapshot.return_value = KisAccountSnapshot(
        cash_balance=Decimal("1000"),
        holdings=[],
    )

    service = _make_service(balance_client, log_path=log_path)
    service.holding_repository.list_by_account.return_value = []
    service.stock_repository.list_all.return_value = []

    service.sync_account(account=account, cano="12345678", acnt_prdt_cd="01")

    assert log_path.exists()
    lines = log_path.read_text(encoding="utf-8").strip().splitlines()
    assert len(lines) == 1
    event = json.loads(lines[0])
    assert event["event"] == "sync_success"
    assert event["account_id"] == str(account.id)
    assert event["cano"] == "12345678"
    assert event["holding_count"] == 0
    assert "timestamp" in event


def test_sync_writes_guard_event_when_snapshot_empty(tmp_path: Path):
    now = datetime.now()
    account = Account(
        id=uuid4(),
        name="TOSS",
        cash_balance=Decimal("0"),
        created_at=now,
        updated_at=now,
    )
    stock = Stock(
        id=uuid4(),
        ticker="005930",
        group_id=uuid4(),
        created_at=now,
        updated_at=now,
    )
    existing = _make_holding(account, stock, "3")
    log_path = tmp_path / "kis_sync.log"

    balance_client = Mock()
    balance_client.fetch_account_snapshot.return_value = KisAccountSnapshot(
        cash_balance=Decimal("0"),
        holdings=[],
    )

    service = _make_service(balance_client, log_path=log_path)
    service.holding_repository.list_by_account.return_value = [existing]

    with pytest.raises(KisEmptySnapshotError):
        service.sync_account(account=account, cano="12345678", acnt_prdt_cd="01")

    event = json.loads(log_path.read_text(encoding="utf-8").strip())
    assert event["event"] == "sync_guard_empty_snapshot"
    assert event["existing_holding_count"] == 1


def test_sync_writes_snapshot_error_event(tmp_path: Path):
    now = datetime.now()
    account = Account(
        id=uuid4(),
        name="TOSS",
        cash_balance=Decimal("0"),
        created_at=now,
        updated_at=now,
    )
    log_path = tmp_path / "kis_sync.log"

    balance_client = Mock()
    balance_client.fetch_account_snapshot.side_effect = RuntimeError("boom")

    service = _make_service(balance_client, log_path=log_path)

    with pytest.raises(RuntimeError):
        service.sync_account(account=account, cano="12345678", acnt_prdt_cd="01")

    event = json.loads(log_path.read_text(encoding="utf-8").strip())
    assert event["event"] == "sync_snapshot_error"
    assert event["error_type"] == "RuntimeError"
    assert event["error"] == "boom"


def test_log_event_rotates_when_file_exceeds_size_limit(tmp_path: Path):
    log_path = tmp_path / "kis_sync.log"
    log_path.write_bytes(b"\x00" * _MAX_SYNC_LOG_BYTES)

    balance_client = Mock()
    balance_client.fetch_account_snapshot.return_value = KisAccountSnapshot(
        cash_balance=Decimal("0"),
        holdings=[],
    )
    service = _make_service(balance_client, log_path=log_path)
    service._log_event({"event": "test"})

    backup = tmp_path / "kis_sync.log.1"
    assert backup.exists()
    assert backup.stat().st_size == _MAX_SYNC_LOG_BYTES
    assert log_path.exists()
    new_content = log_path.read_text(encoding="utf-8").strip()
    assert '"event": "test"' in new_content
