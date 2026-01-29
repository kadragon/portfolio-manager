"""Rebalance service for generating stock recommendations."""

from collections.abc import Mapping
from dataclasses import dataclass
from decimal import Decimal
from uuid import UUID

from portfolio_manager.models import Group
from portfolio_manager.models.rebalance import (
    GroupRebalanceAction,
    GroupRebalanceSignal,
    RebalanceAction,
    RebalanceRecommendation,
)
from portfolio_manager.services.portfolio_service import (
    PortfolioSummary,
    StockHoldingWithPrice,
)

# Tolerance band thresholds for group-level rebalancing
_BUY_THRESHOLD = Decimal("-2")  # Groups at target -2% or lower trigger BUY
_DEFAULT_SELL_THRESHOLD = Decimal("4")  # Default: target +4% to consider sell
_GROWTH_SELL_THRESHOLD = Decimal("6")  # Growth assets: wider band at +6%
_RETURN_THRESHOLD = Decimal("20")  # Minimum return since rebalance to consider sell


@dataclass
class GroupMetrics:
    """Extracted and validated metrics for a group."""

    return_since_rebalance: Decimal | None
    momentum_6m: Decimal | None
    asset_class: str | None

    @property
    def is_dividend(self) -> bool:
        """Check if this is a dividend asset class."""
        return self.asset_class == "dividend"

    @property
    def sell_threshold(self) -> Decimal:
        """Get the appropriate sell threshold based on asset class."""
        if self.asset_class == "growth":
            return _GROWTH_SELL_THRESHOLD
        return _DEFAULT_SELL_THRESHOLD

    def meets_sell_gates(self) -> bool:
        """Check if all sell gate conditions are met."""
        return (
            self.return_since_rebalance is not None
            and self.momentum_6m is not None
            and self.return_since_rebalance >= _RETURN_THRESHOLD
            and self.momentum_6m < Decimal("0")
            and not self.is_dividend
        )


@dataclass
class GroupDifference:
    """Difference between current and target allocation for a group."""

    group: Group
    current_value: Decimal
    target_value: Decimal
    difference: Decimal  # positive = overweight, negative = underweight


class RebalanceService:
    """Service for calculating rebalancing recommendations."""

    def _calculate_quantity(
        self, amount: Decimal, holding: StockHoldingWithPrice
    ) -> Decimal | None:
        if holding.quantity <= 0:
            return None

        total_value = (
            holding.value_krw if holding.value_krw is not None else holding.value
        )
        if total_value == 0:
            return None

        return (amount / total_value) * holding.quantity

    def _extract_group_metrics(
        self,
        group_id: UUID,
        group_metrics: Mapping[UUID, Mapping[str, Decimal | str]] | None,
    ) -> GroupMetrics:
        """Extract and validate metrics for a single group."""
        if group_metrics is None:
            return GroupMetrics(
                return_since_rebalance=None, momentum_6m=None, asset_class=None
            )

        metrics = group_metrics.get(group_id)
        if metrics is None:
            return GroupMetrics(
                return_since_rebalance=None, momentum_6m=None, asset_class=None
            )

        return_since_rebalance = metrics.get("return_since_rebalance")
        momentum_6m = metrics.get("momentum_6m")
        asset_class = metrics.get("asset_class")

        # Validate types
        if not isinstance(return_since_rebalance, Decimal):
            return_since_rebalance = None
        if not isinstance(momentum_6m, Decimal):
            momentum_6m = None
        if not isinstance(asset_class, str):
            asset_class = None

        return GroupMetrics(
            return_since_rebalance=return_since_rebalance,
            momentum_6m=momentum_6m,
            asset_class=asset_class,
        )

    def _determine_group_action(
        self, delta: Decimal, metrics: GroupMetrics
    ) -> tuple[GroupRebalanceAction, bool]:
        """Determine the rebalance action for a group based on delta and metrics.

        Returns:
            Tuple of (action, manual_review_required)
        """
        if delta <= _BUY_THRESHOLD:
            return GroupRebalanceAction.BUY, False

        if delta >= metrics.sell_threshold and metrics.meets_sell_gates():
            return GroupRebalanceAction.SELL_CANDIDATE, True

        return GroupRebalanceAction.NO_ACTION, False

    def calculate_group_differences(
        self, summary: PortfolioSummary
    ) -> list[GroupDifference]:
        """Calculate the difference between current and target for each group."""
        total_value = summary.total_value

        # Aggregate holdings by group
        group_values: dict[str, tuple[Group, Decimal]] = {}

        for group, holding in summary.holdings:
            group_key = str(group.id)
            value = holding.value_krw if holding.value_krw else holding.value

            if group_key in group_values:
                existing_group, existing_value = group_values[group_key]
                group_values[group_key] = (existing_group, existing_value + value)
            else:
                group_values[group_key] = (group, value)

        # Calculate differences
        differences: list[GroupDifference] = []

        for _group_key, (group, current_value) in group_values.items():
            target_percentage = Decimal(str(group.target_percentage))
            target_value = (total_value * target_percentage) / Decimal("100")
            difference = current_value - target_value

            differences.append(
                GroupDifference(
                    group=group,
                    current_value=current_value,
                    target_value=target_value,
                    difference=difference,
                )
            )

        return differences

    def get_group_actions_v2(
        self,
        summary: PortfolioSummary,
        group_metrics: Mapping[UUID, Mapping[str, Decimal | str]] | None = None,
    ) -> list[GroupRebalanceSignal]:
        """Get group-level rebalancing signals using v2 tolerance bands."""
        differences = self.calculate_group_differences(summary)
        total_value = summary.total_value
        actions: list[GroupRebalanceSignal] = []

        for diff in differences:
            if total_value == 0:
                # No meaningful rebalancing with zero portfolio value
                actions.append(
                    GroupRebalanceSignal(
                        group=diff.group,
                        action=GroupRebalanceAction.NO_ACTION,
                        delta=Decimal("0"),
                        manual_review_required=False,
                    )
                )
                continue

            # Calculate weight delta
            current_weight = (diff.current_value / total_value) * Decimal("100")
            target_percentage = Decimal(str(diff.group.target_percentage))
            delta = current_weight - target_percentage

            # Extract and validate metrics
            metrics = self._extract_group_metrics(diff.group.id, group_metrics)

            # Determine action based on delta and metrics
            action, manual_review_required = self._determine_group_action(
                delta, metrics
            )

            actions.append(
                GroupRebalanceSignal(
                    group=diff.group,
                    action=action,
                    delta=delta,
                    manual_review_required=manual_review_required,
                )
            )

        return actions

    def get_sell_recommendations(
        self, summary: PortfolioSummary
    ) -> list[RebalanceRecommendation]:
        """Get sell recommendations for overweight groups, overseas stocks first."""
        differences = self.calculate_group_differences(summary)

        recommendations: list[RebalanceRecommendation] = []

        for diff in differences:
            if diff.difference <= 0:
                # Not overweight, no sell needed
                continue

            amount_to_sell = diff.difference

            # Get holdings for this group
            group_holdings: list[tuple[StockHoldingWithPrice, Decimal]] = []
            for group, holding in summary.holdings:
                if str(group.id) == str(diff.group.id):
                    value = holding.value_krw if holding.value_krw else holding.value
                    group_holdings.append((holding, value))

            # Sort: overseas (USD) first, then domestic (KRW)
            group_holdings.sort(key=lambda x: (0 if x[0].currency == "USD" else 1))

            remaining = amount_to_sell
            priority = 1

            for holding, value in group_holdings:
                if remaining <= 0:
                    break

                sell_amount = min(value, remaining)
                recommendations.append(
                    RebalanceRecommendation(
                        ticker=holding.stock.ticker,
                        action=RebalanceAction.SELL,
                        amount=sell_amount,
                        priority=priority,
                        currency=holding.currency,
                        quantity=self._calculate_quantity(sell_amount, holding),
                        stock_name=holding.name or holding.stock.ticker,
                        group_name=diff.group.name,
                    )
                )
                remaining -= sell_amount
                priority += 1

        return recommendations

    def get_buy_recommendations(
        self, summary: PortfolioSummary
    ) -> list[RebalanceRecommendation]:
        """Get buy recommendations for underweight groups, domestic stocks first."""
        differences = self.calculate_group_differences(summary)

        recommendations: list[RebalanceRecommendation] = []

        for diff in differences:
            if diff.difference >= 0:
                # Not underweight, no buy needed
                continue

            amount_to_buy = abs(diff.difference)

            # Get holdings for this group
            group_holdings: list[tuple[StockHoldingWithPrice, Decimal]] = []
            for group, holding in summary.holdings:
                if str(group.id) == str(diff.group.id):
                    value = holding.value_krw if holding.value_krw else holding.value
                    group_holdings.append((holding, value))

            # Sort: domestic (KRW) first, then overseas (USD)
            group_holdings.sort(key=lambda x: (0 if x[0].currency == "KRW" else 1))

            priority = 1

            for holding, _value in group_holdings:
                recommendations.append(
                    RebalanceRecommendation(
                        ticker=holding.stock.ticker,
                        action=RebalanceAction.BUY,
                        amount=amount_to_buy,
                        priority=priority,
                        currency=holding.currency,
                        quantity=self._calculate_quantity(amount_to_buy, holding),
                        stock_name=holding.name or holding.stock.ticker,
                        group_name=diff.group.name,
                    )
                )
                priority += 1
                # For buy, just recommend the first stock (domestic priority)
                break

        return recommendations
