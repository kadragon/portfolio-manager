def test_deposits_page_uses_button_semantics_and_korean_copy(client):
    response = client.get("/deposits")

    assert response.status_code == 200
    body = response.text

    assert "입금 내역" in body
    assert "입금 추가" in body
    assert 'hx-delete="/deposits/' in body
    assert "<a hx-delete" not in body
    assert "required-marker" in body


def test_deposit_edit_partial_has_accessible_labels(client, fake_container):
    response = client.get(f"/deposits/{fake_container.deposit.id}/edit")

    assert response.status_code == 200
    body = response.text

    assert 'class="sr-only"' in body
    assert 'for="deposit-date-' in body
    assert 'for="deposit-amount-' in body
    assert 'for="deposit-note-' in body


def test_create_deposit_duplicate_date_requests_full_refresh(client, fake_container):
    response = client.post(
        "/deposits",
        data={
            "amount": "1000",
            "deposit_date": fake_container.deposit.deposit_date.isoformat(),
            "note": "수정",
        },
    )

    assert response.status_code == 200
    assert response.headers.get("HX-Refresh") == "true"
    assert "HX-Reswap" not in response.headers


def test_create_deposit_rejects_invalid_date(client):
    response = client.post(
        "/deposits",
        data={"amount": "1000", "deposit_date": "not-a-date", "note": ""},
    )
    assert response.status_code == 422
