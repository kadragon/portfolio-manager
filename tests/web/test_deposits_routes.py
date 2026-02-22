from datetime import date
from uuid import uuid4


def test_get_deposit_row_returns_404_for_unknown_deposit(client):
    response = client.get(f"/deposits/{uuid4()}")
    assert response.status_code == 404


def test_create_deposit_without_duplicate_date_does_not_force_refresh(
    client, fake_container
):
    response = client.post(
        "/deposits",
        data={
            "amount": "1234",
            "deposit_date": date(2026, 1, 6).isoformat(),
            "note": "",
        },
    )

    assert response.status_code == 200
    assert response.headers.get("HX-Refresh") is None
    assert "₩1,234" in response.text
    assert fake_container.deposit_repository.list_all()[0].amount == 1234


def test_edit_deposit_form_returns_404_for_unknown_deposit(client):
    response = client.get(f"/deposits/{uuid4()}/edit")
    assert response.status_code == 404


def test_delete_deposit_removes_item_and_returns_200(client, fake_container):
    response = client.delete(f"/deposits/{fake_container.deposit.id}")

    assert response.status_code == 200
    assert fake_container.deposit_repository.list_all() == []


def test_update_deposit_sets_trimmed_non_blank_note(client, fake_container):
    response = client.put(
        f"/deposits/{fake_container.deposit.id}",
        data={
            "amount": "900001",
            "deposit_date": fake_container.deposit.deposit_date.isoformat(),
            "note": "  업데이트 메모  ",
        },
    )

    assert response.status_code == 200
    assert "업데이트 메모" in response.text
    assert fake_container.deposit_repository.list_all()[0].note == "업데이트 메모"
