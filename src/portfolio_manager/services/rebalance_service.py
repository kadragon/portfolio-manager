"""Rebalance service for generating stock recommendations."""

from collections.abc import Mapping
from dataclasses import dataclass
from decimal import Decimal
from uuid import UUID

from portfolio_manager.models import Group
from portfolio_manager.models.rebalance import (
    GroupRebalanceAction,
    GroupRebalanceSignal,
)
from portfolio_manager.services.portfolio_service import PortfolioSummary

# Tolerance band thresholds for group-level rebalancing
_BUY_THRESHOLD = Decimal("-2")  # Groups at target -2% or lower trigger BUY
_DEFAULT_SELL_THRESHOLD = Decimal("4")  # Default: target +4% to consider sell
_GROWTH_SELL_THRESHOLD = Decimal("6")  # Growth assets: wider band at +6%
_RETURN_THRESHOLD = Decimal("20")  # Minimum return since rebalance to consider sell
# Rebalance action reasons
REASON_SELL_GATES_NOT_MET = "Sell gates not met"
REASON_WITHIN_TOLERANCE = "Within tolerance band"
REASON_NO_PORTFOLIO_VALUE = "No portfolio value"


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
    ) -> tuple[GroupRebalanceAction, bool, str | None]:
        """Determine the rebalance action for a group based on delta and metrics.

        Returns:
            Tuple of (action, manual_review_required, reason)
        """
        if delta <= _BUY_THRESHOLD:
            return GroupRebalanceAction.BUY, False, None

        if delta >= metrics.sell_threshold:
            if metrics.meets_sell_gates():
                return GroupRebalanceAction.SELL_CANDIDATE, True, None
            return GroupRebalanceAction.NO_ACTION, False, REASON_SELL_GATES_NOT_MET

        return GroupRebalanceAction.NO_ACTION, False, REASON_WITHIN_TOLERANCE

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
                        reason=REASON_NO_PORTFOLIO_VALUE,
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
            action, manual_review_required, reason = self._determine_group_action(
                delta, metrics
            )

            actions.append(
                GroupRebalanceSignal(
                    group=diff.group,
                    action=action,
                    delta=delta,
                    manual_review_required=manual_review_required,
                    reason=reason,
                )
            )

        return actions
