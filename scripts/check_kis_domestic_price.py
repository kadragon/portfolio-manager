from __future__ import annotations

import os
import sys
from pathlib import Path

import httpx

SRC_PATH = Path(__file__).resolve().parents[1] / "src"
sys.path.insert(0, str(SRC_PATH))

from portfolio_manager.services.kis.kis_auth_client import (  # noqa: E402
    KisAuthClient,
)
from portfolio_manager.services.kis.kis_domestic_price_client import (  # noqa: E402
    KisDomesticPriceClient,
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
    fid_cond_mrkt_div_code = os.getenv("KIS_FID_COND_MRKT_DIV_CODE", "J")
    fid_input_iscd = os.getenv("KIS_FID_INPUT_ISCD", "005930")

    base_url = "https://openapi.koreainvestment.com:9443"
    if env in {"demo", "vps", "paper"}:
        base_url = "https://openapivts.koreainvestment.com:29443"

    with httpx.Client(base_url=base_url) as client:
        auth = KisAuthClient(client=client, app_key=app_key, app_secret=app_secret)
        store = FileTokenStore(Path(".data/kis_token.json"))
        manager = TokenManager(store=store, auth_client=auth)
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

        price_client = KisDomesticPriceClient(
            client=client,
            app_key=app_key,
            app_secret=app_secret,
            access_token=token,
            cust_type=cust_type,
            env=env,
        )
        try:
            quote = price_client.fetch_current_price(
                fid_cond_mrkt_div_code=fid_cond_mrkt_div_code,
                fid_input_iscd=fid_input_iscd,
            )
        except httpx.HTTPStatusError as exc:
            response = exc.response
            print(f"Price request failed: {response.status_code}")
            try:
                print(response.json())
            except ValueError:
                print(response.text)
            return 1

    print(f"OK: {quote.symbol} {quote.name} {quote.price} ({quote.market})")
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
