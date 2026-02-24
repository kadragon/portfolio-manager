def test_deposits_page_uses_button_semantics_and_korean_copy(client):
    response = client.get("/deposits")

    assert response.status_code == 200
    body = response.text

    assert "입금 내역" in body
    assert '<h1 class="page-header">입금 내역</h1>' in body
    assert "날짜별 입금 금액과 메모" in body
    assert (
        'hx-on::after-request="if (event.detail.successful) { this.reset(); }"' in body
    )
    assert 'hx-on::after-request="this.reset()"' not in body
    assert "입금 추가" in body
    assert 'hx-delete="/deposits/' in body
    assert "<a hx-delete" not in body
    assert "required-marker" in body
    assert body.index("입금 추가") < body.index('id="deposits-body"')


def test_deposit_edit_partial_has_accessible_labels(client, fake_container):
    response = client.get(f"/deposits/{fake_container.deposit.id}/edit")

    assert response.status_code == 200
    body = response.text

    assert 'class="sr-only"' in body
    assert 'for="deposit-date-' in body
    assert 'for="deposit-amount-' in body
    assert 'for="deposit-note-' in body
    assert "/clear" in body


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


def test_update_deposit_keeps_note_when_blank_input(client, fake_container):
    response = client.put(
        f"/deposits/{fake_container.deposit.id}",
        data={
            "amount": "900001",
            "deposit_date": fake_container.deposit.deposit_date.isoformat(),
            "note": "   ",
        },
    )
    assert response.status_code == 200
    assert "초기 입금" in response.text


def test_update_deposit_clears_note_with_clear_keyword(client, fake_container):
    response = client.put(
        f"/deposits/{fake_container.deposit.id}",
        data={
            "amount": "900001",
            "deposit_date": fake_container.deposit.deposit_date.isoformat(),
            "note": "/clear",
        },
    )
    assert response.status_code == 200
    assert ">-<" in response.text
