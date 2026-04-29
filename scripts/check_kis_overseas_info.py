from __future__ import annotations

import json
import os
import sys
from pathlib import Path

import httpx

SRC_PATH = Path(__file__).resolve().parents[1] / "src"
sys.path.insert(0, str(SRC_PATH))

from portfolio_manager.services.kis.kis_auth_client import KisAuthClient  # noqa: E402
from portfolio_manager.services.kis.kis_overseas_info_client import (  # noqa: E402
    KisOverseasInfoClient,
)
from portfolio_manager.services.kis.kis_token_manager import TokenManager  # noqa: E402
from portfolio_manager.services.kis.kis_token_store import FileTokenStore  # noqa: E402


def load_dotenv(path: Path) -> None:
    if not path.exists():
        return
    for line in path.read_text(encoding="utf-8").splitlines():
        stripped = line.strip()
        if not stripped or stripped.startswith("#"):
            continue
        if "=" not in stripped:
            continue
        key, value = stripped.split("=", 1)
        key = key.strip()
        value = value.strip().strip('"').strip("'")
        os.environ.setdefault(key, value)


def main() -> int:
    load_dotenv(Path(".env"))

    app_key = os.getenv("KIS_APP_KEY")
    app_secret = os.getenv("KIS_APP_SECRET")
    if not app_key or not app_secret:
        print("Missing KIS_APP_KEY or KIS_APP_SECRET in .env")
        return 1

    env = os.getenv("KIS_ENV", "real").strip().lower()
    if "/" in env:
        env = env.split("/", 1)[0]
    cust_type = os.getenv("KIS_CUST_TYPE", "P")
    excd = os.getenv("KIS_OVERSEAS_EXCD", "NAS")
    symb = os.getenv("KIS_OVERSEAS_SYMB", "AAPL")
    tr_id = os.getenv("KIS_OVERSEAS_INFO_TR_ID", "CTPF1702R")

    base_url = "https://openapi.koreainvestment.com:9443"
    if env in {"demo", "vps", "paper"}:
        base_url = "https://openapivts.koreainvestment.com:29443"

    with httpx.Client(base_url=base_url) as client:
        auth_client = KisAuthClient(
            client=client, app_key=app_key, app_secret=app_secret
        )
        store = FileTokenStore(Path(".data/kis_token.json"))
        manager = TokenManager(store=store, auth_client=auth_client)
        try:
            token = manager.get_token()
        except httpx.HTTPStatusError as exc:
            response = exc.response
            print(f"Token request failed: {response.status_code}")
            try:
                print(response.json())
            except ValueError:
                print(response.text)
            return 1

        info_client = KisOverseasInfoClient(
            client=client,
            app_key=app_key,
            app_secret=app_secret,
            access_token=token,
            tr_id=tr_id,
            cust_type=cust_type,
        )

        # Also capture raw response for field inspection
        raw_response = client.get(
            "/uapi/overseas-price/v1/quotations/search-info",
            params={
                "PRDT_TYPE_CD": {"NAS": "512", "NYS": "513", "AMS": "529"}.get(
                    excd.upper(), "512"
                ),
                "PDNO": symb,
            },
            headers={
                "content-type": "application/json",
                "authorization": f"Bearer {token}",
                "appkey": app_key,
                "appsecret": app_secret,
                "tr_id": tr_id,
                "custtype": cust_type,
            },
        )
        print(f"Raw response ({raw_response.status_code}):")
        try:
            print(json.dumps(raw_response.json(), ensure_ascii=False, indent=2))
        except ValueError:
            print(raw_response.text)

        try:
            info = info_client.fetch_basic_info(excd=excd, symb=symb)
            print(f"\nOK: {info.pdno} | excd={info.excd} | name={info.name!r}")
        except Exception as e:
            print(f"\nfetch_basic_info failed: {e}")
            return 1

    return 0


if __name__ == "__main__":
    raise SystemExit(main())
