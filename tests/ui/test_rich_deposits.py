"""Tests for Rich-based deposit flows."""

from datetime import date, datetime
from decimal import Decimal
from unittest.mock import MagicMock, patch
from uuid import uuid4

from rich.console import Console

from portfolio_manager.cli.deposits import (
    add_deposit_flow,
    render_deposit_list,
    run_deposit_menu,
    delete_deposit_flow,
    get_date_input,
    select_deposit_to_edit,
    update_deposit_flow,
)
from portfolio_manager.cli.prompt_select import choose_deposit_from_list
from portfolio_manager.models import Deposit


def _make_deposit(*, amount: Decimal, note: str | None = None) -> Deposit:
    now = datetime.now()
    return Deposit(
        id=uuid4(),
        amount=amount,
        deposit_date=date(2026, 2, 6),
        note=note,
        created_at=now,
        updated_at=now,
    )


def test_update_deposit_flow_blank_amount_keeps_existing_value():
    """Blank amount input should keep the current amount."""
    console = Console(record=True, width=80)
    repo = MagicMock()
    deposit = _make_deposit(amount=Decimal("100000"), note="seed")

    update_deposit_flow(
        console,
        repo,
        deposit,
        prompt_amount=lambda: "   ",
        prompt_note=lambda: "updated",
    )

    repo.update.assert_called_once_with(
        deposit.id,
        amount=Decimal("100000"),
        note="updated",
    )


def test_update_deposit_flow_blank_note_keeps_existing_value():
    """Blank note input should keep the current note."""
    console = Console(record=True, width=80)
    repo = MagicMock()
    deposit = _make_deposit(amount=Decimal("100000"), note="existing note")

    update_deposit_flow(
        console,
        repo,
        deposit,
        prompt_amount=lambda: "100001",
        prompt_note=lambda: "   ",
    )

    repo.update.assert_called_once_with(
        deposit.id,
        amount=Decimal("100001"),
    )


def test_update_deposit_flow_clear_note_with_clear_keyword():
    """/clear should explicitly clear note to NULL."""
    console = Console(record=True, width=80)
    repo = MagicMock()
    deposit = _make_deposit(amount=Decimal("100000"), note="existing note")

    update_deposit_flow(
        console,
        repo,
        deposit,
        prompt_amount=lambda: "100001",
        prompt_note=lambda: " /ClEaR ",
    )

    repo.update.assert_called_once_with(
        deposit.id,
        amount=Decimal("100001"),
        note=None,
    )


def test_update_deposit_flow_invalid_amount_keeps_existing_value():
    """Invalid amount input should keep current amount."""
    console = Console(record=True, width=80)
    repo = MagicMock()
    deposit = _make_deposit(amount=Decimal("100000"), note="seed")

    update_deposit_flow(
        console,
        repo,
        deposit,
        prompt_amount=lambda: "abc",
        prompt_note=lambda: "updated",
    )

    repo.update.assert_called_once_with(
        deposit.id,
        amount=Decimal("100000"),
        note="updated",
    )
    output = console.export_text()
    assert "Invalid amount" in output


def test_choose_deposit_from_list_returns_deposit_id():
    """Should return selected deposit id from chooser."""
    deposit = _make_deposit(amount=Decimal("1000"))
    chooser = MagicMock(return_value=deposit.id)

    result = choose_deposit_from_list([deposit], chooser)

    chooser.assert_called_once()
    assert result == deposit.id


def test_delete_deposit_flow_deletes_selected_deposit_with_choice_picker():
    """Delete flow should use choice-based selection and delete chosen deposit."""
    console = Console(record=True, width=80)
    repo = MagicMock()
    deposit = _make_deposit(amount=Decimal("1000"))

    with patch(
        "portfolio_manager.cli.deposits.choose_deposit_from_list",
        return_value=deposit.id,
    ):
        with patch("portfolio_manager.cli.deposits.Confirm.ask", return_value=True):
            delete_deposit_flow(console, repo, [deposit])

    repo.delete.assert_called_once_with(deposit.id)


def test_select_deposit_to_edit_updates_selected_deposit_with_choice_picker():
    """Edit selection should use choice picker and update chosen deposit."""
    console = Console(record=True, width=80)
    repo = MagicMock()
    deposit = _make_deposit(amount=Decimal("1000"))

    with patch(
        "portfolio_manager.cli.deposits.choose_deposit_from_list",
        return_value=deposit.id,
    ):
        with patch("portfolio_manager.cli.deposits.update_deposit_flow") as mock_update:
            select_deposit_to_edit(console, repo, [deposit])

    mock_update.assert_called_once_with(console, repo, deposit)


def test_get_date_input_renders_invalid_format_with_console():
    """Invalid date format should render a console warning and retry."""
    console = Console(record=True, width=80)
    responses = iter(["2026/02/06", "2026-02-06"])

    result = get_date_input(
        prompt_func=lambda: next(responses),
        console=console,
    )

    assert result == date(2026, 2, 6)
    output = console.export_text()
    assert "Invalid format" in output


def test_get_date_input_prints_invalid_format_without_console(capsys):
    """Invalid date format without console should print fallback warning."""
    responses = iter(["2026/02/06", "2026-02-06"])

    result = get_date_input(
        prompt_func=lambda: next(responses),
        console=None,
    )

    assert result == date(2026, 2, 6)
    output = capsys.readouterr().out
    assert "Invalid format" in output


def test_render_deposit_list_shows_empty_message():
    """Empty deposit list should show empty-state text."""
    console = Console(record=True, width=80)

    render_deposit_list(console, [])

    output = console.export_text()
    assert "No deposits found" in output


def test_render_deposit_list_shows_total_row():
    """Deposit list should include total row for non-empty data."""
    console = Console(record=True, width=100)
    deposits = [
        _make_deposit(amount=Decimal("1000")),
        _make_deposit(amount=Decimal("2500")),
    ]

    render_deposit_list(console, deposits)

    output = console.export_text()
    assert "Total" in output
    assert "3,500" in output


def test_add_deposit_flow_cancelled_at_date_prompt():
    """Add flow should abort when date entry is cancelled."""
    console = Console(record=True, width=80)
    repo = MagicMock()

    add_deposit_flow(console, repo, prompt_date=lambda: None)

    repo.create.assert_not_called()
    assert "Cancelled" in console.export_text()


def test_add_deposit_flow_cancelled_at_amount_prompt():
    """Add flow should abort when amount entry is cancelled."""
    console = Console(record=True, width=80)
    repo = MagicMock()
    repo.get_by_date.return_value = None

    add_deposit_flow(
        console,
        repo,
        prompt_date=lambda: "2026-02-06",
        prompt_amount=lambda: None,
    )

    repo.create.assert_not_called()
    assert "Cancelled" in console.export_text()


def test_add_deposit_flow_cancelled_at_note_prompt():
    """Add flow should abort when note entry is cancelled."""
    console = Console(record=True, width=80)
    repo = MagicMock()
    repo.get_by_date.return_value = None

    add_deposit_flow(
        console,
        repo,
        prompt_date=lambda: "2026-02-06",
        prompt_amount=lambda: "1000",
        prompt_note=lambda: None,
    )

    repo.create.assert_not_called()
    assert "Cancelled" in console.export_text()


def test_add_deposit_flow_retries_after_invalid_amount():
    """Invalid amount should be retried until valid value is entered."""
    console = Console(record=True, width=80)
    repo = MagicMock()
    repo.get_by_date.return_value = None
    amount_values = iter(["abc", "1000"])

    add_deposit_flow(
        console,
        repo,
        prompt_date=lambda: "2026-02-06",
        prompt_amount=lambda: next(amount_values),
        prompt_note=lambda: "",
    )

    repo.create.assert_called_once()
    output = console.export_text()
    assert "Invalid amount" in output
    assert "Added deposit" in output


def test_add_deposit_flow_duplicate_date_routes_to_update_when_confirmed():
    """Duplicate date should route to update flow when user confirms."""
    console = Console(record=True, width=80)
    repo = MagicMock()
    existing = _make_deposit(amount=Decimal("1000"), note="existing")
    repo.get_by_date.return_value = existing

    with patch("portfolio_manager.cli.deposits.Confirm.ask", return_value=True):
        with patch("portfolio_manager.cli.deposits.update_deposit_flow") as update_flow:
            add_deposit_flow(console, repo, prompt_date=lambda: "2026-02-06")

    update_flow.assert_called_once_with(console, repo, existing)
    repo.create.assert_not_called()


def test_add_deposit_flow_duplicate_date_does_not_update_when_declined():
    """Duplicate date should return without update when user declines."""
    console = Console(record=True, width=80)
    repo = MagicMock()
    repo.get_by_date.return_value = _make_deposit(
        amount=Decimal("1000"), note="existing"
    )

    with patch("portfolio_manager.cli.deposits.Confirm.ask", return_value=False):
        with patch("portfolio_manager.cli.deposits.update_deposit_flow") as update_flow:
            add_deposit_flow(console, repo, prompt_date=lambda: "2026-02-06")

    update_flow.assert_not_called()
    repo.create.assert_not_called()


def test_delete_deposit_flow_handles_empty_list():
    """Delete flow should show warning when no deposits exist."""
    console = Console(record=True, width=80)
    repo = MagicMock()

    delete_deposit_flow(console, repo, [])

    assert "No deposits to delete" in console.export_text()
    repo.delete.assert_not_called()


def test_delete_deposit_flow_returns_when_selection_cancelled():
    """Delete flow should return when selection is cancelled."""
    console = Console(record=True, width=80)
    repo = MagicMock()
    deposit = _make_deposit(amount=Decimal("1000"))

    with patch(
        "portfolio_manager.cli.deposits.choose_deposit_from_list", return_value=None
    ):
        delete_deposit_flow(console, repo, [deposit])

    repo.delete.assert_not_called()


def test_delete_deposit_flow_returns_when_selected_id_not_found():
    """Delete flow should return when selected id is not present in list."""
    console = Console(record=True, width=80)
    repo = MagicMock()
    deposit = _make_deposit(amount=Decimal("1000"))

    with patch(
        "portfolio_manager.cli.deposits.choose_deposit_from_list", return_value=uuid4()
    ):
        delete_deposit_flow(console, repo, [deposit])

    repo.delete.assert_not_called()


def test_select_deposit_to_edit_handles_empty_list():
    """Edit flow should show warning when no deposits exist."""
    console = Console(record=True, width=80)
    repo = MagicMock()

    select_deposit_to_edit(console, repo, [])

    assert "No deposits to edit" in console.export_text()


def test_select_deposit_to_edit_returns_when_selection_cancelled():
    """Edit flow should return when no deposit is selected."""
    console = Console(record=True, width=80)
    repo = MagicMock()
    deposit = _make_deposit(amount=Decimal("1000"))

    with patch(
        "portfolio_manager.cli.deposits.choose_deposit_from_list", return_value=None
    ):
        with patch("portfolio_manager.cli.deposits.update_deposit_flow") as update_flow:
            select_deposit_to_edit(console, repo, [deposit])

    update_flow.assert_not_called()


def test_run_deposit_menu_dispatches_actions():
    """Menu loop should dispatch add/edit/delete/back actions in sequence."""
    console = Console(record=True, width=80)
    repo = MagicMock()
    deposits = [_make_deposit(amount=Decimal("1000"))]
    repo.list_all.return_value = deposits

    with patch(
        "portfolio_manager.cli.deposits.choose_deposit_menu",
        side_effect=["add", "edit", "delete", "back"],
    ):
        with patch("portfolio_manager.cli.deposits.add_deposit_flow") as add_flow:
            with patch(
                "portfolio_manager.cli.deposits.select_deposit_to_edit"
            ) as edit_flow:
                with patch(
                    "portfolio_manager.cli.deposits.delete_deposit_flow"
                ) as delete_flow:
                    run_deposit_menu(console, repo)

    add_flow.assert_called_once_with(console, repo)
    edit_flow.assert_called_once_with(console, repo, deposits)
    delete_flow.assert_called_once_with(console, repo, deposits)
