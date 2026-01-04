"""Tests for KIS API token auto-refresh."""

from __future__ import annotations

from unittest.mock import Mock

import httpx

from portfolio_manager.services.kis.kis_domestic_price_client import (
    KisDomesticPriceClient,
)
from portfolio_manager.services.kis.kis_error_handler import is_token_expired_error
from portfolio_manager.services.kis.kis_overseas_price_client import (
    KisOverseasPriceClient,
)


def test_is_token_expired_error_returns_true_for_expired_token():
    """Should detect token expiration error from KIS API response."""
    response = httpx.Response(
        status_code=500,
        json={
            "rt_cd": "1",
            "msg1": "기간이 만료된 token 입니다.",
            "msg_cd": "EGW00123",
        },
        request=httpx.Request("GET", "https://example.com"),
    )

    assert is_token_expired_error(response) is True


def test_is_token_expired_error_returns_false_for_other_errors():
    """Should not detect non-token errors as token expiration."""
    response = httpx.Response(
        status_code=500,
        json={"rt_cd": "1", "msg1": "Other error", "msg_cd": "OTHER123"},
        request=httpx.Request("GET", "https://example.com"),
    )

    assert is_token_expired_error(response) is False


def test_is_token_expired_error_returns_false_for_success():
    """Should return False for successful responses."""
    response = httpx.Response(
        status_code=200,
        json={"rt_cd": "0", "output": {}},
        request=httpx.Request("GET", "https://example.com"),
    )

    assert is_token_expired_error(response) is False


def test_domestic_price_client_retries_on_token_expiration():
    """Should automatically refresh token and retry on token expiration error."""
    # Mock client that returns token error first, then success
    mock_client = Mock(spec=httpx.Client)
    expired_response = httpx.Response(
        status_code=500,
        json={
            "rt_cd": "1",
            "msg1": "기간이 만료된 token 입니다.",
            "msg_cd": "EGW00123",
        },
        request=httpx.Request("GET", "https://example.com"),
    )
    success_response = httpx.Response(
        status_code=200,
        json={
            "rt_cd": "0",
            "output": {
                "stck_prpr": "128500",
                "hts_kor_isnm": "삼성전자",
            },
        },
        request=httpx.Request("GET", "https://example.com"),
    )
    mock_client.get.side_effect = [expired_response, success_response]

    # Mock token manager
    mock_token_manager = Mock()
    mock_token_manager.get_token.return_value = "new_token_after_refresh"

    client = KisDomesticPriceClient(
        client=mock_client,
        app_key="test_key",
        app_secret="test_secret",
        access_token="old_token",
        cust_type="P",
        env="real",
    )

    # This should succeed after retry
    quote = client.fetch_current_price_with_retry(
        fid_cond_mrkt_div_code="J",
        fid_input_iscd="005930",
        token_manager=mock_token_manager,
    )

    assert quote.symbol == "005930"
    assert quote.price == 128500
    assert mock_client.get.call_count == 2
    assert mock_token_manager.get_token.call_count == 1


def test_domestic_price_client_with_token_manager_auto_retries():
    """Should use token_manager for automatic retry when provided at init."""
    # Mock client that returns token error first, then success
    mock_client = Mock(spec=httpx.Client)
    expired_response = httpx.Response(
        status_code=500,
        json={
            "rt_cd": "1",
            "msg1": "기간이 만료된 token 입니다.",
            "msg_cd": "EGW00123",
        },
        request=httpx.Request("GET", "https://example.com"),
    )
    success_response = httpx.Response(
        status_code=200,
        json={
            "rt_cd": "0",
            "output": {
                "stck_prpr": "128500",
                "hts_kor_isnm": "삼성전자",
            },
        },
        request=httpx.Request("GET", "https://example.com"),
    )
    mock_client.get.side_effect = [expired_response, success_response]

    # Mock token manager
    mock_token_manager = Mock()
    mock_token_manager.get_token.return_value = "new_token_after_refresh"

    # Client with token_manager should auto-retry
    client = KisDomesticPriceClient(
        client=mock_client,
        app_key="test_key",
        app_secret="test_secret",
        access_token="old_token",
        cust_type="P",
        env="real",
        token_manager=mock_token_manager,
    )

    # Regular fetch_current_price should now auto-retry
    quote = client.fetch_current_price("J", "005930")

    assert quote.symbol == "005930"
    assert quote.price == 128500
    assert mock_client.get.call_count == 2
    assert mock_token_manager.get_token.call_count == 1


def test_overseas_price_client_with_token_manager_auto_retries():
    """Should use token_manager for automatic retry when provided at init."""
    # Mock client that returns token error first, then success
    mock_client = Mock(spec=httpx.Client)
    expired_response = httpx.Response(
        status_code=500,
        json={
            "rt_cd": "1",
            "msg1": "기간이 만료된 token 입니다.",
            "msg_cd": "EGW00123",
        },
        request=httpx.Request("GET", "https://example.com"),
    )
    success_response = httpx.Response(
        status_code=200,
        json={
            "rt_cd": "0",
            "output": {
                "last": "150.25",
                "symb": "AAPL",
                "name": "Apple Inc",
            },
        },
        request=httpx.Request("GET", "https://example.com"),
    )
    mock_client.get.side_effect = [expired_response, success_response]

    # Mock token manager
    mock_token_manager = Mock()
    mock_token_manager.get_token.return_value = "new_token_after_refresh"

    # Client with token_manager should auto-retry
    client = KisOverseasPriceClient(
        client=mock_client,
        app_key="test_key",
        app_secret="test_secret",
        access_token="old_token",
        cust_type="P",
        env="real",
        token_manager=mock_token_manager,
    )

    # Regular fetch_current_price should now auto-retry
    quote = client.fetch_current_price("NAS", "AAPL")

    assert quote.symbol == "AAPL"
    assert quote.price == 150.25
    assert mock_client.get.call_count == 2
    assert mock_token_manager.get_token.call_count == 1
