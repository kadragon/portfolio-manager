"""Rebalance service implementing sleeve/region based v2.0 logic."""

from __future__ import annotations

from collections import defaultdict
from dataclasses import dataclass
from decimal import Decimal
from uuid import UUID

from portfolio_manager.models import Account, Group, Holding, Stock
from portfolio_manager.models.rebalance import RebalanceAction, RebalanceRecommendation
from portfolio_manager.services.kis.kis_market_detector import is_domestic_ticker
from portfolio_manager.services.portfolio_service import PortfolioSummary

_PERCENT_BASE = Decimal("100")
_REGION_BAND = Decimal("5")

_SLEEVE_ORDER = (
    "국내성장",
    "국내배당",
    "해외성장",
    "해외안정",
    "해외배당",
)

_SLEEVE_BANDS: dict[str, Decimal] = {
    "국내성장": Decimal("5"),
    "국내배당": Decimal("3"),
    "해외성장": Decimal("5"),
    "해외안정": Decimal("2"),
    "해외배당": Decimal("3"),
}


@dataclass
class SleeveDiagnostic:
    """Current/target/band status for a sleeve."""

    sleeve_name: str
    target_percentage: Decimal
    band_percentage: Decimal
    lower_percentage: Decimal
    upper_percentage: Decimal
    current_percentage: Decimal
    current_value_krw: Decimal
    is_upper_breached: bool
    is_lower_breached: bool


@dataclass
class RegionDiagnostic:
    """Current/target status for KR/US region split."""

    target_kr_percentage: Decimal
    target_us_percentage: Decimal
    current_kr_percentage: Decimal
    current_us_percentage: Decimal
    lower_kr_percentage: Decimal
    upper_kr_percentage: Decimal
    is_triggered: bool


@dataclass
class RebalancePlan:
    """Complete rebalance output for UI and execution."""

    sell_recommendations: list[RebalanceRecommendation]
    buy_recommendations: list[RebalanceRecommendation]
    sleeve_diagnostics: list[SleeveDiagnostic]
    region_diagnostic: RegionDiagnostic
    total_assets_krw: Decimal


@dataclass
class _TickerSnapshot:
    ticker: str
    sleeve_name: str
    group_name: str
    currency: str
    stock_name: str
    total_quantity: Decimal
    total_value_local: Decimal
    total_value_krw: Decimal


@dataclass
class _AccountPosition:
    account_id: UUID
    account_name: str
    ticker: str
    sleeve_name: str
    group_name: str
    currency: str
    stock_name: str
    quantity: Decimal
    value_local: Decimal
    value_krw: Decimal


@dataclass
class _BuyCandidate:
    ticker: str
    currency: str
    stock_name: str
    group_name: str
    sleeve_name: str
    quantity_base: Decimal
    value_local_base: Decimal
    value_krw_base: Decimal


class RebalanceService:
    """Service for calculating v2.0 rebalance recommendations."""

    def build_plan(
        self,
        *,
        summary: PortfolioSummary,
        accounts: list[Account],
        holdings_by_account: dict[UUID, list[Holding]],
        groups: list[Group],
        stocks: list[Stock],
    ) -> RebalancePlan:
        total_assets = (
            summary.total_assets if summary.total_assets != 0 else summary.total_value
        )
        if total_assets <= 0:
            return RebalancePlan(
                sell_recommendations=[],
                buy_recommendations=[],
                sleeve_diagnostics=[],
                region_diagnostic=RegionDiagnostic(
                    target_kr_percentage=Decimal("0"),
                    target_us_percentage=Decimal("0"),
                    current_kr_percentage=Decimal("0"),
                    current_us_percentage=Decimal("0"),
                    lower_kr_percentage=Decimal("0"),
                    upper_kr_percentage=Decimal("0"),
                    is_triggered=False,
                ),
                total_assets_krw=Decimal("0"),
            )

        group_by_id = {group.id: group for group in groups}
        stock_by_id = {stock.id: stock for stock in stocks}
        account_by_id = {account.id: account for account in accounts}

        target_by_sleeve = self._build_target_by_sleeve(groups)
        current_by_sleeve: dict[str, Decimal] = {
            sleeve: Decimal("0") for sleeve in _SLEEVE_ORDER
        }
        ticker_snapshots = self._build_ticker_snapshots(summary)

        for snapshot in ticker_snapshots.values():
            current_by_sleeve[snapshot.sleeve_name] += snapshot.total_value_krw

        sleeve_diagnostics = self._build_sleeve_diagnostics(
            current_by_sleeve=current_by_sleeve,
            target_by_sleeve=target_by_sleeve,
            total_assets=total_assets,
        )
        sleeve_diag_by_name = {diag.sleeve_name: diag for diag in sleeve_diagnostics}

        region_diagnostic = self._build_region_diagnostic(
            current_by_sleeve=current_by_sleeve,
            target_by_sleeve=target_by_sleeve,
            total_assets=total_assets,
        )

        positions = self._build_account_positions(
            holdings_by_account=holdings_by_account,
            account_by_id=account_by_id,
            stock_by_id=stock_by_id,
            group_by_id=group_by_id,
            ticker_snapshots=ticker_snapshots,
        )

        sell_by_sleeve = self._calculate_sell_amounts_by_sleeve(
            sleeve_diagnostics=sleeve_diagnostics,
            total_assets=total_assets,
        )

        sell_recommendations, sold_by_sleeve, sell_cash_by_account = (
            self._build_sell_recommendations(
                sell_by_sleeve=sell_by_sleeve,
                positions=positions,
                sleeve_diag_by_name=sleeve_diag_by_name,
            )
        )

        buy_candidates_by_sleeve: dict[str, list[_BuyCandidate]] = defaultdict(list)
        for snapshot in ticker_snapshots.values():
            if snapshot.total_quantity <= 0 or snapshot.total_value_local <= 0:
                continue
            buy_candidates_by_sleeve[snapshot.sleeve_name].append(
                _BuyCandidate(
                    ticker=snapshot.ticker,
                    currency=snapshot.currency,
                    stock_name=snapshot.stock_name,
                    group_name=snapshot.group_name,
                    sleeve_name=snapshot.sleeve_name,
                    quantity_base=snapshot.total_quantity,
                    value_local_base=snapshot.total_value_local,
                    value_krw_base=snapshot.total_value_krw,
                )
            )

        for sleeve in buy_candidates_by_sleeve:
            buy_candidates_by_sleeve[sleeve].sort(
                key=lambda candidate: (
                    0 if is_domestic_ticker(candidate.ticker) else 1,
                    candidate.ticker,
                )
            )

        buy_recommendations = self._build_buy_recommendations(
            accounts=accounts,
            positions=positions,
            target_by_sleeve=target_by_sleeve,
            current_by_sleeve=current_by_sleeve,
            sold_by_sleeve=sold_by_sleeve,
            sell_cash_by_account=sell_cash_by_account,
            buy_candidates_by_sleeve=buy_candidates_by_sleeve,
            total_assets=total_assets,
        )

        return RebalancePlan(
            sell_recommendations=sell_recommendations,
            buy_recommendations=buy_recommendations,
            sleeve_diagnostics=sleeve_diagnostics,
            region_diagnostic=region_diagnostic,
            total_assets_krw=total_assets,
        )

    def _build_target_by_sleeve(self, groups: list[Group]) -> dict[str, Decimal]:
        target_by_sleeve: dict[str, Decimal] = {
            sleeve: Decimal("0") for sleeve in _SLEEVE_ORDER
        }
        for group in groups:
            sleeve_name = self._to_sleeve(group.name)
            if sleeve_name is None:
                continue
            target_by_sleeve[sleeve_name] += Decimal(str(group.target_percentage))
        return target_by_sleeve

    def _build_ticker_snapshots(
        self, summary: PortfolioSummary
    ) -> dict[str, _TickerSnapshot]:
        snapshots: dict[str, _TickerSnapshot] = {}

        for group, holding in summary.holdings:
            sleeve_name = self._to_sleeve(group.name)
            if sleeve_name is None:
                raise ValueError(
                    f"슬리브 매핑 불가 그룹이 있습니다: {group.name}. "
                    "그룹명은 국내성장/국내배당/해외성장/해외안정/해외배당 중 하나여야 합니다."
                )

            value_local = holding.value
            value_krw = (
                holding.value_krw if holding.value_krw is not None else value_local
            )
            stock_name = holding.name or holding.stock.ticker
            ticker = holding.stock.ticker

            existing = snapshots.get(ticker)
            if existing is None:
                snapshots[ticker] = _TickerSnapshot(
                    ticker=ticker,
                    sleeve_name=sleeve_name,
                    group_name=group.name,
                    currency=holding.currency,
                    stock_name=stock_name,
                    total_quantity=holding.quantity,
                    total_value_local=value_local,
                    total_value_krw=value_krw,
                )
                continue

            snapshots[ticker] = _TickerSnapshot(
                ticker=ticker,
                sleeve_name=existing.sleeve_name,
                group_name=existing.group_name,
                currency=existing.currency,
                stock_name=existing.stock_name,
                total_quantity=existing.total_quantity + holding.quantity,
                total_value_local=existing.total_value_local + value_local,
                total_value_krw=existing.total_value_krw + value_krw,
            )

        return snapshots

    def _build_sleeve_diagnostics(
        self,
        *,
        current_by_sleeve: dict[str, Decimal],
        target_by_sleeve: dict[str, Decimal],
        total_assets: Decimal,
    ) -> list[SleeveDiagnostic]:
        diagnostics: list[SleeveDiagnostic] = []
        for sleeve_name in _SLEEVE_ORDER:
            target = target_by_sleeve.get(sleeve_name, Decimal("0"))
            band = _SLEEVE_BANDS[sleeve_name]
            current_value = current_by_sleeve.get(sleeve_name, Decimal("0"))
            current = self._to_percent(current_value, total_assets)
            lower = target - band
            upper = target + band
            diagnostics.append(
                SleeveDiagnostic(
                    sleeve_name=sleeve_name,
                    target_percentage=target,
                    band_percentage=band,
                    lower_percentage=lower,
                    upper_percentage=upper,
                    current_percentage=current,
                    current_value_krw=current_value,
                    is_upper_breached=current > upper,
                    is_lower_breached=current < lower,
                )
            )
        return diagnostics

    def _build_region_diagnostic(
        self,
        *,
        current_by_sleeve: dict[str, Decimal],
        target_by_sleeve: dict[str, Decimal],
        total_assets: Decimal,
    ) -> RegionDiagnostic:
        target_kr = target_by_sleeve["국내성장"] + target_by_sleeve["국내배당"]
        target_us = (
            target_by_sleeve["해외성장"]
            + target_by_sleeve["해외안정"]
            + target_by_sleeve["해외배당"]
        )

        current_kr_value = current_by_sleeve["국내성장"] + current_by_sleeve["국내배당"]
        current_kr = self._to_percent(current_kr_value, total_assets)
        current_us = _PERCENT_BASE - current_kr

        lower_kr = target_kr - _REGION_BAND
        upper_kr = target_kr + _REGION_BAND

        return RegionDiagnostic(
            target_kr_percentage=target_kr,
            target_us_percentage=target_us,
            current_kr_percentage=current_kr,
            current_us_percentage=current_us,
            lower_kr_percentage=lower_kr,
            upper_kr_percentage=upper_kr,
            is_triggered=current_kr < lower_kr or current_kr > upper_kr,
        )

    def _build_account_positions(
        self,
        *,
        holdings_by_account: dict[UUID, list[Holding]],
        account_by_id: dict[UUID, Account],
        stock_by_id: dict[UUID, Stock],
        group_by_id: dict[UUID, Group],
        ticker_snapshots: dict[str, _TickerSnapshot],
    ) -> list[_AccountPosition]:
        positions: list[_AccountPosition] = []
        for account_id, holdings in holdings_by_account.items():
            account = account_by_id.get(account_id)
            if account is None:
                continue

            for holding in holdings:
                if holding.quantity <= 0:
                    continue
                stock = stock_by_id.get(holding.stock_id)
                if stock is None:
                    continue
                group = group_by_id.get(stock.group_id)
                if group is None:
                    continue
                sleeve_name = self._to_sleeve(group.name)
                if sleeve_name is None:
                    raise ValueError(
                        f"슬리브 매핑 불가 그룹이 있습니다: {group.name}. "
                        "그룹명은 국내성장/국내배당/해외성장/해외안정/해외배당 중 하나여야 합니다."
                    )

                snapshot = ticker_snapshots.get(stock.ticker)
                if snapshot is None or snapshot.total_quantity <= 0:
                    continue

                ratio = holding.quantity / snapshot.total_quantity
                value_krw = snapshot.total_value_krw * ratio
                value_local = snapshot.total_value_local * ratio
                positions.append(
                    _AccountPosition(
                        account_id=account.id,
                        account_name=account.name,
                        ticker=stock.ticker,
                        sleeve_name=sleeve_name,
                        group_name=group.name,
                        currency=snapshot.currency,
                        stock_name=snapshot.stock_name or stock.ticker,
                        quantity=holding.quantity,
                        value_local=value_local,
                        value_krw=value_krw,
                    )
                )
        return positions

    def _calculate_sell_amounts_by_sleeve(
        self,
        *,
        sleeve_diagnostics: list[SleeveDiagnostic],
        total_assets: Decimal,
    ) -> dict[str, Decimal]:
        upper_breaches = [diag for diag in sleeve_diagnostics if diag.is_upper_breached]
        if not upper_breaches:
            return {}

        upper_breaches.sort(
            key=lambda diag: diag.current_percentage - diag.target_percentage,
            reverse=True,
        )

        sell_by_sleeve: dict[str, Decimal] = {}
        for diag in upper_breaches:
            midpoint = (diag.current_percentage + diag.target_percentage) / Decimal("2")
            next_weight = min(midpoint, diag.upper_percentage)
            sell_weight = diag.current_percentage - next_weight
            if sell_weight <= 0:
                continue
            sell_by_sleeve[diag.sleeve_name] = (
                sell_weight / _PERCENT_BASE
            ) * total_assets
        return sell_by_sleeve

    def _build_sell_recommendations(
        self,
        *,
        sell_by_sleeve: dict[str, Decimal],
        positions: list[_AccountPosition],
        sleeve_diag_by_name: dict[str, SleeveDiagnostic],
    ) -> tuple[list[RebalanceRecommendation], dict[str, Decimal], dict[UUID, Decimal]]:
        recommendations: list[RebalanceRecommendation] = []
        sold_by_sleeve: dict[str, Decimal] = defaultdict(Decimal)
        sell_cash_by_account: dict[UUID, Decimal] = defaultdict(Decimal)

        for sleeve_name in _SLEEVE_ORDER:
            target_sell = sell_by_sleeve.get(sleeve_name, Decimal("0"))
            if target_sell <= 0:
                continue

            sleeve_positions = [
                position
                for position in positions
                if position.sleeve_name == sleeve_name and position.value_krw > 0
            ]
            if not sleeve_positions:
                continue

            total_sleeve_value = sum(
                (position.value_krw for position in sleeve_positions), Decimal("0")
            )
            if total_sleeve_value <= 0:
                continue

            target_sell = min(target_sell, total_sleeve_value)
            account_values: dict[UUID, Decimal] = defaultdict(Decimal)
            for position in sleeve_positions:
                account_values[position.account_id] += position.value_krw

            account_ids = sorted(
                account_values.keys(),
                key=lambda account_id: next(
                    pos.account_name
                    for pos in sleeve_positions
                    if pos.account_id == account_id
                ),
            )

            remaining_sleeve_sell = target_sell
            for account_index, account_id in enumerate(account_ids):
                account_value = account_values[account_id]
                if account_value <= 0:
                    continue

                if account_index == len(account_ids) - 1:
                    account_sell = min(remaining_sleeve_sell, account_value)
                else:
                    proportional = target_sell * (account_value / total_sleeve_value)
                    account_sell = min(
                        proportional, remaining_sleeve_sell, account_value
                    )

                if account_sell <= 0:
                    continue

                account_positions = [
                    position
                    for position in sleeve_positions
                    if position.account_id == account_id
                ]
                account_positions.sort(
                    key=lambda position: (
                        1
                        if is_domestic_ticker(position.ticker)
                        else 0,  # overseas (USD) first
                        position.ticker,
                    )
                )
                account_total_value = sum(
                    (position.value_krw for position in account_positions), Decimal("0")
                )
                remaining_account_sell = account_sell
                for position_index, position in enumerate(account_positions):
                    if remaining_account_sell <= 0:
                        break
                    if account_total_value <= 0:
                        break

                    if position_index == len(account_positions) - 1:
                        sell_krw = min(remaining_account_sell, position.value_krw)
                    else:
                        proportional = account_sell * (
                            position.value_krw / account_total_value
                        )
                        sell_krw = min(
                            proportional, remaining_account_sell, position.value_krw
                        )

                    account_total_value -= position.value_krw
                    if sell_krw <= 0:
                        continue

                    amount_local = self._krw_to_local_amount(
                        sell_krw,
                        position.currency,
                        position.value_local,
                        position.value_krw,
                    )
                    quantity = self._calculate_quantity(
                        amount_local=amount_local,
                        position_local_value=position.value_local,
                        position_quantity=position.quantity,
                    )

                    diag = sleeve_diag_by_name[sleeve_name]
                    recommendations.append(
                        RebalanceRecommendation(
                            ticker=position.ticker,
                            action=RebalanceAction.SELL,
                            amount=amount_local,
                            priority=len(recommendations) + 1,
                            currency=position.currency,
                            quantity=quantity,
                            stock_name=position.stock_name or position.ticker,
                            group_name=position.group_name,
                            account_name=position.account_name,
                            sleeve_name=sleeve_name,
                            reason=(
                                "과열 슬리브 절반 감축 "
                                f"({diag.current_percentage:.2f}% -> 목표근접, "
                                f"상단 {diag.upper_percentage:.2f}%)"
                            ),
                            trigger_type="sleeve",
                            amount_krw=sell_krw,
                            amount_local=amount_local,
                        )
                    )

                    remaining_account_sell -= sell_krw
                    remaining_sleeve_sell -= sell_krw
                    sold_by_sleeve[sleeve_name] += sell_krw
                    sell_cash_by_account[position.account_id] += sell_krw

        return recommendations, dict(sold_by_sleeve), dict(sell_cash_by_account)

    def _build_buy_recommendations(
        self,
        *,
        accounts: list[Account],
        positions: list[_AccountPosition],
        target_by_sleeve: dict[str, Decimal],
        current_by_sleeve: dict[str, Decimal],
        sold_by_sleeve: dict[str, Decimal],
        sell_cash_by_account: dict[UUID, Decimal],
        buy_candidates_by_sleeve: dict[str, list[_BuyCandidate]],
        total_assets: Decimal,
    ) -> list[RebalanceRecommendation]:
        recommendations: list[RebalanceRecommendation] = []

        projected_by_sleeve: dict[str, Decimal] = {}
        need_by_sleeve: dict[str, Decimal] = {}
        for sleeve_name in _SLEEVE_ORDER:
            projected_value = current_by_sleeve.get(
                sleeve_name, Decimal("0")
            ) - sold_by_sleeve.get(sleeve_name, Decimal("0"))
            target_value = (
                target_by_sleeve.get(sleeve_name, Decimal("0")) / _PERCENT_BASE
            ) * total_assets
            projected_by_sleeve[sleeve_name] = projected_value
            need_by_sleeve[sleeve_name] = max(
                Decimal("0"), target_value - projected_value
            )

        for account in accounts:
            cash = account.cash_balance + sell_cash_by_account.get(
                account.id, Decimal("0")
            )
            if cash <= 0:
                continue

            blocked_sleeves: set[str] = set()
            while cash > 0:
                sleeve_name = self._pick_next_sleeve(need_by_sleeve, blocked_sleeves)
                if sleeve_name is None:
                    break

                need = need_by_sleeve[sleeve_name]
                if need <= 0:
                    blocked_sleeves.add(sleeve_name)
                    continue

                candidate = self._select_buy_candidate(
                    account_id=account.id,
                    sleeve_name=sleeve_name,
                    positions=positions,
                    buy_candidates_by_sleeve=buy_candidates_by_sleeve,
                )
                if candidate is None:
                    blocked_sleeves.add(sleeve_name)
                    continue

                buy_krw = min(cash, need)
                if buy_krw <= 0:
                    break

                amount_local = self._krw_to_local_amount(
                    buy_krw,
                    candidate.currency,
                    candidate.value_local_base,
                    candidate.value_krw_base,
                )
                quantity = self._calculate_quantity(
                    amount_local=amount_local,
                    position_local_value=candidate.value_local_base,
                    position_quantity=candidate.quantity_base,
                )

                projected_weight = self._to_percent(
                    projected_by_sleeve[sleeve_name], total_assets
                )
                recommendations.append(
                    RebalanceRecommendation(
                        ticker=candidate.ticker,
                        action=RebalanceAction.BUY,
                        amount=amount_local,
                        priority=len(recommendations) + 1,
                        currency=candidate.currency,
                        quantity=quantity,
                        stock_name=candidate.stock_name or candidate.ticker,
                        group_name=candidate.group_name,
                        account_name=account.name,
                        sleeve_name=sleeve_name,
                        reason=(
                            "목표 대비 부족 수분 공급 "
                            f"({projected_weight:.2f}% -> 목표 {target_by_sleeve[sleeve_name]:.2f}%)"
                        ),
                        trigger_type="sleeve",
                        amount_krw=buy_krw,
                        amount_local=amount_local,
                    )
                )

                cash -= buy_krw
                projected_by_sleeve[sleeve_name] += buy_krw
                need_by_sleeve[sleeve_name] = max(
                    Decimal("0"), need_by_sleeve[sleeve_name] - buy_krw
                )

        return recommendations

    def _pick_next_sleeve(
        self,
        need_by_sleeve: dict[str, Decimal],
        blocked_sleeves: set[str],
    ) -> str | None:
        candidates = [
            sleeve_name
            for sleeve_name in _SLEEVE_ORDER
            if sleeve_name not in blocked_sleeves
            and need_by_sleeve.get(sleeve_name, Decimal("0")) > 0
        ]
        if not candidates:
            return None
        candidates.sort(
            key=lambda sleeve_name: (
                need_by_sleeve.get(sleeve_name, Decimal("0")),
                # tiebreak: higher index in _SLEEVE_ORDER wins (해외 sleeves last → buy
                # US positions only after KR needs are met), negated for reverse sort
                -_SLEEVE_ORDER.index(sleeve_name),
            ),
            reverse=True,
        )
        return candidates[0]

    def _select_buy_candidate(
        self,
        *,
        account_id: UUID,
        sleeve_name: str,
        positions: list[_AccountPosition],
        buy_candidates_by_sleeve: dict[str, list[_BuyCandidate]],
    ) -> _BuyCandidate | None:
        account_positions = [
            position
            for position in positions
            if position.account_id == account_id
            and position.sleeve_name == sleeve_name
            and position.value_local > 0
        ]
        if account_positions:
            account_positions.sort(
                key=lambda position: (
                    0 if is_domestic_ticker(position.ticker) else 1,
                    -position.value_krw,
                    position.ticker,
                )
            )
            top = account_positions[0]
            return _BuyCandidate(
                ticker=top.ticker,
                currency=top.currency,
                stock_name=top.stock_name,
                group_name=top.group_name,
                sleeve_name=top.sleeve_name,
                quantity_base=top.quantity,
                value_local_base=top.value_local,
                value_krw_base=top.value_krw,
            )

        candidates = buy_candidates_by_sleeve.get(sleeve_name, [])
        if not candidates:
            return None
        return candidates[0]

    @staticmethod
    def _krw_to_local_amount(
        amount_krw: Decimal,
        currency: str,
        value_local: Decimal,
        value_krw: Decimal,
    ) -> Decimal:
        if currency == "USD" and value_local > 0 and value_krw > 0:
            fx = value_krw / value_local
            if fx > 0:
                return amount_krw / fx
        return amount_krw

    def _calculate_quantity(
        self,
        *,
        amount_local: Decimal,
        position_local_value: Decimal,
        position_quantity: Decimal,
    ) -> Decimal | None:
        if position_local_value <= 0 or position_quantity <= 0:
            return None
        return (amount_local / position_local_value) * position_quantity

    def _to_percent(self, value: Decimal, total_assets: Decimal) -> Decimal:
        if total_assets == 0:
            return Decimal("0")
        return (value / total_assets) * _PERCENT_BASE

    def _to_sleeve(self, group_name: str) -> str | None:
        normalized = "".join(group_name.split())
        return normalized if normalized in _SLEEVE_BANDS else None
