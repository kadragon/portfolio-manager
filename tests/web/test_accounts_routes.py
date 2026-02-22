from decimal import Decimal
from uuid import uuid4


class _FailingSyncService:
    def sync_account(self, **_: object) -> None:
        raise RuntimeError("sync boom")


def test_get_account_row_returns_404_for_unknown_account(client):
    response = client.get(f"/accounts/{uuid4()}")
    assert response.status_code == 404


def test_edit_account_form_returns_404_for_unknown_account(client):
    response = client.get(f"/accounts/{uuid4()}/edit")
    assert response.status_code == 404


def test_update_account_updates_name_and_cash_balance(client, fake_container):
    response = client.put(
        f"/accounts/{fake_container.account.id}",
        data={"name": "  업데이트 계좌  ", "cash_balance": "12345.67"},
    )

    assert response.status_code == 200
    updated = fake_container.account_repository.list_all()[0]
    assert updated.name == "업데이트 계좌"
    assert updated.cash_balance == Decimal("12345.67")


def test_delete_account_removes_account_and_holdings(client, fake_container):
    response = client.delete(f"/accounts/{fake_container.account.id}")

    assert response.status_code == 200
    assert fake_container.account_repository.list_all() == []
    assert (
        fake_container.holding_repository.list_by_account(fake_container.account.id)
        == []
    )


def test_list_holdings_returns_404_for_unknown_account(client):
    response = client.get(f"/accounts/{uuid4()}/holdings")
    assert response.status_code == 404


def test_create_holding_adds_item_for_account(client, fake_container):
    response = client.post(
        f"/accounts/{fake_container.account.id}/holdings",
        data={"stock_id": str(fake_container.stock.id), "quantity": "3.5"},
    )

    assert response.status_code == 200
    holdings = fake_container.holding_repository.list_by_account(
        fake_container.account.id
    )
    assert len(holdings) == 2
    assert holdings[0].quantity == Decimal("3.5")


def test_get_holding_row_returns_404_for_unknown_holding(client, fake_container):
    response = client.get(f"/accounts/{fake_container.account.id}/holdings/{uuid4()}")
    assert response.status_code == 404


def test_edit_holding_form_returns_404_for_unknown_holding(client, fake_container):
    response = client.get(
        f"/accounts/{fake_container.account.id}/holdings/{uuid4()}/edit",
    )
    assert response.status_code == 404


def test_update_holding_updates_quantity(client, fake_container):
    response = client.put(
        f"/accounts/{fake_container.account.id}/holdings/{fake_container.holding.id}",
        data={"quantity": "7.25"},
    )

    assert response.status_code == 200
    updated = fake_container.holding_repository.list_by_account(
        fake_container.account.id
    )[0]
    assert updated.quantity == Decimal("7.25")


def test_delete_holding_removes_item(client, fake_container):
    response = client.delete(
        f"/accounts/{fake_container.account.id}/holdings/{fake_container.holding.id}",
    )

    assert response.status_code == 200
    assert (
        fake_container.holding_repository.list_by_account(fake_container.account.id)
        == []
    )


def test_create_holding_by_ticker_returns_422_for_blank_ticker(client, fake_container):
    response = client.post(
        f"/accounts/{fake_container.account.id}/holdings/by-ticker",
        data={"ticker": "   ", "quantity": "1"},
    )

    assert response.status_code == 422
    assert "티커를 입력하세요." in response.text


def test_create_holding_by_ticker_returns_422_for_invalid_group_id(
    client, fake_container
):
    response = client.post(
        f"/accounts/{fake_container.account.id}/holdings/by-ticker",
        data={"ticker": "AAPL", "quantity": "1", "group_id": "not-a-uuid"},
    )

    assert response.status_code == 422
    assert "선택한 그룹이 올바르지 않습니다." in response.text


def test_create_holding_by_ticker_returns_422_for_missing_group(client, fake_container):
    response = client.post(
        f"/accounts/{fake_container.account.id}/holdings/by-ticker",
        data={"ticker": "AAPL", "quantity": "1", "group_id": str(uuid4())},
    )

    assert response.status_code == 422
    assert "선택한 그룹을 찾을 수 없습니다." in response.text


def test_sync_account_returns_service_not_configured_message(client, fake_container):
    fake_container.kis_account_sync_service = None

    response = client.post(f"/accounts/{fake_container.account.id}/sync")

    assert response.status_code == 200
    assert "KIS 계좌 동기화 서비스가 설정되지 않았습니다." in response.text


def test_sync_account_returns_account_not_found_message(client):
    response = client.post(f"/accounts/{uuid4()}/sync")

    assert response.status_code == 200
    assert "계좌를 찾을 수 없습니다." in response.text


def test_sync_account_returns_failure_message_on_exception(client, fake_container):
    fake_container.kis_account_sync_service = _FailingSyncService()

    response = client.post(f"/accounts/{fake_container.account.id}/sync")

    assert response.status_code == 200
    assert "동기화 실패: sync boom" in response.text
