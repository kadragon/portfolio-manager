from __future__ import annotations

from dataclasses import dataclass


@dataclass(frozen=True)
class PriceQuote:
    symbol: str
    name: str
    price: float
    market: str
    currency: str


def parse_korea_price(payload: dict) -> PriceQuote:
    output = payload["output"]
    return PriceQuote(
        symbol=output["stck_code"],
        name=output["hts_kor_isnm"],
        price=int(output["stck_prpr"]),
        market="KR",
        currency="KRW",
    )


def parse_us_price(payload: dict) -> PriceQuote:
    output = payload["output"]
    return PriceQuote(
        symbol=output["symbol"],
        name=output["name"],
        price=float(output["last"]),
        market="US",
        currency="USD",
    )
