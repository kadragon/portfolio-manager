from dataclasses import dataclass

import pytest

from portfolio_manager.services.kis.kis_base_client import KisBaseClient


@dataclass(frozen=True)
class DummyKisClient(KisBaseClient):
    app_key: str
    app_secret: str
    access_token: str
    cust_type: str


def test_base_client_builds_common_headers():
    client = DummyKisClient(
        app_key="app-key",
        app_secret="app-secret",
        access_token="access-token",
        cust_type="P",
    )

    headers = client._build_headers("TR-ID")

    assert headers == {
        "content-type": "application/json",
        "authorization": "Bearer access-token",
        "appkey": "app-key",
        "appsecret": "app-secret",
        "tr_id": "TR-ID",
        "custtype": "P",
    }


def test_base_client_tr_id_mapping_allows_env_variants():
    assert (
        KisBaseClient._tr_id_for_env(" REAL ", real_id="REAL-ID", demo_id="DEMO-ID")
        == "REAL-ID"
    )
    assert (
        KisBaseClient._tr_id_for_env("vps", real_id="REAL-ID", demo_id="DEMO-ID")
        == "DEMO-ID"
    )
    assert (
        KisBaseClient._tr_id_for_env("real/prod", real_id="REAL-ID", demo_id="DEMO-ID")
        == "REAL-ID"
    )

    with pytest.raises(ValueError):
        KisBaseClient._tr_id_for_env("unknown", real_id="REAL-ID", demo_id="DEMO-ID")
