"""KIS API error handling utilities."""

from __future__ import annotations

import httpx


def is_token_expired_error(response: httpx.Response) -> bool:
    """Check if the response indicates a token expiration error.

    Args:
        response: The HTTP response from KIS API

    Returns:
        True if the error is due to token expiration, False otherwise
    """
    if response.status_code != 500:
        return False

    try:
        data = response.json()
        return data.get("msg_cd") == "EGW00123"
    except Exception:
        return False
