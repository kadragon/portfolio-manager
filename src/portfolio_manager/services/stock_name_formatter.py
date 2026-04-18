"""Shared utilities for formatting stock display names."""

_ETF_SUFFIX = "증권상장지수투자신탁(주식)"


def format_stock_name(name: str) -> str:
    return name.replace(_ETF_SUFFIX, "").strip()
