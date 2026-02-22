def test_groups_page_uses_button_semantics_for_mutations(client):
    response = client.get("/groups")

    assert response.status_code == 200
    body = response.text

    assert "그룹 추가" in body
    assert 'type="button"' in body
    assert 'hx-delete="/groups/' in body
    assert "수정" in body
    assert "삭제" in body
    assert "<a hx-delete" not in body
    assert "required-marker" in body


def test_group_stocks_page_has_required_labeled_form(client, fake_container):
    response = client.get(f"/groups/{fake_container.group.id}/stocks")

    assert response.status_code == 200
    body = response.text

    assert "종목 추가" in body
    assert 'name="ticker"' in body
    assert 'class="input-uppercase"' in body
    assert "required-marker" in body


def test_group_stock_row_has_edit_button(client, fake_container):
    response = client.get(f"/groups/{fake_container.group.id}/stocks")
    assert response.status_code == 200
    body = response.text

    assert "수정" in body
    assert f'hx-get="/groups/{fake_container.group.id}/stocks/' in body
    assert "/edit" in body


def test_stock_edit_form_renders_group_move_select(client, fake_container):
    response = client.get(
        f"/groups/{fake_container.group.id}/stocks/{fake_container.stock.id}/edit"
    )
    assert response.status_code == 200
    body = response.text

    assert 'name="ticker"' in body
    assert 'name="target_group_id"' in body
    assert "저장" in body
    assert "취소" in body


def test_update_stock_can_change_ticker_and_group(client, fake_container):
    create_group = client.post(
        "/groups", data={"name": "미국 주식", "target_percentage": 50}
    )
    assert create_group.status_code == 200
    target_group_id = fake_container.group_repository.list_all()[0].id

    response = client.put(
        f"/groups/{fake_container.group.id}/stocks/{fake_container.stock.id}",
        data={
            "ticker": "aapl",
            "target_group_id": str(target_group_id),
        },
    )
    assert response.status_code == 200
    assert "AAPL" in response.text

    moved_page = client.get(f"/groups/{target_group_id}/stocks")
    assert moved_page.status_code == 200
    assert "AAPL" in moved_page.text
