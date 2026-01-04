import pytest

from portfolio_manager.services.kis.kis_market_detector import is_domestic_ticker


@pytest.mark.parametrize(
    "ticker, expected",
    [
        ("005930", True),
        ("AAPL", False),
    ],
)
def test_is_domestic_ticker_by_length(ticker, expected):
    assert is_domestic_ticker(ticker) is expected
