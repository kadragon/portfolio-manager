"""Tests for KIS unified price client market detection."""

from unittest.mock import MagicMock

import pytest

from portfolio_manager.services.kis_price_parser import PriceQuote
from portfolio_manager.services.kis_unified_price_client import KisUnifiedPriceClient


@pytest.fixture
def mock_domestic_client():
    client = MagicMock()
    client.fetch_current_price.return_value = PriceQuote(
        symbol="005930", name="삼성전자", price=70000, market="KR"
    )
    return client


@pytest.fixture
def mock_overseas_client():
    client = MagicMock()
    client.fetch_current_price.return_value = PriceQuote(
        symbol="AAPL", name="Apple Inc.", price=150.0, market="US"
    )
    return client


def test_detects_6_digit_numeric_ticker_as_domestic(
    mock_domestic_client, mock_overseas_client
):
    """6자리 숫자 티커는 국내 주식으로 처리한다."""
    unified = KisUnifiedPriceClient(mock_domestic_client, mock_overseas_client)

    unified.get_price("005930")

    mock_domestic_client.fetch_current_price.assert_called_once_with("J", "005930")
    mock_overseas_client.fetch_current_price.assert_not_called()


def test_detects_alphabetic_ticker_as_overseas(
    mock_domestic_client, mock_overseas_client
):
    """알파벳 티커는 해외 주식으로 처리한다."""
    unified = KisUnifiedPriceClient(mock_domestic_client, mock_overseas_client)

    unified.get_price("AAPL")

    mock_overseas_client.fetch_current_price.assert_called_once_with("NAS", "AAPL")
    mock_domestic_client.fetch_current_price.assert_not_called()


def test_detects_6_digit_alphanumeric_ticker_as_domestic(
    mock_domestic_client, mock_overseas_client
):
    """6자리 영숫자 혼합 티커는 국내 주식으로 처리한다 (예: 0052D0)."""
    unified = KisUnifiedPriceClient(mock_domestic_client, mock_overseas_client)

    unified.get_price("0052D0")

    mock_domestic_client.fetch_current_price.assert_called_once_with("J", "0052D0")
    mock_overseas_client.fetch_current_price.assert_not_called()
