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
