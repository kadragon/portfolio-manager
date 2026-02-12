"""Tests for Supabase auto-resume functionality."""

import pytest
from unittest.mock import patch, MagicMock

import httpx

from portfolio_manager.services.supabase_client import (
    extract_project_ref,
    is_paused_project_error,
    restore_paused_project,
    wait_for_project_ready,
    with_auto_resume,
)


class TestExtractProjectRef:
    """Tests for extract_project_ref function."""

    def test_extracts_ref_from_valid_url(self) -> None:
        """Should extract project ref from valid Supabase URL."""
        url = "https://dnbzniywsqntkxfdwdvn.supabase.co"
        assert extract_project_ref(url) == "dnbzniywsqntkxfdwdvn"

    def test_extracts_ref_with_mixed_alphanumeric(self) -> None:
        """Should extract project ref with mixed alphanumeric."""
        url = "https://abc123def456.supabase.co"
        assert extract_project_ref(url) == "abc123def456"

    def test_raises_on_invalid_url(self) -> None:
        """Should raise ValueError on invalid URL."""
        with pytest.raises(ValueError, match="Invalid Supabase URL format"):
            extract_project_ref("https://example.com")

    def test_raises_on_empty_url(self) -> None:
        """Should raise ValueError on empty URL."""
        with pytest.raises(ValueError, match="Invalid Supabase URL format"):
            extract_project_ref("")


class TestIsPausedProjectError:
    """Tests for is_paused_project_error function."""

    def test_returns_true_for_connect_error(self) -> None:
        """Should return True for httpx.ConnectError."""
        error = httpx.ConnectError("Connection failed")
        assert is_paused_project_error(error) is True

    def test_returns_true_for_nodename_error(self) -> None:
        """Should return True for DNS resolution errors."""
        error = Exception("nodename nor servname provided")
        assert is_paused_project_error(error) is True

    def test_returns_true_for_name_resolution_error(self) -> None:
        """Should return True for name resolution errors."""
        error = Exception("Name resolution failed")
        assert is_paused_project_error(error) is True

    def test_returns_false_for_other_errors(self) -> None:
        """Should return False for unrelated errors."""
        error = Exception("Something else went wrong")
        assert is_paused_project_error(error) is False


class TestRestorePausedProject:
    """Tests for restore_paused_project function."""

    @patch("portfolio_manager.services.supabase_client.httpx.Client")
    def test_returns_true_on_success(self, mock_client_cls: MagicMock) -> None:
        """Should return True when restore succeeds (HTTP 201)."""
        mock_client = MagicMock()
        mock_response = MagicMock()
        mock_response.status_code = 201
        mock_client.post.return_value = mock_response
        mock_client.__enter__ = MagicMock(return_value=mock_client)
        mock_client.__exit__ = MagicMock(return_value=False)
        mock_client_cls.return_value = mock_client

        result = restore_paused_project("testref", "token123")
        assert result is True
        mock_client.post.assert_called_once_with(
            "/v1/projects/testref/restore",
            headers={"Authorization": "Bearer token123"},
        )

    @patch("portfolio_manager.services.supabase_client.httpx.Client")
    def test_returns_false_on_failure(self, mock_client_cls: MagicMock) -> None:
        """Should return False when restore fails."""
        mock_client = MagicMock()
        mock_response = MagicMock()
        mock_response.status_code = 401
        mock_client.post.return_value = mock_response
        mock_client.__enter__ = MagicMock(return_value=mock_client)
        mock_client.__exit__ = MagicMock(return_value=False)
        mock_client_cls.return_value = mock_client

        result = restore_paused_project("testref", "badtoken")
        assert result is False

    @patch("portfolio_manager.services.supabase_client.httpx.Client")
    def test_returns_false_on_200(self, mock_client_cls: MagicMock) -> None:
        """Should return False when HTTP 200 (not 201)."""
        mock_client = MagicMock()
        mock_response = MagicMock()
        mock_response.status_code = 200
        mock_client.post.return_value = mock_response
        mock_client.__enter__ = MagicMock(return_value=mock_client)
        mock_client.__exit__ = MagicMock(return_value=False)
        mock_client_cls.return_value = mock_client

        result = restore_paused_project("testref", "token123")
        assert result is False

    @patch("portfolio_manager.services.supabase_client.httpx.Client")
    def test_returns_false_on_read_timeout(self, mock_client_cls: MagicMock) -> None:
        """Should return False when management API request times out."""
        mock_client = MagicMock()
        mock_client.post.side_effect = httpx.ReadTimeout("The read operation timed out")
        mock_client.__enter__ = MagicMock(return_value=mock_client)
        mock_client.__exit__ = MagicMock(return_value=False)
        mock_client_cls.return_value = mock_client

        result = restore_paused_project("testref", "token123")
        assert result is False


class TestWaitForProjectReady:
    """Tests for wait_for_project_ready function."""

    @patch("portfolio_manager.services.supabase_client.time.sleep")
    @patch("portfolio_manager.services.supabase_client.httpx.Client")
    def test_returns_true_when_active(
        self, mock_client_cls: MagicMock, mock_sleep: MagicMock
    ) -> None:
        """Should return True when project becomes ACTIVE_HEALTHY."""
        mock_client = MagicMock()
        mock_response = MagicMock()
        mock_response.status_code = 200
        mock_response.json.return_value = {"status": "ACTIVE_HEALTHY"}
        mock_client.get.return_value = mock_response
        mock_client.__enter__ = MagicMock(return_value=mock_client)
        mock_client.__exit__ = MagicMock(return_value=False)
        mock_client_cls.return_value = mock_client

        result = wait_for_project_ready("testref", "token123")
        assert result is True

    @patch("portfolio_manager.services.supabase_client.time.sleep")
    @patch("portfolio_manager.services.supabase_client.httpx.Client")
    def test_returns_false_on_timeout(
        self, mock_client_cls: MagicMock, mock_sleep: MagicMock
    ) -> None:
        """Should return False when timeout is reached."""
        mock_client = MagicMock()
        mock_response = MagicMock()
        mock_response.status_code = 200
        mock_response.json.return_value = {"status": "RESTORING"}
        mock_client.get.return_value = mock_response
        mock_client.__enter__ = MagicMock(return_value=mock_client)
        mock_client.__exit__ = MagicMock(return_value=False)
        mock_client_cls.return_value = mock_client

        result = wait_for_project_ready(
            "testref", "token123", max_wait_seconds=20, poll_interval=10
        )
        assert result is False

    @patch("portfolio_manager.services.supabase_client.time.sleep")
    @patch("portfolio_manager.services.supabase_client.httpx.Client")
    def test_handles_non_200_status(
        self, mock_client_cls: MagicMock, mock_sleep: MagicMock
    ) -> None:
        """Should handle non-200 responses gracefully and continue polling."""
        mock_client = MagicMock()
        # First call returns 500, second returns 200 with ACTIVE_HEALTHY
        mock_response_error = MagicMock()
        mock_response_error.status_code = 500
        mock_response_success = MagicMock()
        mock_response_success.status_code = 200
        mock_response_success.json.return_value = {"status": "ACTIVE_HEALTHY"}
        mock_client.get.side_effect = [mock_response_error, mock_response_success]
        mock_client.__enter__ = MagicMock(return_value=mock_client)
        mock_client.__exit__ = MagicMock(return_value=False)
        mock_client_cls.return_value = mock_client

        result = wait_for_project_ready(
            "testref", "token123", max_wait_seconds=30, poll_interval=10
        )
        assert result is True
        assert mock_client.get.call_count == 2


class TestWithAutoResume:
    """Tests for with_auto_resume function."""

    def test_successful_execution_without_errors(self) -> None:
        """Should return result when function executes successfully."""
        result = with_auto_resume(lambda: "success")
        assert result == "success"

    def test_reraises_non_paused_project_error(self) -> None:
        """Should re-raise errors that are not paused project errors."""

        def raise_value_error() -> None:
            raise ValueError("Some other error")

        with pytest.raises(ValueError, match="Some other error"):
            with_auto_resume(raise_value_error)

    @patch.dict(
        "os.environ",
        {"SUPABASE_URL": "https://testref.supabase.co", "SUPABASE_ACCESS_TOKEN": ""},
    )
    def test_raises_when_access_token_missing(self) -> None:
        """Should raise ValueError when access token is not set."""

        def raise_connect_error() -> None:
            raise httpx.ConnectError("Connection failed")

        with pytest.raises(ValueError, match="SUPABASE_ACCESS_TOKEN is not set"):
            with_auto_resume(raise_connect_error)

    @patch("portfolio_manager.services.supabase_client.wait_for_project_ready")
    @patch("portfolio_manager.services.supabase_client.restore_paused_project")
    @patch.dict(
        "os.environ",
        {
            "SUPABASE_URL": "https://testref.supabase.co",
            "SUPABASE_ACCESS_TOKEN": "token123",
        },
    )
    def test_successful_restore_and_retry(
        self, mock_restore: MagicMock, mock_wait: MagicMock
    ) -> None:
        """Should restore project and retry function on paused project error."""
        mock_restore.return_value = True
        mock_wait.return_value = True

        call_count = 0

        def fail_then_succeed() -> str:
            nonlocal call_count
            call_count += 1
            if call_count == 1:
                raise httpx.ConnectError("Connection failed")
            return "success"

        result = with_auto_resume(fail_then_succeed)
        assert result == "success"
        assert call_count == 2
        mock_restore.assert_called_once_with("testref", "token123")
        mock_wait.assert_called_once_with("testref", "token123")

    @patch("portfolio_manager.services.supabase_client.restore_paused_project")
    @patch.dict(
        "os.environ",
        {
            "SUPABASE_URL": "https://testref.supabase.co",
            "SUPABASE_ACCESS_TOKEN": "badtoken",
        },
    )
    def test_raises_when_restore_fails(self, mock_restore: MagicMock) -> None:
        """Should raise RuntimeError when restore request fails."""
        mock_restore.return_value = False

        def raise_connect_error() -> None:
            raise httpx.ConnectError("Connection failed")

        with pytest.raises(RuntimeError, match="Failed to restore Supabase project"):
            with_auto_resume(raise_connect_error)

    @patch("portfolio_manager.services.supabase_client.wait_for_project_ready")
    @patch("portfolio_manager.services.supabase_client.restore_paused_project")
    @patch.dict(
        "os.environ",
        {
            "SUPABASE_URL": "https://testref.supabase.co",
            "SUPABASE_ACCESS_TOKEN": "token123",
        },
    )
    def test_raises_when_wait_times_out(
        self, mock_restore: MagicMock, mock_wait: MagicMock
    ) -> None:
        """Should raise RuntimeError when waiting for project times out."""
        mock_restore.return_value = True
        mock_wait.return_value = False

        def raise_connect_error() -> None:
            raise httpx.ConnectError("Connection failed")

        with pytest.raises(RuntimeError, match="Timeout waiting for Supabase project"):
            with_auto_resume(raise_connect_error)
