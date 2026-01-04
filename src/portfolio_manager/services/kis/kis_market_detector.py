"""Market detection helpers for KIS tickers."""


def is_domestic_ticker(ticker: str) -> bool:
    """Return True when ticker should be treated as a domestic stock."""
    return len(ticker) == 6
