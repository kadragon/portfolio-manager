"""Tests for group repository."""

import pytest

from portfolio_manager.repositories.group_repository import GroupRepository


def test_create_group_returns_group_with_id():
    repo = GroupRepository()
    group = repo.create("Tech Stocks", target_percentage=10.5)

    assert group is not None
    assert group.name == "Tech Stocks"
    assert group.target_percentage == 10.5
    assert group.id is not None


def test_list_all_returns_all_groups():
    repo = GroupRepository()
    repo.create("Tech Stocks", target_percentage=20.0)
    repo.create("Healthcare", target_percentage=30.0)

    groups = repo.list_all()

    assert len(groups) == 2
    names = {g.name for g in groups}
    assert names == {"Tech Stocks", "Healthcare"}


def test_update_group_updates_fields():
    repo = GroupRepository()
    group = repo.create("Original", target_percentage=10.0)

    updated = repo.update(group.id, name="Updated Name", target_percentage=15.0)

    assert updated.name == "Updated Name"
    assert updated.target_percentage == 15.0


def test_update_group_raises_when_no_fields():
    repo = GroupRepository()
    group = repo.create("Test")

    with pytest.raises(ValueError, match="No fields to update"):
        repo.update(group.id)


def test_delete_group():
    repo = GroupRepository()
    group = repo.create("To Delete")

    repo.delete(group.id)

    assert repo.list_all() == []
