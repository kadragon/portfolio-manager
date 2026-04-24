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
    # A is fully within all bands (AUM=790). B holds only 국내성장 (AUM=200, 100% → far above upper 40%).
    # Only B should trigger a sell via the half-rule with safety cap.
    groups = make_standard_groups()
    stocks = make_standard_stocks(groups)
    # 해외안정 set to 90 so A's 해외안정 % = 90/790 ≈ 11.4% < upper 12%.
    summary = make_summary(
        groups,
        stocks,
        {
            "국내성장": Decimal("500"),
            "국내배당": Decimal("100"),
            "해외성장": Decimal("200"),
            "해외안정": Decimal("90"),
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
                "해외안정": Decimal("90"),
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

    by_account = {
        name: sum(
            (rec.amount_krw or Decimal("0"))
            for rec in plan.sell_recommendations
            if rec.account_name == name
        )
        for name in ("A", "B")
    }

    # A: all groups within band → no sell.
    # B: 국내성장 100% (AUM=200), midpoint=(100+35)/2=67.5%, safety cap at upper 40%,
    #    sell_weight=60%, sell_krw=60%*200=120.
    assert by_account["A"] == Decimal("0")
    assert by_account["B"] == Decimal("120")


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
    # A is massively overweight in 국내성장. B has no holdings.
    # Sell cash from A must stay in A; B should generate no actions.
    groups = make_standard_groups()
    stocks = make_standard_stocks(groups)
    summary = make_summary(
        groups,
        stocks,
        {
            "국내성장": Decimal("500"),
            "해외성장": Decimal("20"),
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
            "B": {},
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


def test_build_plan_only_sells_in_overheated_account() -> None:
    # A is balanced. B is overheated in 국내성장. Only B should sell.
    groups = make_standard_groups()
    stocks = make_standard_stocks(groups)

    # A: perfectly target-weighted (AUM=1000)
    # B: 국내성장=100% of AUM=200 → massively overheated
    summary = make_summary(
        groups,
        stocks,
        {
            "국내성장": Decimal("550"),
            "국내배당": Decimal("150"),
            "해외성장": Decimal("250"),
            "해외안정": Decimal("100"),
            "해외배당": Decimal("150"),
        },
    )
    accounts = [make_account("A", Decimal("0")), make_account("B", Decimal("0"))]
    holdings_by_account = make_holdings_by_account(
        accounts,
        stocks,
        {
            "A": {
                "국내성장": Decimal("350"),
                "국내배당": Decimal("150"),
                "해외성장": Decimal("250"),
                "해외안정": Decimal("100"),
                "해외배당": Decimal("150"),
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

    # A: 국내성장 35% of AUM=1000, exactly on target → no sell
    # B: 국내성장 100% of AUM=200, far above upper 40% → sell
    assert plan.sell_recommendations
    assert all(rec.account_name == "B" for rec in plan.sell_recommendations)
    for rec in plan.buy_recommendations:
        assert rec.account_name != "A", (
            f"A had no sell cash so should not buy; got {rec}"
        )


def test_build_plan_unmet_group_produces_warning_not_recommendation() -> None:
    # A holds only domestic stocks; foreign groups are lower-breached.
    # Since A has no foreign candidates, foreign groups go to unmet_groups.
    groups = make_standard_groups()
    stocks = make_standard_stocks(groups)
    account = make_account("A", Decimal("100"))  # has cash to spend
    summary = make_summary(
        groups,
        stocks,
        {
            "국내성장": Decimal("800"),
            "국내배당": Decimal("100"),
        },
    )
    holdings_by_account = make_holdings_by_account(
        [account],
        stocks,
        {
            "A": {
                "국내성장": Decimal("800"),
                "국내배당": Decimal("100"),
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

    # No foreign tickers held → no foreign buy recommendations
    foreign_buys = [
        r
        for r in plan.buy_recommendations
        if r.rebalance_group_name in ("해외성장", "해외안정", "해외배당")
    ]
    assert foreign_buys == []

    # account_summaries should flag the unmet foreign groups
    assert len(plan.account_summaries) == 1
    summary_a = plan.account_summaries[0]
    assert "해외성장" in summary_a.unmet_groups
    assert "해외배당" in summary_a.unmet_groups
    assert summary_a.unused_cash_krw > 0


def test_build_plan_leftover_cash_when_account_has_excess_sell() -> None:
    # A sells 국내성장 but can only reinvest into 해외성장 (limited need).
    # Remaining cash should appear as unused_cash_krw in account summary.
    groups = make_standard_groups()
    stocks = make_standard_stocks(groups)
    account = make_account("A", Decimal("0"))
    # Only 국내성장(550) and 해외성장(50) → 국내성장 overweight, 해외성장 tiny
    summary = make_summary(
        groups,
        stocks,
        {"국내성장": Decimal("550"), "해외성장": Decimal("50")},
    )
    holdings_by_account = make_holdings_by_account(
        [account],
        stocks,
        {"A": {"국내성장": Decimal("550"), "해외성장": Decimal("50")}},
    )

    service = RebalanceService()
    plan = service.build_plan(
        summary=summary,
        accounts=[account],
        holdings_by_account=holdings_by_account,
        groups=groups,
        stocks=list(stocks.values()),
    )

    assert plan.sell_recommendations
    assert len(plan.account_summaries) == 1
    summary_a = plan.account_summaries[0]
    # Some sell cash must remain unused (no candidates for several groups)
    assert summary_a.unused_cash_krw > 0
    assert summary_a.sell_cash_krw > 0


def test_build_plan_does_not_cross_account_for_buy_candidate() -> None:
    # Regression: cash from account A must stay in account A.
    # A sells overweight domestic stock; total buys for A must not exceed
    # A's own starting cash + A's sell proceeds (cash isolation invariant).
    groups = make_standard_groups()
    stocks = make_standard_stocks(groups)
    account_a = make_account("A", Decimal("0"))
    account_b = make_account("B", Decimal("0"))

    summary = make_summary(
        groups,
        stocks,
        {
            "국내성장": Decimal("500"),
            "해외성장": Decimal("100"),
        },
    )
    holdings_by_account = make_holdings_by_account(
        [account_a, account_b],
        stocks,
        {
            "A": {"국내성장": Decimal("500")},
            "B": {"해외성장": Decimal("100")},
        },
    )

    service = RebalanceService()
    plan = service.build_plan(
        summary=summary,
        accounts=[account_a, account_b],
        holdings_by_account=holdings_by_account,
        groups=groups,
        stocks=list(stocks.values()),
    )

    summary_a = next(s for s in plan.account_summaries if s.account_name == "A")
    summary_b = next(s for s in plan.account_summaries if s.account_name == "B")

    # Cash isolation: A's total buys ≤ A's starting cash + A's sell proceeds
    a_total_buy = sum(
        r.amount_krw if r.amount_krw is not None else r.amount
        for r in summary_a.buy_recommendations
    )
    assert a_total_buy <= account_a.cash_balance + summary_a.sell_cash_krw

    # B's cash is not spent on A's behalf
    b_total_buy = sum(
        r.amount_krw if r.amount_krw is not None else r.amount
        for r in summary_b.buy_recommendations
    )
    assert b_total_buy <= account_b.cash_balance + summary_b.sell_cash_krw


def test_build_plan_same_name_accounts_correct_attribution() -> None:
    # Regression: two accounts with the same name must not have their recs
    # cross-attributed (old name-keyed lookup would only keep one of them).
    groups = make_standard_groups()
    stocks = make_standard_stocks(groups)
    account_a = make_account("계좌", Decimal("0"))  # overweight in 국내성장
    account_b = make_account("계좌", Decimal("0"))  # same name, no holdings → no sells

    summary = make_summary(groups, stocks, {"국내성장": Decimal("600")})
    # Build holdings manually to avoid the name-collision in make_holdings_by_account
    holdings_by_account = {
        account_a.id: [
            make_holding(account_a.id, stocks["국내성장"].id, Decimal("600"))
        ],
        account_b.id: [],
    }

    service = RebalanceService()
    plan = service.build_plan(
        summary=summary,
        accounts=[account_a, account_b],
        holdings_by_account=holdings_by_account,
        groups=groups,
        stocks=list(stocks.values()),
    )

    assert len(plan.account_summaries) == 2
    s_a = next(s for s in plan.account_summaries if s.account_id == str(account_a.id))
    s_b = next(s for s in plan.account_summaries if s.account_id == str(account_b.id))

    # account_a is overweight in 국내성장 → has sell recs attributed to it
    assert s_a.sell_recommendations
    assert all(r.ticker == "005930" for r in s_a.sell_recommendations)
    # account_b has no holdings → no sell recs (old name-keyed code would give b all of a's recs)
    assert not s_b.sell_recommendations


def test_build_plan_portfolio_fallback_buys_into_new_group() -> None:
    # account_a has no holdings in 해외성장 but portfolio snapshot has QQQ.
    # With fallback, account_a should receive a buy rec for QQQ using its cash.
    groups = make_standard_groups()
    stocks = make_standard_stocks(groups)
    account_a = make_account("A", Decimal("10000"))  # cash only, no foreign holdings
    account_b = make_account("B", Decimal("0"))

    summary = make_summary(
        groups,
        stocks,
        {"국내성장": Decimal("30000"), "해외성장": Decimal("10000")},
    )
    holdings_by_account = {
        account_a.id: [
            make_holding(account_a.id, stocks["국내성장"].id, Decimal("30000"))
        ],
        account_b.id: [
            make_holding(account_b.id, stocks["해외성장"].id, Decimal("10000"))
        ],
    }

    service = RebalanceService()
    plan = service.build_plan(
        summary=summary,
        accounts=[account_a, account_b],
        holdings_by_account=holdings_by_account,
        groups=groups,
        stocks=list(stocks.values()),
    )

    # account_a has cash and 해외성장 is lower-breached; portfolio has QQQ → fallback fires
    a_buy_tickers = {
        rec.ticker for rec in plan.buy_recommendations if rec.account_name == "A"
    }
    assert "QQQ" in a_buy_tickers


def test_build_plan_account_summaries_populated() -> None:
    # Smoke: account_summaries contains one entry per account with correct names.
    groups = make_standard_groups()
    stocks = make_standard_stocks(groups)
    accounts = [make_account("Alpha", Decimal("0")), make_account("Beta", Decimal("0"))]
    summary = make_summary(
        groups,
        stocks,
        {
            "국내성장": Decimal("300"),
            "해외성장": Decimal("200"),
        },
    )
    holdings_by_account = make_holdings_by_account(
        accounts,
        stocks,
        {
            "Alpha": {"국내성장": Decimal("300")},
            "Beta": {"해외성장": Decimal("200")},
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

    assert len(plan.account_summaries) == 2
    names = {s.account_name for s in plan.account_summaries}
    assert names == {"Alpha", "Beta"}
