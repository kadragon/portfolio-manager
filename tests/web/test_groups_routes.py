from uuid import uuid4


def test_get_group_row_returns_404_for_unknown_group(client):
    response = client.get(f"/groups/{uuid4()}")
    assert response.status_code == 404


def test_edit_group_form_returns_404_for_unknown_group(client):
    response = client.get(f"/groups/{uuid4()}/edit")
    assert response.status_code == 404


def test_list_stocks_returns_404_for_unknown_group(client):
    response = client.get(f"/groups/{uuid4()}/stocks")
    assert response.status_code == 404


def test_get_stock_row_returns_404_when_group_does_not_match(client, fake_container):
    client.post("/groups", data={"name": "다른 그룹", "target_percentage": 0})
    other_group_id = fake_container.group_repository.list_all()[0].id

    response = client.get(
        f"/groups/{other_group_id}/stocks/{fake_container.stock.id}",
    )
    assert response.status_code == 404


def test_edit_stock_form_returns_404_when_group_does_not_match(client, fake_container):
    client.post("/groups", data={"name": "다른 그룹", "target_percentage": 0})
    other_group_id = fake_container.group_repository.list_all()[0].id

    response = client.get(
        f"/groups/{other_group_id}/stocks/{fake_container.stock.id}/edit",
    )
    assert response.status_code == 404


def test_update_stock_returns_422_for_blank_ticker(client, fake_container):
    response = client.put(
        f"/groups/{fake_container.group.id}/stocks/{fake_container.stock.id}",
        data={"ticker": "   ", "target_group_id": ""},
    )
    assert response.status_code == 422


def test_update_stock_returns_422_for_invalid_target_group_id(client, fake_container):
    response = client.put(
        f"/groups/{fake_container.group.id}/stocks/{fake_container.stock.id}",
        data={"ticker": "AAPL", "target_group_id": "not-a-uuid"},
    )
    assert response.status_code == 422


def test_update_stock_returns_404_for_missing_target_group(client, fake_container):
    response = client.put(
        f"/groups/{fake_container.group.id}/stocks/{fake_container.stock.id}",
        data={"ticker": "AAPL", "target_group_id": str(uuid4())},
    )
    assert response.status_code == 404


def test_create_stock_normalizes_ticker(client, fake_container):
    response = client.post(
        f"/groups/{fake_container.group.id}/stocks",
        data={"ticker": "  aapl  "},
    )
    assert response.status_code == 200
    assert "AAPL" in response.text
    assert fake_container.stock_repository.list_by_group(fake_container.group.id)[
        0
    ].ticker == ("AAPL")


def test_delete_stock_removes_stock_and_returns_200(client, fake_container):
    response = client.delete(
        f"/groups/{fake_container.group.id}/stocks/{fake_container.stock.id}",
    )

    assert response.status_code == 200
    assert fake_container.stock_repository.get_by_id(fake_container.stock.id) is None
