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


def test_holdings_page_contains_labeled_inputs_and_required_fields(
    client, fake_container
):
    response = client.get(f"/accounts/{fake_container.account.id}/holdings")

    assert response.status_code == 200
    body = response.text

    assert "보유 추가" in body
    assert 'name="stock_id"' in body
    assert 'name="quantity"' in body
    assert "required-marker" in body


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
