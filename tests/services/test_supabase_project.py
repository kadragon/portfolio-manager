"""Tests for Supabase project status restore helpers."""

from unittest.mock import MagicMock, patch

import httpx

from portfolio_manager.services.supabase_project import (
    ProjectStatus,
    check_and_restore_project,
    extract_project_ref,
    get_project_status,
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


def test_extract_project_ref_returns_none_for_invalid_url() -> None:
    """Invalid Supabase URL should return None."""
    assert extract_project_ref("https://example.com") is None


@patch("portfolio_manager.services.supabase_project.httpx.get")
def test_get_project_status_maps_paused(mock_get: MagicMock) -> None:
    """INACTIVE_PAUSED should map to ProjectStatus.PAUSED."""
    response = MagicMock()
    response.status_code = 200
    response.json.return_value = {"status": "INACTIVE_PAUSED"}
    mock_get.return_value = response

    assert get_project_status("testref", "token") == ProjectStatus.PAUSED


@patch("portfolio_manager.services.supabase_project.httpx.get")
def test_get_project_status_maps_restoring(mock_get: MagicMock) -> None:
    """RESTORING status should map to ProjectStatus.RESTORING."""
    response = MagicMock()
    response.status_code = 200
    response.json.return_value = {"status": "SOMETHING_RESTORING_NOW"}
    mock_get.return_value = response

    assert get_project_status("testref", "token") == ProjectStatus.RESTORING


@patch("portfolio_manager.services.supabase_project.httpx.get")
def test_get_project_status_returns_unknown_for_unexpected_status(
    mock_get: MagicMock,
) -> None:
    """Unrecognized status should map to ProjectStatus.UNKNOWN."""
    response = MagicMock()
    response.status_code = 200
    response.json.return_value = {"status": "BROKEN"}
    mock_get.return_value = response

    assert get_project_status("testref", "token") == ProjectStatus.UNKNOWN


@patch("portfolio_manager.services.supabase_project.httpx.get")
def test_get_project_status_returns_unknown_on_http_error(mock_get: MagicMock) -> None:
    """HTTP errors should map to ProjectStatus.UNKNOWN."""
    mock_get.side_effect = httpx.ReadTimeout("timed out")

    assert get_project_status("testref", "token") == ProjectStatus.UNKNOWN


@patch("portfolio_manager.services.supabase_project.httpx.post")
def test_restore_project_returns_false_on_http_error(mock_post: MagicMock) -> None:
    """HTTP errors during restore should return False."""
    mock_post.side_effect = httpx.ConnectError("connect failed")

    assert restore_project("testref", "token123") is False


class TestCheckAndRestoreProject:
    """Tests for check_and_restore_project orchestration flow."""

    @patch.dict(
        "os.environ",
        {"SUPABASE_URL": "https://testref.supabase.co", "SUPABASE_ACCESS_TOKEN": ""},
        clear=True,
    )
    def test_returns_config_error_when_access_token_missing(self) -> None:
        """Should return non-blocking config error when token is missing."""
        result = check_and_restore_project()
        assert result.status == ProjectStatus.UNKNOWN
        assert result.is_config_error is True
        assert result.error is not None

    @patch.dict(
        "os.environ", {"SUPABASE_URL": "", "SUPABASE_ACCESS_TOKEN": "token"}, clear=True
    )
    def test_returns_unknown_when_supabase_url_missing(self) -> None:
        """Missing SUPABASE_URL should return UNKNOWN with error."""
        result = check_and_restore_project()
        assert result.status == ProjectStatus.UNKNOWN
        assert result.error == "SUPABASE_URL not set"

    @patch.dict(
        "os.environ",
        {
            "SUPABASE_URL": "https://not-supabase.example.com",
            "SUPABASE_ACCESS_TOKEN": "token",
        },
        clear=True,
    )
    def test_returns_unknown_when_project_ref_extraction_fails(self) -> None:
        """Invalid Supabase URL should return UNKNOWN with extraction error."""
        result = check_and_restore_project()
        assert result.status == ProjectStatus.UNKNOWN
        assert result.error is not None
        assert "Could not extract project ref" in result.error

    @patch.dict(
        "os.environ",
        {
            "SUPABASE_URL": "https://testref.supabase.co",
            "SUPABASE_ACCESS_TOKEN": "token123",
        },
        clear=True,
    )
    @patch("portfolio_manager.services.supabase_project.get_project_status")
    def test_returns_active_when_project_is_already_active(
        self, mock_get_status: MagicMock
    ) -> None:
        """Should return ACTIVE immediately when project is already ready."""
        mock_get_status.return_value = ProjectStatus.ACTIVE
        updates: list[str] = []

        result = check_and_restore_project(on_status_update=updates.append)

        assert result.status == ProjectStatus.ACTIVE
        assert result.restored is False
        assert result.error is None
        assert updates == ["Checking Supabase project status..."]

    @patch.dict(
        "os.environ",
        {
            "SUPABASE_URL": "https://testref.supabase.co",
            "SUPABASE_ACCESS_TOKEN": "token123",
        },
        clear=True,
    )
    @patch("portfolio_manager.services.supabase_project.wait_for_project_ready")
    @patch("portfolio_manager.services.supabase_project.restore_project")
    @patch("portfolio_manager.services.supabase_project.get_project_status")
    def test_restores_paused_project_successfully(
        self,
        mock_get_status: MagicMock,
        mock_restore_project: MagicMock,
        mock_wait_for_project_ready: MagicMock,
    ) -> None:
        """Should restore a paused project and return ACTIVE."""
        mock_get_status.return_value = ProjectStatus.PAUSED
        mock_restore_project.return_value = True
        mock_wait_for_project_ready.return_value = True

        result = check_and_restore_project()

        assert result.status == ProjectStatus.ACTIVE
        assert result.restored is True
        assert result.error is None

    @patch.dict(
        "os.environ",
        {
            "SUPABASE_URL": "https://testref.supabase.co",
            "SUPABASE_ACCESS_TOKEN": "token123",
        },
        clear=True,
    )
    @patch("portfolio_manager.services.supabase_project.restore_project")
    @patch("portfolio_manager.services.supabase_project.get_project_status")
    def test_returns_paused_with_error_when_restore_request_fails(
        self, mock_get_status: MagicMock, mock_restore_project: MagicMock
    ) -> None:
        """Should return PAUSED with error when restore request fails."""
        mock_get_status.return_value = ProjectStatus.PAUSED
        mock_restore_project.return_value = False

        result = check_and_restore_project()

        assert result.status == ProjectStatus.PAUSED
        assert result.restored is False
        assert result.error is not None

    @patch.dict(
        "os.environ",
        {
            "SUPABASE_URL": "https://testref.supabase.co",
            "SUPABASE_ACCESS_TOKEN": "token123",
        },
        clear=True,
    )
    @patch("portfolio_manager.services.supabase_project.wait_for_project_ready")
    @patch("portfolio_manager.services.supabase_project.restore_project")
    @patch("portfolio_manager.services.supabase_project.get_project_status")
    def test_returns_restoring_when_paused_restore_not_ready(
        self,
        mock_get_status: MagicMock,
        mock_restore_project: MagicMock,
        mock_wait_for_project_ready: MagicMock,
    ) -> None:
        """Paused project should return RESTORING if wait times out."""
        mock_get_status.return_value = ProjectStatus.PAUSED
        mock_restore_project.return_value = True
        mock_wait_for_project_ready.return_value = False

        result = check_and_restore_project()

        assert result.status == ProjectStatus.RESTORING
        assert result.restored is False
        assert result.error is not None

    @patch.dict(
        "os.environ",
        {
            "SUPABASE_URL": "https://testref.supabase.co",
            "SUPABASE_ACCESS_TOKEN": "token123",
        },
        clear=True,
    )
    @patch("portfolio_manager.services.supabase_project.wait_for_project_ready")
    @patch("portfolio_manager.services.supabase_project.get_project_status")
    def test_returns_active_when_restoring_project_becomes_ready(
        self, mock_get_status: MagicMock, mock_wait_for_project_ready: MagicMock
    ) -> None:
        """Should return ACTIVE when project is restoring and then becomes ready."""
        mock_get_status.return_value = ProjectStatus.RESTORING
        mock_wait_for_project_ready.return_value = True

        result = check_and_restore_project()

        assert result.status == ProjectStatus.ACTIVE
        assert result.restored is True

    @patch.dict(
        "os.environ",
        {
            "SUPABASE_URL": "https://testref.supabase.co",
            "SUPABASE_ACCESS_TOKEN": "token123",
        },
        clear=True,
    )
    @patch("portfolio_manager.services.supabase_project.wait_for_project_ready")
    @patch("portfolio_manager.services.supabase_project.get_project_status")
    def test_returns_restoring_when_restoring_project_times_out(
        self, mock_get_status: MagicMock, mock_wait_for_project_ready: MagicMock
    ) -> None:
        """RESTORING project should remain RESTORING if readiness times out."""
        mock_get_status.return_value = ProjectStatus.RESTORING
        mock_wait_for_project_ready.return_value = False

        result = check_and_restore_project()

        assert result.status == ProjectStatus.RESTORING
        assert result.restored is False
        assert result.error is not None

    @patch.dict(
        "os.environ",
        {
            "SUPABASE_URL": "https://testref.supabase.co",
            "SUPABASE_ACCESS_TOKEN": "token123",
        },
        clear=True,
    )
    @patch("portfolio_manager.services.supabase_project.get_project_status")
    def test_returns_unknown_when_status_cannot_be_determined(
        self, mock_get_status: MagicMock
    ) -> None:
        """UNKNOWN project status should return UNKNOWN with message."""
        mock_get_status.return_value = ProjectStatus.UNKNOWN

        result = check_and_restore_project()

        assert result.status == ProjectStatus.UNKNOWN
        assert result.error == "Could not determine project status"
