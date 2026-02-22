from decimal import Decimal


def test_accounts_page_uses_buttons_for_edit_sync_delete(client):
    response = client.get("/accounts")

    assert response.status_code == 200
    body = response.text

    assert "계좌 추가" in body
    assert "KIS 동기화" in body
    assert 'hx-post="/accounts/' in body
    assert 'hx-delete="/accounts/' in body
    assert "<a hx-post" not in body
    assert "<a hx-delete" not in body


def test_accounts_page_contains_bulk_cash_form(client):
    response = client.get("/accounts")
    assert response.status_code == 200
    body = response.text

    assert "일괄 예수금 수정" in body
    assert 'hx-put="/accounts/bulk-cash"' in body
    assert 'name="cash_' in body


def test_bulk_cash_update_succeeds_and_requests_refresh(client, fake_container):
    response = client.put(
        "/accounts/bulk-cash",
        data={f"cash_{fake_container.account.id}": "777777"},
    )
    assert response.status_code == 200
    assert response.headers.get("HX-Refresh") == "true"
    assert fake_container.account_repository.list_all()[0].cash_balance == 777777


def test_bulk_cash_update_rejects_partial_input(client, fake_container):
    response = client.put("/accounts/bulk-cash", data={})
    assert response.status_code == 422
    assert "예수금을 입력하세요" in response.text


def test_bulk_cash_update_escapes_account_name_for_missing_input(
    client, fake_container
):
    fake_container.account_repository._accounts[0].name = "<script>alert(1)</script>"

    response = client.put("/accounts/bulk-cash", data={})

    assert response.status_code == 422
    assert "<script>alert(1)</script>" not in response.text
    assert "&lt;script&gt;alert(1)&lt;/script&gt;" in response.text


def test_bulk_cash_update_escapes_account_name_for_invalid_number(
    client, fake_container
):
    fake_container.account_repository._accounts[0].name = "<img src=x onerror=1>"

    response = client.put(
        "/accounts/bulk-cash",
        data={f"cash_{fake_container.account.id}": "not-a-number"},
    )

    assert response.status_code == 422
    assert "<img src=x onerror=1>" not in response.text
    assert "&lt;img src=x onerror=1&gt;" in response.text
    assert "예수금 형식이 올바르지 않습니다." in response.text


def test_holdings_page_contains_labeled_inputs_and_required_fields(
    client, fake_container
):
    response = client.get(f"/accounts/{fake_container.account.id}/holdings")

    assert response.status_code == 200
    body = response.text

    assert "보유 추가" in body
    assert "종목명" in body
    assert "일괄 저장" in body
    assert f'hx-put="/accounts/{fake_container.account.id}/holdings/bulk"' in body
    assert 'name="stock_id"' in body
    assert 'name="quantity"' in body
    assert 'name="ticker"' in body
    assert 'name="group_id"' in body
    assert 'name="new_group_name"' in body
    assert 'name="holding_id"' in body
    assert "required-marker" in body
    assert (
        f"/accounts/{fake_container.account.id}/holdings/{fake_container.holding.id}/edit"
        not in body
    )


def test_holdings_page_shows_stock_name_when_price_service_available(
    client, fake_container
):
    class FakePriceService:
        def get_stock_price(self, ticker: str, preferred_exchange=None):
            assert ticker == fake_container.stock.ticker
            return Decimal("70000"), "KRW", "삼성전자", preferred_exchange

    fake_container.price_service = FakePriceService()

    response = client.get(f"/accounts/{fake_container.account.id}/holdings")

    assert response.status_code == 200
    body = response.text
    assert "삼성전자" in body


def test_bulk_update_holdings_updates_all_quantities(client, fake_container):
    second_holding = fake_container.holding_repository.create(
        account_id=fake_container.account.id,
        stock_id=fake_container.stock.id,
        quantity=Decimal("4.0"),
    )

    response = client.put(
        f"/accounts/{fake_container.account.id}/holdings/bulk",
        data={
            "holding_id": [
                str(fake_container.holding.id),
                str(second_holding.id),
            ],
            "quantity": ["11.5", "7.25"],
        },
    )

    assert response.status_code == 200
    body = response.text
    assert "보유 수량을 일괄 저장했습니다." in body
    assert 'hx-swap-oob="innerHTML"' in body

    holdings = fake_container.holding_repository.list_by_account(
        fake_container.account.id
    )
    quantity_by_id = {holding.id: holding.quantity for holding in holdings}
    assert quantity_by_id[fake_container.holding.id] == Decimal("11.5")
    assert quantity_by_id[second_holding.id] == Decimal("7.25")


def test_bulk_update_holdings_rejects_zero_and_keeps_existing_values(
    client, fake_container
):
    second_holding = fake_container.holding_repository.create(
        account_id=fake_container.account.id,
        stock_id=fake_container.stock.id,
        quantity=Decimal("4.0"),
    )
    before = {
        holding.id: holding.quantity
        for holding in fake_container.holding_repository.list_by_account(
            fake_container.account.id
        )
    }

    response = client.put(
        f"/accounts/{fake_container.account.id}/holdings/bulk",
        data={
            "holding_id": [
                str(fake_container.holding.id),
                str(second_holding.id),
            ],
            "quantity": ["11.5", "0"],
        },
    )

    assert response.status_code == 400
    assert "모든 수량은 0보다 커야 합니다." in response.text
    after = {
        holding.id: holding.quantity
        for holding in fake_container.holding_repository.list_by_account(
            fake_container.account.id
        )
    }
    assert after == before


def test_bulk_update_holdings_rejects_other_account_holding_and_keeps_existing_values(
    client, fake_container
):
    other_account = fake_container.account_repository.create(
        name="서브 계좌",
        cash_balance=Decimal("0"),
    )
    other_holding = fake_container.holding_repository.create(
        account_id=other_account.id,
        stock_id=fake_container.stock.id,
        quantity=Decimal("9"),
    )
    before = {
        holding.id: holding.quantity
        for holding in fake_container.holding_repository.list_by_account(
            fake_container.account.id
        )
    }

    response = client.put(
        f"/accounts/{fake_container.account.id}/holdings/bulk",
        data={
            "holding_id": [
                str(fake_container.holding.id),
                str(other_holding.id),
            ],
            "quantity": ["11.5", "7"],
        },
    )

    assert response.status_code == 400
    assert "요청한 holding_id가 현재 계좌에 속하지 않습니다." in response.text
    after = {
        holding.id: holding.quantity
        for holding in fake_container.holding_repository.list_by_account(
            fake_container.account.id
        )
    }
    assert after == before


def test_bulk_update_holdings_rejects_duplicate_ids_and_keeps_existing_values(
    client, fake_container
):
    before = {
        holding.id: holding.quantity
        for holding in fake_container.holding_repository.list_by_account(
            fake_container.account.id
        )
    }

    response = client.put(
        f"/accounts/{fake_container.account.id}/holdings/bulk",
        data={
            "holding_id": [
                str(fake_container.holding.id),
                str(fake_container.holding.id),
            ],
            "quantity": ["11.5", "7"],
        },
    )

    assert response.status_code == 400
    assert "요청에 중복된 holding_id가 포함되어 있습니다." in response.text
    after = {
        holding.id: holding.quantity
        for holding in fake_container.holding_repository.list_by_account(
            fake_container.account.id
        )
    }
    assert after == before


def test_bulk_update_holdings_rejects_length_mismatch_and_keeps_existing_values(
    client, fake_container
):
    before = {
        holding.id: holding.quantity
        for holding in fake_container.holding_repository.list_by_account(
            fake_container.account.id
        )
    }

    response = client.put(
        f"/accounts/{fake_container.account.id}/holdings/bulk",
        data={
            "holding_id": [str(fake_container.holding.id)],
            "quantity": ["11.5", "7"],
        },
    )

    assert response.status_code == 400
    assert "holding_id와 quantity 개수가 일치하지 않습니다." in response.text
    after = {
        holding.id: holding.quantity
        for holding in fake_container.holding_repository.list_by_account(
            fake_container.account.id
        )
    }
    assert after == before


def test_bulk_update_holdings_rejects_missing_holding_ids_payload(
    client, fake_container
):
    response = client.put(
        f"/accounts/{fake_container.account.id}/holdings/bulk",
        data={"quantity": ["11.5"]},
    )

    assert response.status_code == 400
    assert "요청 payload에 holding_id가 없습니다." in response.text


def test_create_holding_by_ticker_uses_existing_stock(client, fake_container):
    response = client.post(
        f"/accounts/{fake_container.account.id}/holdings/by-ticker",
        data={"ticker": fake_container.stock.ticker, "quantity": "3"},
    )
    assert response.status_code == 200
    assert fake_container.stock.ticker in response.text


def test_create_holding_by_ticker_creates_stock_in_selected_group(
    client, fake_container
):
    response = client.post(
        f"/accounts/{fake_container.account.id}/holdings/by-ticker",
        data={
            "ticker": "AAPL",
            "quantity": "2",
            "group_id": str(fake_container.group.id),
        },
    )
    assert response.status_code == 200
    assert "AAPL" in response.text


def test_create_holding_by_ticker_requires_group_for_new_ticker(client, fake_container):
    response = client.post(
        f"/accounts/{fake_container.account.id}/holdings/by-ticker",
        data={"ticker": "MSFT", "quantity": "2"},
    )
    assert response.status_code == 422
    assert "새 티커는 그룹을 선택해야 합니다" in response.text


def test_create_holding_by_ticker_creates_group_when_none_exists(
    client, fake_container
):
    fake_container.group_repository._groups = []
    fake_container.stock_repository._stocks = []

    response = client.post(
        f"/accounts/{fake_container.account.id}/holdings/by-ticker",
        data={
            "ticker": "TSLA",
            "quantity": "1",
            "new_group_name": "신규 그룹",
        },
    )
    assert response.status_code == 200
    assert "TSLA" in response.text
    assert fake_container.group_repository.list_all()[0].name == "신규 그룹"


def test_account_sync_partial_renders_live_message(client, fake_container):
    response = client.post(f"/accounts/{fake_container.account.id}/sync")

    assert response.status_code == 200
    body = response.text

    assert 'id="sync-result-' in body
    assert 'aria-live="polite"' in body


def test_account_sync_fails_fast_when_kis_credentials_missing(client, fake_container):
    fake_container.kis_cano = None
    fake_container.kis_acnt_prdt_cd = None

    response = client.post(f"/accounts/{fake_container.account.id}/sync")
    assert response.status_code == 200
    assert "KIS 계좌 정보(번호/상품코드)가 설정되지 않았습니다." in response.text


def test_create_account_rejects_invalid_cash_balance(client):
    response = client.post(
        "/accounts",
        data={"name": "테스트 계좌", "cash_balance": "not-a-number"},
    )
    assert response.status_code == 422


def test_create_holding_rejects_invalid_quantity(client, fake_container):
    response = client.post(
        f"/accounts/{fake_container.account.id}/holdings",
        data={"stock_id": str(fake_container.stock.id), "quantity": "bad"},
    )
    assert response.status_code == 422
