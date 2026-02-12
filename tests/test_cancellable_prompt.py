"""Tests for cancellable prompt functionality."""

from decimal import Decimal
from unittest.mock import MagicMock

from rich.console import Console


class TestCancellablePrompt:
    """Test cancellable_prompt returns None on ESC or Ctrl+C."""

    def test_returns_input_on_normal_entry(self):
        """Normal input should return the entered value."""
        from portfolio_manager.cli.prompt_select import cancellable_prompt

        mock_session = MagicMock()
        mock_session.prompt.return_value = "test value"

        result = cancellable_prompt("Enter name:", session=mock_session)

        assert result == "test value"
        mock_session.prompt.assert_called_once()

    def test_returns_none_on_keyboard_interrupt(self):
        """Ctrl+C should return None instead of raising."""
        from portfolio_manager.cli.prompt_select import cancellable_prompt

        mock_session = MagicMock()
        mock_session.prompt.side_effect = KeyboardInterrupt()

        result = cancellable_prompt("Enter name:", session=mock_session)

        assert result is None

    def test_returns_none_on_eof_error(self):
        """Ctrl+D or ESC abort should return None."""
        from portfolio_manager.cli.prompt_select import cancellable_prompt

        mock_session = MagicMock()
        mock_session.prompt.side_effect = EOFError()

        result = cancellable_prompt("Enter name:", session=mock_session)

        assert result is None

    def test_accepts_default_value(self):
        """Default value should be passed to the prompt."""
        from portfolio_manager.cli.prompt_select import cancellable_prompt

        mock_session = MagicMock()
        mock_session.prompt.return_value = "default_val"

        result = cancellable_prompt(
            "Enter name:", default="default_val", session=mock_session
        )

        assert result == "default_val"
        call_kwargs = mock_session.prompt.call_args
        assert call_kwargs[1].get("default") == "default_val"

    def test_prompt_decimal_retries_until_valid_decimal(self):
        """prompt_decimal should retry on invalid input and return Decimal."""
        from portfolio_manager.cli.prompt_select import prompt_decimal

        mock_session = MagicMock()
        mock_session.prompt.side_effect = ["abc", "12.50"]

        result = prompt_decimal("Amount:", session=mock_session)

        assert result == Decimal("12.50")

    def test_prompt_decimal_shows_error_message_on_invalid_input(self):
        """Invalid decimal input should render an error message before retrying."""
        from portfolio_manager.cli.prompt_select import prompt_decimal

        mock_session = MagicMock()
        mock_session.prompt.side_effect = ["abc", "12.50"]
        console = Console(record=True, width=80)

        result = prompt_decimal("Amount:", session=mock_session, console=console)

        assert result == Decimal("12.50")
        output = console.export_text()
        assert "Invalid number. Please try again." in output

    def test_prompt_decimal_returns_none_when_cancelled(self):
        """prompt_decimal should return None when prompt is cancelled."""
        from portfolio_manager.cli.prompt_select import prompt_decimal

        mock_session = MagicMock()
        mock_session.prompt.side_effect = KeyboardInterrupt()

        result = prompt_decimal("Amount:", session=mock_session)

        assert result is None
