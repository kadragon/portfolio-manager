"""Tests for stock_name_formatter."""

from portfolio_manager.services.stock_name_formatter import format_stock_name


def test_strips_etf_suffix():
    assert format_stock_name("KODEX 200증권상장지수투자신탁(주식)") == "KODEX 200"


def test_strips_suffix_with_trailing_whitespace():
    assert (
        format_stock_name("TIGER 미국S&P500증권상장지수투자신탁(주식) ")
        == "TIGER 미국S&P500"
    )


def test_passthrough_when_no_suffix():
    assert format_stock_name("삼성전자") == "삼성전자"


def test_empty_string_returns_empty():
    assert format_stock_name("") == ""


def test_suffix_only_returns_empty():
    assert format_stock_name("증권상장지수투자신탁(주식)") == ""
