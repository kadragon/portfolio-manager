from __future__ import annotations

import os
import sys
from datetime import date, datetime, timedelta
from pathlib import Path

SRC_PATH = Path(__file__).resolve().parents[1] / "src"
sys.path.insert(0, str(SRC_PATH))

import httpx

from portfolio_manager.services.exim_exchange_rate_client import EximExchangeRateClient


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


def previous_business_day(current: date) -> date:
    weekday = current.weekday()
    if weekday == 6:
        return current - timedelta(days=2)
    if weekday == 5:
        return current - timedelta(days=1)
    return current - timedelta(days=1)


def main() -> int:
    load_dotenv(Path(".env"))

    auth_key = os.getenv("EXIM_AUTH_KEY")
    if not auth_key:
        print("Missing EXIM_AUTH_KEY in .env")
        return 1

    search_date = os.getenv("EXIM_SEARCH_DATE")
    search_date_from_env = bool(search_date)
    if not search_date:
        search_date = datetime.now().strftime("%Y%m%d")

    with httpx.Client(base_url="https://oapi.koreaexim.go.kr") as client:
        exim = EximExchangeRateClient(client=client, auth_key=auth_key)
        try:
            rate = exim.fetch_usd_rate(search_date=search_date)
        except ValueError as exc:
            if not search_date_from_env:
                current = datetime.strptime(search_date, "%Y%m%d").date()
                fallback = previous_business_day(current)
                fallback_str = fallback.strftime("%Y%m%d")
                try:
                    rate = exim.fetch_usd_rate(search_date=fallback_str)
                    print(
                        "USD rate not found for today; "
                        f"retried previous business day ({fallback_str})."
                    )
                    print(f"OK: USD {rate} (search_date={fallback_str})")
                    return 0
                except ValueError:
                    pass
            print(str(exc))
            return 1
        except httpx.HTTPStatusError as exc:
            response = exc.response
            print(f"Request failed: {response.status_code}")
            try:
                print(response.json())
            except ValueError:
                print(response.text)
            return 1

    print(f"OK: USD {rate} (search_date={search_date})")
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
