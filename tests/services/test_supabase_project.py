"""Tests for Supabase project status restore helpers."""

from unittest.mock import MagicMock, patch

from portfolio_manager.services.supabase_project import (
    ProjectStatus,
    restore_project,
    wait_for_project_ready,
)


@patch("portfolio_manager.services.supabase_project.httpx.post")
def test_restore_project_treats_201_as_success(mock_post: MagicMock) -> None:
    """Should treat restore API accepted response as success."""
    mock_response = MagicMock()
    mock_response.status_code = 201
    mock_post.return_value = mock_response

    assert restore_project("testref", "token123") is True


@patch("portfolio_manager.services.supabase_project.time.sleep")
@patch("portfolio_manager.services.supabase_project.get_project_status")
def test_wait_for_project_ready_retries_when_status_is_paused(
    mock_get_status: MagicMock, mock_sleep: MagicMock
) -> None:
    """Should continue polling through transient paused states."""
    mock_get_status.side_effect = [
        ProjectStatus.PAUSED,
        ProjectStatus.RESTORING,
        ProjectStatus.ACTIVE,
    ]

    result = wait_for_project_ready(
        "testref",
        "token123",
        max_wait_seconds=30,
        poll_interval=10,
    )

    assert result is True
    assert mock_get_status.call_count == 3
