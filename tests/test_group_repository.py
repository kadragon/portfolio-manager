"""Tests for group repository."""

from datetime import datetime
from unittest.mock import MagicMock, Mock
from uuid import uuid4

from portfolio_manager.repositories.group_repository import GroupRepository


def test_create_group_returns_group_with_id():
    """Should create a group and return it with an ID."""
    # Arrange
    mock_client = Mock()
    mock_response = MagicMock()
    mock_response.data = [
        {
            "id": str(uuid4()),
            "name": "Tech Stocks",
            "created_at": datetime.now().isoformat(),
            "updated_at": datetime.now().isoformat(),
        }
    ]
    mock_client.table.return_value.insert.return_value.execute.return_value = (
        mock_response
    )

    repository = GroupRepository(mock_client)

    # Act
    group = repository.create("Tech Stocks")

    # Assert
    assert group is not None
    assert group.name == "Tech Stocks"
    assert group.id is not None
    mock_client.table.assert_called_once_with("groups")


def test_list_all_returns_all_groups():
    """Should return all groups from the database."""
    # Arrange
    mock_client = Mock()
    mock_response = MagicMock()
    group1_id = str(uuid4())
    group2_id = str(uuid4())
    now = datetime.now().isoformat()

    mock_response.data = [
        {
            "id": group1_id,
            "name": "Tech Stocks",
            "created_at": now,
            "updated_at": now,
        },
        {
            "id": group2_id,
            "name": "Healthcare",
            "created_at": now,
            "updated_at": now,
        },
    ]
    mock_client.table.return_value.select.return_value.execute.return_value = (
        mock_response
    )

    repository = GroupRepository(mock_client)

    # Act
    groups = repository.list_all()

    # Assert
    assert len(groups) == 2
    assert groups[0].name == "Tech Stocks"
    assert groups[1].name == "Healthcare"
    mock_client.table.assert_called_once_with("groups")
