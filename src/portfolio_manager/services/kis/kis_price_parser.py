from __future__ import annotations

from dataclasses import dataclass


@dataclass(frozen=True)
class PriceQuote:
    symbol: str
    name: str
    price: float
    market: str
    currency: str
    exchange: str | None = None


def parse_korea_price(payload: dict, *, symbol: str | None = None) -> PriceQuote:
    output = payload.get("output") or {}
    if isinstance(output, list):
        output = output[0] if output else {}
    resolved_symbol = output.get("stck_code") or symbol or ""
    return PriceQuote(
        symbol=resolved_symbol,
        name=output.get("hts_kor_isnm", ""),
        price=int(output.get("stck_prpr") or 0),
        market="KR",
        currency="KRW",
        exchange=None,
    )


def parse_us_price(
    payload: dict,
    *,
    symbol: str | None = None,
    exchange: str | None = None,
) -> PriceQuote:
    output = payload.get("output") or {}
    if isinstance(output, list):
        output = output[0] if output else {}
    name = ""
    for key in (
        "name",
        "enname",
        "ename",
        "en_name",
        "symb_name",
        "symbol_name",
        "prdt_name",
        "product_name",
        "item_name",
    ):
        value = output.get(key)
        if isinstance(value, str) and value.strip():
            name = value.strip()
            break
    resolved_symbol = output.get("symbol") or output.get("symb") or output.get("rsym")
    if not resolved_symbol:
        resolved_symbol = symbol or ""
    raw_last = (output.get("last") or "").strip()
    price = float(raw_last) if raw_last else 0.0
    return PriceQuote(
        symbol=resolved_symbol,
        name=name,
        price=float(price),
        market="US",
        currency="USD",
        exchange=exchange,
    )
