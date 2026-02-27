"""Tests for RebalanceService v2.0."""

from datetime import datetime, timezone
from decimal import Decimal
from uuid import uuid4

import pytest

from portfolio_manager.models import Account, Group, Holding, Stock
from portfolio_manager.services.portfolio_service import (
    PortfolioSummary,
    StockHoldingWithPrice,
)
from portfolio_manager.services.rebalance_service import (
    RebalancePlan,
    RebalanceService,
    RegionDiagnostic,
    SleeveDiagnostic,
)


def make_group(name: str, target_percentage: float) -> Group:
    now = datetime.now(timezone.utc)
    return Group(
        id=uuid4(),
        name=name,
        created_at=now,
        updated_at=now,
        target_percentage=target_percentage,
    )


def make_stock(ticker: str, group_id) -> Stock:
    now = datetime.now(timezone.utc)
    return Stock(
        id=uuid4(),
        ticker=ticker,
        group_id=group_id,
        created_at=now,
        updated_at=now,
    )


def make_account(name: str, cash_balance: Decimal) -> Account:
    now = datetime.now(timezone.utc)
    return Account(
        id=uuid4(),
        name=name,
        cash_balance=cash_balance,
        created_at=now,
        updated_at=now,
    )


def make_holding(account_id, stock_id, quantity: Decimal) -> Holding:
    now = datetime.now(timezone.utc)
    return Holding(
        id=uuid4(),
        account_id=account_id,
        stock_id=stock_id,
        quantity=quantity,
        created_at=now,
        updated_at=now,
    )


def make_standard_groups() -> list[Group]:
    return [
        make_group("국내성장", 35.0),
        make_group("국내배당", 15.0),
        make_group("해외성장", 25.0),
        make_group("해외안정", 10.0),
        make_group("해외배당", 15.0),
    ]


def make_standard_stocks(groups: list[Group]) -> dict[str, Stock]:
    group_by_name = {group.name: group for group in groups}
    return {
        "국내성장": make_stock("005930", group_by_name["국내성장"].id),
        "국내배당": make_stock("000660", group_by_name["국내배당"].id),
        "해외성장": make_stock("QQQ", group_by_name["해외성장"].id),
        "해외안정": make_stock("VOO", group_by_name["해외안정"].id),
        "해외배당": make_stock("SCHD", group_by_name["해외배당"].id),
    }


def make_summary(
    groups: list[Group],
    stocks_by_sleeve: dict[str, Stock],
    sleeve_values: dict[str, Decimal],
) -> PortfolioSummary:
    group_by_name = {group.name: group for group in groups}
    holdings: list[tuple[Group, StockHoldingWithPrice]] = []
    total_value = Decimal("0")

    for sleeve_name, value in sleeve_values.items():
        stock = stocks_by_sleeve[sleeve_name]
        currency = "KRW" if sleeve_name.startswith("국내") else "USD"
        holding = StockHoldingWithPrice(
            stock=stock,
            quantity=value,
            price=Decimal("1"),
            currency=currency,
            name=stock.ticker,
            value_krw=value,
        )
        holdings.append((group_by_name[sleeve_name], holding))
        total_value += value

    return PortfolioSummary(
        holdings=holdings,
        total_value=total_value,
        total_assets=total_value,
    )


def make_holdings_by_account(
    accounts: list[Account],
    stocks_by_sleeve: dict[str, Stock],
    per_account_sleeve_values: dict[str, dict[str, Decimal]],
) -> dict:
    account_by_name = {account.name: account for account in accounts}
    result = {}
    for account_name, sleeve_values in per_account_sleeve_values.items():
        account = account_by_name[account_name]
        result[account.id] = [
            make_holding(
                account_id=account.id,
                stock_id=stocks_by_sleeve[sleeve_name].id,
                quantity=quantity,
            )
            for sleeve_name, quantity in sleeve_values.items()
            if quantity > 0
        ]
    return result


def test_rebalance_plan_accepts_legacy_sleeve_diagnostics_only() -> None:
    plan = RebalancePlan(
        sell_recommendations=[],
        buy_recommendations=[],
        sleeve_diagnostics=[
            SleeveDiagnostic(
                sleeve_name="국내성장",
                target_percentage=Decimal("35"),
                band_percentage=Decimal("5"),
                lower_percentage=Decimal("30"),
                upper_percentage=Decimal("40"),
                current_percentage=Decimal("41"),
                current_value_krw=Decimal("410"),
                is_upper_breached=True,
                is_lower_breached=False,
            )
        ],
        region_diagnostic=RegionDiagnostic(
            target_kr_percentage=Decimal("50"),
            target_us_percentage=Decimal("50"),
            current_kr_percentage=Decimal("50"),
            current_us_percentage=Decimal("50"),
            lower_kr_percentage=Decimal("45"),
            upper_kr_percentage=Decimal("55"),
            is_triggered=False,
        ),
        total_assets_krw=Decimal("1000"),
    )

    assert plan.group_diagnostics is not None
    assert len(plan.group_diagnostics) == 1
    assert plan.group_diagnostics[0].rebalance_group_name == "국내성장"


def test_build_plan_raises_for_unmapped_group_name() -> None:
    group = make_group("국내 주식", 100.0)
    stock = make_stock("005930", group.id)
    summary = PortfolioSummary(
        holdings=[
            (
                group,
                StockHoldingWithPrice(
                    stock=stock,
                    quantity=Decimal("100"),
                    price=Decimal("1"),
                    currency="KRW",
                    name="삼성전자",
                    value_krw=Decimal("100"),
                ),
            )
        ],
        total_value=Decimal("100"),
        total_assets=Decimal("100"),
    )

    service = RebalanceService()
    with pytest.raises(ValueError, match="리밸런싱 그룹 매핑 불가 그룹"):
        service.build_plan(
            summary=summary,
            accounts=[],
            holdings_by_account={},
            groups=[group],
            stocks=[stock],
        )


def test_build_plan_flags_upper_and_lower_band_breaches() -> None:
    groups = make_standard_groups()
    stocks = make_standard_stocks(groups)
    summary = make_summary(
        groups,
        stocks,
        {
            "국내성장": Decimal("500"),
            "국내배당": Decimal("120"),
            "해외성장": Decimal("240"),
            "해외안정": Decimal("100"),
            "해외배당": Decimal("40"),
        },
    )
    account = make_account("A", Decimal("0"))
    holdings_by_account = make_holdings_by_account(
        [account],
        stocks,
        {
            "A": {
                "국내성장": Decimal("500"),
                "국내배당": Decimal("120"),
                "해외성장": Decimal("240"),
                "해외안정": Decimal("100"),
                "해외배당": Decimal("40"),
            }
        },
    )

    service = RebalanceService()
    plan = service.build_plan(
        summary=summary,
        accounts=[account],
        holdings_by_account=holdings_by_account,
        groups=groups,
        stocks=list(stocks.values()),
    )

    assert plan.group_diagnostics is not None
    diag = {item.rebalance_group_name: item for item in plan.group_diagnostics}
    assert diag["국내성장"].is_upper_breached is True
    assert diag["해외배당"].is_lower_breached is True
    assert plan.sleeve_diagnostics is not None
    assert len(plan.sleeve_diagnostics) == len(plan.group_diagnostics)
    assert plan.sleeve_diagnostics[0].sleeve_name == "국내성장"


def test_build_plan_region_trigger_uses_sleeve_target_sum() -> None:
    groups = make_standard_groups()
    stocks = make_standard_stocks(groups)
    summary = make_summary(
        groups,
        stocks,
        {
            "국내성장": Decimal("400"),
            "국내배당": Decimal("200"),
            "해외성장": Decimal("180"),
            "해외안정": Decimal("120"),
            "해외배당": Decimal("100"),
        },
    )
    account = make_account("A", Decimal("0"))
    holdings_by_account = make_holdings_by_account(
        [account],
        stocks,
        {
            "A": {
                "국내성장": Decimal("400"),
                "국내배당": Decimal("200"),
                "해외성장": Decimal("180"),
                "해외안정": Decimal("120"),
                "해외배당": Decimal("100"),
            }
        },
    )

    service = RebalanceService()
    plan = service.build_plan(
        summary=summary,
        accounts=[account],
        holdings_by_account=holdings_by_account,
        groups=groups,
        stocks=list(stocks.values()),
    )

    assert plan.region_diagnostic.target_kr_percentage == Decimal("50")
    assert plan.region_diagnostic.current_kr_percentage == Decimal("60")
    assert plan.region_diagnostic.is_triggered is True


def test_build_plan_applies_half_rule_sell_with_safety_cap() -> None:
    groups = make_standard_groups()
    stocks = make_standard_stocks(groups)
    summary = make_summary(
        groups,
        stocks,
        {
            "국내성장": Decimal("500"),
            "국내배당": Decimal("100"),
            "해외성장": Decimal("200"),
            "해외안정": Decimal("100"),
            "해외배당": Decimal("100"),
        },
    )
    accounts = [make_account("A", Decimal("0")), make_account("B", Decimal("0"))]
    holdings_by_account = make_holdings_by_account(
        accounts,
        stocks,
        {
            "A": {
                "국내성장": Decimal("300"),
                "국내배당": Decimal("100"),
                "해외성장": Decimal("200"),
                "해외안정": Decimal("100"),
                "해외배당": Decimal("100"),
            },
            "B": {
                "국내성장": Decimal("200"),
            },
        },
    )

    service = RebalanceService()
    plan = service.build_plan(
        summary=summary,
        accounts=accounts,
        holdings_by_account=holdings_by_account,
        groups=groups,
        stocks=list(stocks.values()),
    )

    sell_total = sum(
        (rec.amount_krw or Decimal("0")) for rec in plan.sell_recommendations
    )
    by_account = {
        name: sum(
            (rec.amount_krw or Decimal("0"))
            for rec in plan.sell_recommendations
            if rec.account_name == name
        )
        for name in ("A", "B")
    }

    # 국내성장 50% -> 목표35%, 상단40%: sell 10% of total assets = 100
    assert sell_total == Decimal("100")
    assert by_account["A"] == Decimal("60")
    assert by_account["B"] == Decimal("40")


def test_build_plan_sell_allocation_uses_fixed_account_total_denominator() -> None:
    groups = make_standard_groups()
    group_by_name = {group.name: group for group in groups}

    stock_growth_a = make_stock("100001", group_by_name["국내성장"].id)
    stock_growth_b = make_stock("100002", group_by_name["국내성장"].id)
    stock_growth_c = make_stock("100003", group_by_name["국내성장"].id)
    stock_kr_div = make_stock("200001", group_by_name["국내배당"].id)
    stock_us_growth = make_stock("QQQ", group_by_name["해외성장"].id)
    stock_us_stable = make_stock("VOO", group_by_name["해외안정"].id)
    stock_us_div = make_stock("SCHD", group_by_name["해외배당"].id)

    summary = PortfolioSummary(
        holdings=[
            (
                group_by_name["국내성장"],
                StockHoldingWithPrice(
                    stock=stock_growth_a,
                    quantity=Decimal("250"),
                    price=Decimal("1"),
                    currency="KRW",
                    name=stock_growth_a.ticker,
                    value_krw=Decimal("250"),
                ),
            ),
            (
                group_by_name["국내성장"],
                StockHoldingWithPrice(
                    stock=stock_growth_b,
                    quantity=Decimal("150"),
                    price=Decimal("1"),
                    currency="KRW",
                    name=stock_growth_b.ticker,
                    value_krw=Decimal("150"),
                ),
            ),
            (
                group_by_name["국내성장"],
                StockHoldingWithPrice(
                    stock=stock_growth_c,
                    quantity=Decimal("100"),
                    price=Decimal("1"),
                    currency="KRW",
                    name=stock_growth_c.ticker,
                    value_krw=Decimal("100"),
                ),
            ),
            (
                group_by_name["국내배당"],
                StockHoldingWithPrice(
                    stock=stock_kr_div,
                    quantity=Decimal("100"),
                    price=Decimal("1"),
                    currency="KRW",
                    name=stock_kr_div.ticker,
                    value_krw=Decimal("100"),
                ),
            ),
            (
                group_by_name["해외성장"],
                StockHoldingWithPrice(
                    stock=stock_us_growth,
                    quantity=Decimal("200"),
                    price=Decimal("1"),
                    currency="USD",
                    name=stock_us_growth.ticker,
                    value_krw=Decimal("200"),
                ),
            ),
            (
                group_by_name["해외안정"],
                StockHoldingWithPrice(
                    stock=stock_us_stable,
                    quantity=Decimal("100"),
                    price=Decimal("1"),
                    currency="USD",
                    name=stock_us_stable.ticker,
                    value_krw=Decimal("100"),
                ),
            ),
            (
                group_by_name["해외배당"],
                StockHoldingWithPrice(
                    stock=stock_us_div,
                    quantity=Decimal("100"),
                    price=Decimal("1"),
                    currency="USD",
                    name=stock_us_div.ticker,
                    value_krw=Decimal("100"),
                ),
            ),
        ],
        total_value=Decimal("1000"),
        total_assets=Decimal("1000"),
    )

    account = make_account("A", Decimal("0"))
    holdings_by_account = {
        account.id: [
            make_holding(account.id, stock_growth_a.id, Decimal("250")),
            make_holding(account.id, stock_growth_b.id, Decimal("150")),
            make_holding(account.id, stock_growth_c.id, Decimal("100")),
            make_holding(account.id, stock_kr_div.id, Decimal("100")),
            make_holding(account.id, stock_us_growth.id, Decimal("200")),
            make_holding(account.id, stock_us_stable.id, Decimal("100")),
            make_holding(account.id, stock_us_div.id, Decimal("100")),
        ]
    }

    service = RebalanceService()
    plan = service.build_plan(
        summary=summary,
        accounts=[account],
        holdings_by_account=holdings_by_account,
        groups=groups,
        stocks=[
            stock_growth_a,
            stock_growth_b,
            stock_growth_c,
            stock_kr_div,
            stock_us_growth,
            stock_us_stable,
            stock_us_div,
        ],
    )

    growth_sells = [
        rec
        for rec in plan.sell_recommendations
        if rec.sleeve_name == "국내성장" and rec.account_name == "A"
    ]
    assert len(growth_sells) == 3

    amount_by_ticker = {rec.ticker: rec.amount_krw for rec in growth_sells}
    assert amount_by_ticker["100001"] == Decimal("50")
    assert amount_by_ticker["100002"] == Decimal("30")
    assert amount_by_ticker["100003"] == Decimal("20")


def test_build_plan_skips_sell_when_only_lower_breaches_exist() -> None:
    groups = make_standard_groups()
    stocks = make_standard_stocks(groups)
    summary = make_summary(
        groups,
        stocks,
        {
            "국내성장": Decimal("380"),
            "국내배당": Decimal("160"),
            "해외성장": Decimal("260"),
            "해외안정": Decimal("100"),
            "해외배당": Decimal("100"),
        },
    )
    account = make_account("A", Decimal("50"))
    holdings_by_account = make_holdings_by_account(
        [account],
        stocks,
        {
            "A": {
                "국내성장": Decimal("380"),
                "국내배당": Decimal("160"),
                "해외성장": Decimal("260"),
                "해외안정": Decimal("100"),
                "해외배당": Decimal("100"),
            }
        },
    )

    service = RebalanceService()
    plan = service.build_plan(
        summary=summary,
        accounts=[account],
        holdings_by_account=holdings_by_account,
        groups=groups,
        stocks=list(stocks.values()),
    )

    # total = 1000; 해외배당 = 10% vs target 15% (lower 12%) → lower breach
    assert plan.sell_recommendations == []
    assert len(plan.buy_recommendations) == 1
    buy = plan.buy_recommendations[0]
    assert buy.sleeve_name == "해외배당"
    assert buy.rebalance_group_name == "해외배당"
    assert buy.ticker == "SCHD"
    assert buy.amount_krw == Decimal("50")


def test_build_plan_reinvests_cash_only_within_same_account() -> None:
    groups = make_standard_groups()
    stocks = make_standard_stocks(groups)
    summary = make_summary(
        groups,
        stocks,
        {
            "국내성장": Decimal("500"),
            "국내배당": Decimal("150"),
            "해외성장": Decimal("150"),
            "해외안정": Decimal("100"),
            "해외배당": Decimal("100"),
        },
    )

    accounts = [make_account("A", Decimal("0")), make_account("B", Decimal("0"))]
    holdings_by_account = make_holdings_by_account(
        accounts,
        stocks,
        {
            "A": {
                "국내성장": Decimal("500"),
                "해외성장": Decimal("20"),
            },
            "B": {
                "국내배당": Decimal("150"),
                "해외성장": Decimal("130"),
                "해외안정": Decimal("100"),
                "해외배당": Decimal("100"),
            },
        },
    )

    service = RebalanceService()
    plan = service.build_plan(
        summary=summary,
        accounts=accounts,
        holdings_by_account=holdings_by_account,
        groups=groups,
        stocks=list(stocks.values()),
    )

    assert plan.sell_recommendations
    assert all(rec.account_name == "A" for rec in plan.sell_recommendations)
    assert plan.buy_recommendations
    assert all(rec.account_name == "A" for rec in plan.buy_recommendations)
