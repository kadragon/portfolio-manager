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
