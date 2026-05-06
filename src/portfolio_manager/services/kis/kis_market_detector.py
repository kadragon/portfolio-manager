"""Market detection helpers for KIS tickers."""


def is_domestic_ticker(ticker: str) -> bool:
    """Return True when ticker should be treated as a domestic stock.

    KOSPI/KOSDAQ codes are exactly 6 numeric characters (e.g. "005930").
    Overseas tickers (e.g. "AAPL", "TSLA") are typically 1–5 alphabetic
    characters.  A 6-character overseas ticker (e.g. a 6-char ADR symbol)
    would be misclassified as domestic — that case does not arise in the
    current KIS integration.
    """
    return len(ticker) == 6
