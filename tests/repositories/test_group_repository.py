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
            "target_percentage": 10.5,
            "created_at": datetime.now().isoformat(),
            "updated_at": datetime.now().isoformat(),
        }
    ]
    mock_client.table.return_value.insert.return_value.execute.return_value = (
        mock_response
    )

    repository = GroupRepository(mock_client)

    # Act
    group = repository.create("Tech Stocks", target_percentage=10.5)

    # Assert
    assert group is not None
    assert group.name == "Tech Stocks"
    assert group.target_percentage == 10.5
    assert group.id is not None
    mock_client.table.assert_called_once_with("groups")
    # Verify insert call arguments
    args, _ = mock_client.table.return_value.insert.call_args
    assert args[0] == {"name": "Tech Stocks", "target_percentage": 10.5}


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
            "target_percentage": 20.0,
            "created_at": now,
            "updated_at": now,
        },
        {
            "id": group2_id,
            "name": "Healthcare",
            "target_percentage": 30.0,
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
    assert groups[0].target_percentage == 20.0
    assert groups[1].name == "Healthcare"
    assert groups[1].target_percentage == 30.0
    mock_client.table.assert_called_once_with("groups")


def test_update_group_updates_fields():
    """Should update group fields."""
    # Arrange
    mock_client = Mock()
    mock_response = MagicMock()
    group_id = str(uuid4())
    now = datetime.now().isoformat()

    mock_response.data = [
        {
            "id": group_id,
            "name": "Updated Name",
            "target_percentage": 15.0,
            "created_at": now,
            "updated_at": now,
        }
    ]
    mock_client.table.return_value.update.return_value.eq.return_value.execute.return_value = mock_response

    repository = GroupRepository(mock_client)

    # Act
    group = repository.update(uuid4(), name="Updated Name", target_percentage=15.0)

    # Assert
    assert group.name == "Updated Name"
    assert group.target_percentage == 15.0
    # Verify update call arguments
    args, _ = mock_client.table.return_value.update.call_args
    assert args[0] == {"name": "Updated Name", "target_percentage": 15.0}
