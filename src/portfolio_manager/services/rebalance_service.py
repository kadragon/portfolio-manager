"""Rebalance service for generating stock recommendations."""

from dataclasses import dataclass
from decimal import Decimal

from portfolio_manager.models import Group
from portfolio_manager.models.rebalance import RebalanceAction, RebalanceRecommendation
from portfolio_manager.services.portfolio_service import (
    PortfolioSummary,
    StockHoldingWithPrice,
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

    def calculate_group_differences(
        self, summary: PortfolioSummary
    ) -> list[GroupDifference]:
        """Calculate the difference between current and target for each group."""
        total_value = (
            summary.total_assets if summary.total_assets > 0 else summary.total_value
        )

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
            group_holdings.sort(key=lambda x: 0 if x[0].currency == "USD" else 1)

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
            group_holdings.sort(key=lambda x: 0 if x[0].currency == "KRW" else 1)

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
