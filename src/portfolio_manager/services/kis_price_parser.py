from __future__ import annotations

from dataclasses import dataclass


@dataclass(frozen=True)
class PriceQuote:
    symbol: str
    name: str
    price: int
    market: str


def parse_korea_price(payload: dict) -> PriceQuote:
    output = payload["output"]
    return PriceQuote(
        symbol=output["stck_code"],
        name=output["hts_kor_isnm"],
        price=int(output["stck_prpr"]),
        market="KR",
    )


def parse_us_price(payload: dict) -> PriceQuote:
    output = payload["output"]
    return PriceQuote(
        symbol=output["symbol"],
        name=output["name"],
        price=float(output["last"]),
        market="US",
    )
