from decimal import Decimal
from uuid import uuid4

from portfolio_manager.services.kis.kis_api_error import KisApiBusinessError
from portfolio_manager.services.kis_account_sync_service import KisEmptySnapshotError


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


def test_update_account_validates_new_kis_account_no(client, fake_container):
    response = client.put(
        f"/accounts/{fake_container.account.id}",
        data={
            "name": "업데이트 계좌",
            "cash_balance": "12345.67",
            "kis_account_no": "46592856-01",
        },
    )

    assert response.status_code == 200
    updated = fake_container.account_repository.list_all()[0]
    assert updated.kis_account_no == "46592856-01"
    assert fake_container.kis_account_sync_service.validated_accounts == [
        ("46592856", "01")
    ]


def test_update_account_rejects_invalid_kis_account_format(client, fake_container):
    before = fake_container.account_repository.list_all()[0]

    response = client.put(
        f"/accounts/{fake_container.account.id}",
        data={
            "name": "업데이트 계좌",
            "cash_balance": "12345.67",
            "kis_account_no": "123",
        },
    )

    assert response.status_code == 422
    assert "KIS 계좌번호 형식이 올바르지 않습니다" in response.text
    after = fake_container.account_repository.list_all()[0]
    assert after.kis_account_no == before.kis_account_no
    assert fake_container.kis_account_sync_service.validated_accounts == []


def test_update_account_rejects_kis_business_error(client, fake_container):
    fake_container.kis_account_sync_service.validate_exception = KisApiBusinessError(
        code="OPSQ2000",
        message="ERROR : INPUT INVALID_CHECK_ACNO",
    )

    response = client.put(
        f"/accounts/{fake_container.account.id}",
        data={
            "name": "업데이트 계좌",
            "cash_balance": "12345.67",
            "kis_account_no": "46592856-01",
        },
    )

    assert response.status_code == 422
    assert (
        "KIS 계좌 검증 실패: OPSQ2000 - ERROR : INPUT INVALID_CHECK_ACNO"
        in response.text
    )
    updated = fake_container.account_repository.list_all()[0]
    assert updated.kis_account_no == "12345678-01"


def test_update_account_skips_validation_for_same_kis_account_digits(
    client, fake_container
):
    response = client.put(
        f"/accounts/{fake_container.account.id}",
        data={
            "name": "업데이트 계좌",
            "cash_balance": "12345.67",
            "kis_account_no": "1234567801",
        },
    )

    assert response.status_code == 200
    updated = fake_container.account_repository.list_all()[0]
    assert updated.kis_account_no == "1234567801"
    assert fake_container.kis_account_sync_service.validated_accounts == []


def test_update_account_rejects_kis_account_change_without_sync_service(
    client, fake_container
):
    fake_container.kis_account_sync_service = None

    response = client.put(
        f"/accounts/{fake_container.account.id}",
        data={
            "name": "업데이트 계좌",
            "cash_balance": "12345.67",
            "kis_account_no": "46592856-01",
        },
    )

    assert response.status_code == 422
    assert "KIS 계좌 검증 서비스가 설정되지 않았습니다." in response.text


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


def test_sync_account_returns_kis_business_error_message(client, fake_container):
    fake_container.kis_account_sync_service.sync_exception = KisApiBusinessError(
        code="OPSQ2000",
        message="ERROR : INPUT INVALID_CHECK_ACNO",
    )

    response = client.post(f"/accounts/{fake_container.account.id}/sync")

    assert response.status_code == 200
    assert "동기화 실패: OPSQ2000 - ERROR : INPUT INVALID_CHECK_ACNO" in response.text


def test_sync_account_empty_snapshot_shows_confirm_then_allows_retry(
    client, fake_container
):
    fake_container.kis_account_sync_service.sync_exception_unless_confirm = (
        KisEmptySnapshotError("보유 종목 스냅샷이 비어 있습니다.")
    )

    first = client.post(f"/accounts/{fake_container.account.id}/sync")

    assert first.status_code == 200
    assert "동기화 중단" in first.text
    assert "전량 매도 확정" in first.text
    assert (
        fake_container.kis_account_sync_service.sync_calls[-1]["allow_empty_snapshot"]
        is False
    )

    second = client.post(
        f"/accounts/{fake_container.account.id}/sync",
        data={"confirm_empty": "true"},
    )

    assert second.status_code == 200
    assert "KIS 계좌 동기화 완료" in second.text
    assert (
        fake_container.kis_account_sync_service.sync_calls[-1]["allow_empty_snapshot"]
        is True
    )
    # Confirm button should not appear on success
    assert "전량 매도 확정" not in second.text


def test_build_stock_name_map_persists_name_when_stock_name_empty(fake_container):
    """Opportunistic name fill: update_name called when stock.name empty."""
    from portfolio_manager.web.routes.accounts import _build_stock_name_map

    class _StubPriceService:
        def get_stock_price(self, ticker, *, preferred_exchange=None):
            return (Decimal("70000"), "KRW", "삼성전자", None)

    fake_container.price_service = _StubPriceService()

    result = _build_stock_name_map(fake_container, [fake_container.stock])

    assert result[fake_container.stock.id] == "삼성전자"
    updated = fake_container.stock_repository.get_by_id(fake_container.stock.id)
    assert updated is not None
    assert updated.name == "삼성전자"


def test_build_stock_name_map_skips_update_when_stock_already_named(fake_container):
    """Skip update_name when stock already has a persisted name."""
    from portfolio_manager.models import Stock
    from portfolio_manager.web.routes.accounts import _build_stock_name_map

    named_stock = Stock(
        id=fake_container.stock.id,
        ticker=fake_container.stock.ticker,
        group_id=fake_container.stock.group_id,
        created_at=fake_container.stock.created_at,
        updated_at=fake_container.stock.updated_at,
        exchange=fake_container.stock.exchange,
        name="기존이름",
    )
    fake_container.stock_repository._stocks[0] = named_stock

    price_calls: list = []

    class _StubPriceService:
        def get_stock_price(self, ticker, *, preferred_exchange=None):
            price_calls.append(ticker)
            return (Decimal("70000"), "KRW", "새이름", None)

    fake_container.price_service = _StubPriceService()

    update_calls: list = []
    original_update_name = fake_container.stock_repository.update_name

    def _spy_update_name(stock_id, name):
        update_calls.append((stock_id, name))
        return original_update_name(stock_id, name)

    fake_container.stock_repository.update_name = _spy_update_name

    result = _build_stock_name_map(fake_container, [named_stock])

    assert result[named_stock.id] == "기존이름"
    assert update_calls == []
    assert price_calls == []
    stored = fake_container.stock_repository.get_by_id(named_stock.id)
    assert stored is not None
    assert stored.name == "기존이름"
