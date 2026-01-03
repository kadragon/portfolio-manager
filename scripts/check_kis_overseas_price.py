from __future__ import annotations

import os
import sys
from pathlib import Path

SRC_PATH = Path(__file__).resolve().parents[1] / "src"
sys.path.insert(0, str(SRC_PATH))

import httpx

from portfolio_manager.services.kis_auth_client import KisAuthClient
from portfolio_manager.services.kis_overseas_price_client import KisOverseasPriceClient
from portfolio_manager.services.kis_token_manager import TokenManager
from portfolio_manager.services.kis_token_store import FileTokenStore


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
    auth = os.getenv("KIS_OVERSEAS_AUTH", "")

    base_url = "https://openapi.koreainvestment.com:9443"
    if env in {"demo", "vps", "paper"}:
        base_url = "https://openapivts.koreainvestment.com:29443"

    with httpx.Client(base_url=base_url) as client:
        auth_client = KisAuthClient(client=client, app_key=app_key, app_secret=app_secret)
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

        price_client = KisOverseasPriceClient(
            client=client,
            app_key=app_key,
            app_secret=app_secret,
            access_token=token,
            cust_type=cust_type,
            env=env,
        )
        quote = price_client.fetch_current_price(excd=excd, symb=symb, auth=auth)

    print(f"OK: {quote.symbol} {quote.name} {quote.price} ({quote.market})")
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
