"""Tests for holding repository."""

from decimal import Decimal
from uuid import uuid4
from unittest.mock import Mock

import pytest
from postgrest.exceptions import APIError

from portfolio_manager.repositories.holding_repository import HoldingRepository


def test_holding_repository_creates_holding_with_decimal_quantity():
    """Should create a holding with decimal quantity."""
    holding_id = uuid4()
    account_id = uuid4()
    stock_id = uuid4()
    response = Mock()
    response.data = [
        {
            "id": str(holding_id),
            "account_id": str(account_id),
            "stock_id": str(stock_id),
            "quantity": "10.75",
            "created_at": "2026-01-03T00:00:00",
            "updated_at": "2026-01-03T00:00:00",
        }
    ]

    client = Mock()
    client.table.return_value.insert.return_value.execute.return_value = response

    repository = HoldingRepository(client)
    holding = repository.create(
        account_id=account_id, stock_id=stock_id, quantity=Decimal("10.75")
    )

    client.table.assert_called_once_with("holdings")
    client.table.return_value.insert.assert_called_once_with(
        {
            "account_id": str(account_id),
            "stock_id": str(stock_id),
            "quantity": "10.75",
        }
    )
    assert holding.quantity == Decimal("10.75")


def test_holding_repository_deletes_by_id():
    """Should delete holding by id."""
    holding_id = uuid4()
    client = Mock()
    client.table.return_value.delete.return_value.eq.return_value.execute.return_value = Mock()

    repository = HoldingRepository(client)
    repository.delete(holding_id)

    client.table.assert_called_once_with("holdings")
    client.table.return_value.delete.assert_called_once()
    client.table.return_value.delete.return_value.eq.assert_called_once_with(
        "id", str(holding_id)
    )


def test_aggregate_holdings_by_stock():
    """모든 계좌에서 동일 주식의 보유 수량을 합산한다."""
    # Given: 서버 사이드 집계 결과가 반환됨
    stock_id = uuid4()

    response = Mock()
    response.data = [
        {
            "stock_id": str(stock_id),
            "quantity": "15",
        },
    ]

    client = Mock()
    client.rpc.return_value.execute.return_value = response
    client.table.side_effect = AssertionError("table select should not be used")

    repository = HoldingRepository(client)

    # When: 주식별로 합산된 보유 수량을 조회
    aggregated = repository.get_aggregated_holdings_by_stock()

    # Then: 해당 주식의 수량이 15로 합산됨
    client.rpc.assert_called_once_with("aggregate_holdings_by_stock")
    assert stock_id in aggregated
    assert aggregated[stock_id] == Decimal("15")


def test_aggregate_holdings_by_stock_falls_back_when_rpc_missing():
    """RPC 함수가 없으면 holdings 테이블 조회로 집계한다."""
    stock_id = uuid4()
    other_stock_id = uuid4()

    rpc_missing_error = APIError(
        {
            "message": "Could not find the function public.aggregate_holdings_by_stock without parameters in the schema cache",
            "code": "PGRST202",
            "hint": None,
            "details": "Searched for the function public.aggregate_holdings_by_stock without parameters",
        }
    )

    fallback_response = Mock()
    fallback_response.data = [
        {"stock_id": str(stock_id), "quantity": "10"},
        {"stock_id": str(stock_id), "quantity": "2.5"},
        {"stock_id": str(other_stock_id), "quantity": "3"},
    ]

    client = Mock()
    client.rpc.return_value.execute.side_effect = rpc_missing_error
    client.table.return_value.select.return_value.execute.return_value = (
        fallback_response
    )

    repository = HoldingRepository(client)

    aggregated = repository.get_aggregated_holdings_by_stock()

    client.rpc.assert_called_once_with("aggregate_holdings_by_stock")
    client.table.assert_called_once_with("holdings")
    client.table.return_value.select.assert_called_once_with("stock_id,quantity")
    assert aggregated[stock_id] == Decimal("12.5")
    assert aggregated[other_stock_id] == Decimal("3")


def test_holding_repository_reads_decimal_quantity():
    """Should parse decimal quantities when listing holdings."""
    account_id = uuid4()
    response = Mock()
    response.data = [
        {
            "id": str(uuid4()),
            "account_id": str(account_id),
            "stock_id": str(uuid4()),
            "quantity": "3.1415",
            "created_at": "2026-01-03T00:00:00",
            "updated_at": "2026-01-03T00:00:00",
        }
    ]
    client = Mock()
    client.table.return_value.select.return_value.eq.return_value.execute.return_value = response

    repository = HoldingRepository(client)
    holdings = repository.list_by_account(account_id)

    assert holdings[0].quantity == Decimal("3.1415")


def test_holding_repository_create_raises_when_no_rows():
    """Should raise ValueError when create returns no rows."""
    account_id = uuid4()
    stock_id = uuid4()
    response = Mock()
    response.data = []
    client = Mock()
    client.table.return_value.insert.return_value.execute.return_value = response
    repository = HoldingRepository(client)

    with pytest.raises(ValueError, match="Failed to create holding"):
        repository.create(
            account_id=account_id, stock_id=stock_id, quantity=Decimal("1")
        )


def test_holding_repository_list_by_account_returns_empty_when_no_rows():
    """Should return empty list when account has no holdings."""
    account_id = uuid4()
    response = Mock()
    response.data = []
    client = Mock()
    client.table.return_value.select.return_value.eq.return_value.execute.return_value = response
    repository = HoldingRepository(client)

    assert repository.list_by_account(account_id) == []


def test_holding_repository_update_returns_updated_holding():
    """Should update quantity and return updated holding."""
    holding_id = uuid4()
    account_id = uuid4()
    stock_id = uuid4()
    response = Mock()
    response.data = [
        {
            "id": str(holding_id),
            "account_id": str(account_id),
            "stock_id": str(stock_id),
            "quantity": "9.5",
            "created_at": "2026-01-03T00:00:00",
            "updated_at": "2026-01-03T00:00:00",
        }
    ]
    client = Mock()
    client.table.return_value.update.return_value.eq.return_value.execute.return_value = response
    repository = HoldingRepository(client)

    holding = repository.update(holding_id, quantity=Decimal("9.5"))

    assert holding.quantity == Decimal("9.5")
    client.table.return_value.update.assert_called_once_with({"quantity": "9.5"})


def test_holding_repository_update_raises_when_no_rows():
    """Should raise ValueError when update returns no rows."""
    holding_id = uuid4()
    response = Mock()
    response.data = []
    client = Mock()
    client.table.return_value.update.return_value.eq.return_value.execute.return_value = response
    repository = HoldingRepository(client)

    with pytest.raises(ValueError, match="Failed to update holding"):
        repository.update(holding_id, quantity=Decimal("1"))


def test_bulk_update_by_account_updates_holdings_via_rpc():
    """Should call bulk update RPC and parse updated holdings."""
    account_id = uuid4()
    holding_id_a = uuid4()
    holding_id_b = uuid4()
    stock_id_a = uuid4()
    stock_id_b = uuid4()
    response = Mock()
    response.data = [
        {
            "id": str(holding_id_a),
            "account_id": str(account_id),
            "stock_id": str(stock_id_a),
            "quantity": "11.5",
            "created_at": "2026-01-03T00:00:00",
            "updated_at": "2026-01-03T00:00:01",
        },
        {
            "id": str(holding_id_b),
            "account_id": str(account_id),
            "stock_id": str(stock_id_b),
            "quantity": "3",
            "created_at": "2026-01-03T00:00:00",
            "updated_at": "2026-01-03T00:00:01",
        },
    ]
    client = Mock()
    client.rpc.return_value.execute.return_value = response
    repository = HoldingRepository(client)

    updated = repository.bulk_update_by_account(
        account_id,
        [
            (holding_id_a, Decimal("11.5")),
            (holding_id_b, Decimal("3")),
        ],
    )

    client.rpc.assert_called_once_with(
        "bulk_update_account_holdings",
        {
            "p_account_id": str(account_id),
            "p_holding_ids": [str(holding_id_a), str(holding_id_b)],
            "p_quantities": ["11.5", "3"],
        },
    )
    assert len(updated) == 2
    assert updated[0].quantity == Decimal("11.5")
    assert updated[1].quantity == Decimal("3")


def test_bulk_update_by_account_returns_empty_without_rpc_call_for_no_updates():
    """Empty update payload should return empty list."""
    account_id = uuid4()
    client = Mock()
    repository = HoldingRepository(client)

    assert repository.bulk_update_by_account(account_id, []) == []
    client.rpc.assert_not_called()


def test_bulk_update_by_account_raises_when_rpc_returns_no_rows():
    """Should raise ValueError when bulk RPC returns no rows."""
    account_id = uuid4()
    response = Mock()
    response.data = []
    client = Mock()
    client.rpc.return_value.execute.return_value = response
    repository = HoldingRepository(client)

    with pytest.raises(ValueError, match="Failed to bulk update holdings"):
        repository.bulk_update_by_account(
            account_id,
            [(uuid4(), Decimal("1"))],
        )


def test_bulk_update_by_account_propagates_api_error():
    """APIError from bulk RPC should propagate."""
    account_id = uuid4()
    rpc_error = APIError(
        {
            "message": "permission denied",
            "code": "42501",
            "hint": None,
            "details": None,
        }
    )
    client = Mock()
    client.rpc.return_value.execute.side_effect = rpc_error
    repository = HoldingRepository(client)

    with pytest.raises(APIError):
        repository.bulk_update_by_account(
            account_id,
            [(uuid4(), Decimal("1"))],
        )


def test_aggregate_holdings_by_stock_reraises_non_missing_rpc_error():
    """Non-missing RPC errors should propagate without fallback."""
    rpc_error = APIError(
        {
            "message": "permission denied",
            "code": "42501",
            "hint": None,
            "details": None,
        }
    )
    client = Mock()
    client.rpc.return_value.execute.side_effect = rpc_error
    repository = HoldingRepository(client)

    with pytest.raises(APIError):
        repository.get_aggregated_holdings_by_stock()
