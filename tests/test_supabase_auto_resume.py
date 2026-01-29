"""Tests for Supabase auto-resume functionality."""

import pytest
from unittest.mock import patch, MagicMock

import httpx

from portfolio_manager.services.supabase_client import (
    extract_project_ref,
    is_paused_project_error,
    restore_paused_project,
    wait_for_project_ready,
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
        """Should return True when restore succeeds."""
        mock_client = MagicMock()
        mock_response = MagicMock()
        mock_response.status_code = 200
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
